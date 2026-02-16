package telegram_bot

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/channel_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_channel_role"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/logger"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/channels"
	"github.com/wickedv43/TAM-backend/internal/modules/storage"
	"github.com/wickedv43/TAM-backend/internal/modules/ton"

	"github.com/pkg/errors"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type Bot struct {
	tg       *tele.Bot
	web      *tele.WebApp
	resty    *resty.Client
	messages *messages.Messages

	log *zap.SugaredLogger
	cfg *config.Config

	states    storage.BotStateInterface
	customers storage.CustomersInterface
	deals     storage.DealsInterface
	channels  storage.ChannelsInterface

	bridge *channels.Channels
	ton    ton.WalletInterface

	albumTimers sync.Map
	userMu      sync.Map

	rootCtx context.Context
}

func updateType(u *tele.Update) string {
	switch {
	case u.Message != nil:
		return "message"
	case u.EditedMessage != nil:
		return "edited_message"
	case u.ChannelPost != nil:
		return "channel_post"
	case u.EditedChannelPost != nil:
		return "edited_channel_post"
	case u.Callback != nil:
		return "callback_query"
	case u.MyChatMember != nil:
		return "my_chat_member"
	default:
		return "other"
	}
}

// NewBot creates a Telegram bot, configures middleware, and sets up command handlers.
func NewBot(i do.Injector) (*Bot, error) {
	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*logger.Logger](i).Named("telegram_bot")

	lp := &tele.LongPoller{
		Timeout:        10 * time.Second,
		AllowedUpdates: tele.AllowedUpdates,
	}
	poller := tele.NewMiddlewarePoller(lp, func(u *tele.Update) bool {
		typ := updateType(u)
		if typ == "edited_channel_post" || typ == "channel_post" {
			log.Infof("poller: update_id=%d type=%s", u.ID, typ)
		}
		return true
	})
	pref := tele.Settings{
		Token:   cfg.Bot.Token,
		Poller:  poller,
		OnError: onError,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, errors.Wrap(err, "init bot")
	}

	bot := &Bot{
		tg: b,
		web: &tele.WebApp{
			URL: cfg.Bot.WebAppURL,
		},

		resty:    resty.New(),
		messages: do.MustInvoke[*messages.Messages](i),

		log: log,
		cfg: cfg,

		states:    do.MustInvoke[*storage.PostgresDB](i),
		customers: do.MustInvoke[*storage.PostgresDB](i),
		deals:     do.MustInvoke[*storage.PostgresDB](i),
		channels:  do.MustInvoke[*storage.PostgresDB](i),

		bridge: do.MustInvoke[*channels.Channels](i),
		ton:    do.MustInvoke[*ton.Ton](i),
	}

	bot.cfg.Bot.ID = bot.tg.Me.ID
	bot.cfg.Bot.Username = "@" + bot.tg.Me.Username

	bot.rootCtx, err = do.InvokeNamed[context.Context](i, "root.context")
	if err != nil {
		return nil, errors.Wrap(err, "invoke root.context")
	}

	b.Use(middleware.Recover())
	b.Use(middleware.AutoRespond())
	b.Use(bot.Logger())

	bot.HandleWithEndpoint("/start", bot.onStart)
	bot.HandleWithEndpoint("/add", bot.testAdd)
	bot.HandleWithEndpoint("/top", bot.topUp)
	bot.HandleWithEndpoint("/pub", bot.pubNow)

	bot.HandleWithEndpoint(answerMessage, bot.handleAnswer)
	bot.HandleWithEndpoint(cancelMessage, bot.handleCancel)
	bot.HandleWithEndpoint(tele.OnText, bot.onText)

	bot.HandleWithEndpoint(acceptPost, bot.acceptPost)

	bot.HandleWithEndpoint(tele.OnMedia, bot.onMedia)

	bot.HandleWithEndpoint(tele.OnMyChatMember, bot.onMyChatMember)
	bot.HandleWithEndpoint(tele.OnChannelPost, bot.onChannelPost)

	return bot, nil
}

func (b *Bot) Start(ctx context.Context) {
	if err := b.tg.RemoveWebhook(); err != nil {
		b.log.Warnf("RemoveWebhook: %v (ignore if bot was never on webhook)", err)
	} else {
		b.log.Info("webhook removed, using long poll")
	}

	go func() {
		<-ctx.Done()

		b.log.Infoln("bot stopped...")
		b.tg.Stop()
	}()

	go b.listenChannels(ctx)
	go b.startPostMonitoring(ctx)

	b.log.Infof("bot ID:%d started", b.cfg.Bot.ID)
	b.log.Infof("admins ID: %d", b.cfg.Bot.AdminsID)
	b.tg.Start()
}

// todo: remove below
func (b *Bot) testAdd(c tele.Context) error {
	cust, err := b.customers.GetCustomerByTgID(b.rootCtx, c.Sender().ID)
	if err != nil {
		return errors.Wrap(err, "get customer")
	}

	stats := &common.ChannelStats{
		Graphs:    common.ChannelGraphs{},
		Languages: []common.LanguageStat{{Name: "en", Total: 100, Ratio: 1.0}},
		Posts: &common.PostsStats{
			DateFrom:        time.Now().AddDate(0, -1, 0),
			DateTo:          time.Now(),
			Count:           50,
			AvgViews:        500,
			MedianViews:     450,
			AvgForwards:     10,
			MedianForwards:  5,
			AvgReactions:    20,
			MedianReactions: 15,
		},
		Broadcast: &common.BroadcastStats{
			Period: common.BroadcastPeriod{
				MinDate: time.Now().AddDate(0, -1, 0),
				MaxDate: time.Now(),
			},
			Followers:            common.BroadcastValue{Current: 1000, Previous: 950},
			ViewsPerPost:         common.BroadcastValue{Current: 500, Previous: 480},
			SharesPerPost:        common.BroadcastValue{Current: 10, Previous: 8},
			ReactionsPerPost:     common.BroadcastValue{Current: 20, Previous: 18},
			ViewsPerStory:        common.BroadcastValue{Current: 0, Previous: 0},
			SharesPerStory:       common.BroadcastValue{Current: 0, Previous: 0},
			ReactionsPerStory:    common.BroadcastValue{Current: 0, Previous: 0},
			EnabledNotifications: common.BroadcastValue{Current: 0, Previous: 0},
		},
	}

	channel := &ent.Channel{
		TgID:               -1003716300270,
		TgUsername:         "vcbmxniuhrs",
		TgName:             "TestAds",
		TgDescription:      "Channel for manual testing",
		Status:             int(channel_status.AWAITING_DATA),
		IsListed:           true,
		Subscribers:        1000,
		PremiumSubscribers: 50,
		MedianPostViews:    450,
		AvgPostViews:       500,
		TotalPosts:         100,
		Tags:               []string{"1"},
		Stats:              common.ChannelStatsToMap(stats),
		Prices:             map[string]float64{"post_1_24": 1},
	}

	ch, err := b.channels.CreateChannel(b.rootCtx, channel)
	if err != nil {
		b.log.Errorf("failed to create channel: %v", err)
		return errors.Wrap(err, "create channel")
	}

	_, err = b.customers.AddCustomerChannel(b.rootCtx, cust.ID, ch.ID, int(customer_channel_role.ADMIN), nil)

	return c.Send("success")
}

func (b *Bot) topUp(c tele.Context) error {
	cust, err := b.customers.GetCustomerByTgID(b.rootCtx, c.Sender().ID)
	if err != nil {
		return errors.Wrap(err, "get customer")
	}
	_, err = b.customers.UpdateCustomer(b.rootCtx, cust.ID, func(u *ent.CustomerUpdateOne) {
		u.AddTonBalance(9999)
	})

	return c.Send("success")
}

func (b *Bot) pubNow(c tele.Context) error {
	ctx, cancel := context.WithTimeout(b.rootCtx, 10*time.Second)
	defer cancel()

	text := strings.TrimSpace(c.Message().Text)
	text = strings.TrimPrefix(text, "/pubnow")
	text = strings.TrimSpace(text)

	dID, err := uuid.Parse(text)
	if err != nil {
		return c.Reply("Invalid deal ID. Usage: /pubnow <deal-uuid>")
	}

	_, err = b.deals.GetDeal(ctx, dID)
	if err != nil {
		b.log.Errorf("pubNow: get deal %s: %v", dID, err)
		return c.Reply("Deal not found")
	}

	now := time.Now()
	_, err = b.deals.UpdateDeal(ctx, dID, func(u *ent.DealUpdateOne) {
		u.SetPublicationTime(now)
		u.SetStatus(int(deal_status.AWAITING_PUBLICATION))
	})
	if err != nil {
		b.log.Errorf("pubNow: update deal %s: %v", dID, err)
		return c.Reply("Failed to update deal")
	}

	select {
	case b.bridge.AutoPost <- dID:
		return c.Reply("Publication time set to now. Post scheduled.")
	default:
		return c.Reply("AutoPost queue full, try again in a moment.")
	}
}
