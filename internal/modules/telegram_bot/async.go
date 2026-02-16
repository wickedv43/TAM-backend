package telegram_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/bot_state"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	tele "gopkg.in/telebot.v4"
)

const (
	senderID    = "sender_id"
	recipientID = "recipient_id"
)

func (b *Bot) listenChannels(ctx context.Context) {
	b.log.Info("bot waiting requests...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				b.log.Infoln("bridge closed")
				return

			case req, ok := <-b.bridge.Dialog:
				if !ok {
					b.log.Warn("channel dialog closed")
					return
				}
				b.handleDialog(req)

			case req, ok := <-b.bridge.Notification:
				if !ok {
					b.log.Warn("channel notify closed")
					return
				}
				b.handleNotification(req)

			case req, ok := <-b.bridge.Post:
				if !ok {
					b.log.Warn("channel dialog closed")
					return
				}
				b.handlePost(req)

			case req, ok := <-b.bridge.AdminMessage:
				if !ok {
					b.log.Warn("admin message closed")
				}
				b.handleAdmin(req)

			case dealID, ok := <-b.bridge.AutoPost:
				if !ok {
					b.log.Warn("autopost channel closed")
					return
				}
				b.handleAutoPost(dealID)

			case req, ok := <-b.bridge.Deal:
				if !ok {
					b.log.Warn("deal channel closed")
					return
				}
				b.handleDeal(req)
			}
		}
	}()
}

func (b *Bot) handleDialog(req common.DealMessageBotRequest) {
	ctx, cancel := context.WithTimeout(b.rootCtx, 15*time.Second)
	defer cancel()

	sender, err := b.customers.GetCustomer(ctx, req.SenderID)
	if err != nil {
		b.log.Errorln(err)
		return
	}

	recipient, err := b.customers.GetCustomer(ctx, req.RecipientID)
	if err != nil {
		b.log.Errorln(err)
		return
	}

	data := map[string]interface{}{
		senderID:    sender.TgID,
		recipientID: recipient.TgID,
	}

	_, err = b.states.SetState(sender.TgID, bot_state.AWAITING_DIALOG_MESSAGE, req.DealID, data)
	if err != nil {
		b.log.Errorln(err)
		return
	}

	deal, err := b.deals.GetDeal(ctx, req.DealID)
	if err != nil {
		b.log.Errorf("handleDialog: get deal: %v", err)
		deal = nil
	}

	locale := resolveLocale(sender.TgLanguage)

	channelName := ""
	if deal != nil && deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
		channelName = "@" + deal.Edges.Channel.TgUsername
	} else if deal != nil {
		channelName = "—"
	}

	params := map[string]interface{}{
		"DealID":      util.EscapeMarkdownV2(req.DealID.String()),
		"ChannelName": util.EscapeMarkdownV2(channelName),
	}
	msg, err := b.messages.Render(locale, "dialog", "send_message", params)
	if err != nil {
		b.log.Errorf("render send_message: %v", err)
		return
	}

	cancelLabel := b.messages.RenderOrDefault(locale, "buttons", "cancel", nil, "Cancel")
	menu := &tele.ReplyMarkup{}
	btnCancel := menu.Data(cancelLabel, cancelMessage.Unique())
	menu.Inline(
		menu.Row(btnCancel),
	)

	_, err = b.tg.Send(&tele.User{ID: sender.TgID}, msg, menu, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
	if err != nil {
		b.log.Errorln(err)
	}

}

func (b *Bot) handlePost(req common.SendDealDataBotRequest) {
	ctx, cancel := context.WithTimeout(b.rootCtx, 1*time.Minute)
	defer cancel()

	advertiserCustomer, err := b.customers.GetCustomer(ctx, req.UserID)
	if err != nil {
		b.log.Errorln(err)
		return
	}

	deal, err := b.deals.GetDeal(ctx, req.DealID)
	if err != nil {
		b.log.Errorf("Failed to get deal: %v", err)
		return
	}

	data := map[string]interface{}{}

	b.log.Debugf("SET Customer: %d STATE: %s", advertiserCustomer.TgID, bot_state.AWAITING_POST_CONTENT.String())
	_, err = b.states.SetState(advertiserCustomer.TgID, bot_state.AWAITING_POST_CONTENT, req.DealID, data)
	if err != nil {
		b.log.Errorln(err)
		return
	}

	locale := resolveLocale(advertiserCustomer.TgLanguage)
	channelName := deal.Edges.Channel.TgName
	if channelName == "" {
		channelName = deal.Edges.Channel.TgUsername
	}

	// Escape parameters manually for formatted message
	msg, err := b.messages.Render(locale, "dialog", "send_post", map[string]interface{}{
		"DealID":      util.EscapeMarkdownV2(deal.ID.String()),
		"ChannelName": util.EscapeMarkdownV2(channelName),
	})
	if err != nil {
		b.log.Errorf("render send_post: %v", err)
		return
	}

	_, err = b.tg.Send(&tele.User{ID: advertiserCustomer.TgID}, msg, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
	if err != nil {
		b.log.Errorln(err)
	}
}

func (b *Bot) handleAdmin(req string) {
	b.log.Debugln(req)

	data := map[string]interface{}{}

	if len(b.cfg.Bot.AdminsID) < 1 {
		b.log.Errorln("SETUP ADMINS")
		b.bridge.CodeMessage <- "00000"
	}

	for _, adm := range b.cfg.Bot.AdminsID {
		_, err := b.states.SetState(adm, bot_state.ADMIN_MODE, uuid.New(), data)
		if err != nil {
			b.log.Errorln(err)
		}

		_, err = b.tg.Send(&tele.User{ID: adm}, req)
		if err != nil {
			b.log.Errorln(err)
		}
	}
}

// handleAutoPost publishes a deal post to the channel per schedule.
func (b *Bot) handleAutoPost(dealID uuid.UUID) {
	ctx, cancel := context.WithTimeout(b.rootCtx, 30*time.Second)
	defer cancel()

	deal, err := b.deals.GetDeal(ctx, dealID)
	if err != nil {
		b.log.Errorf("Failed to get deal: %v", err)
		b.updateDealStatus(ctx, dealID, deal_status.ERR_PUBLISH)
		return
	}

	b.log.Debugf("deal target before map: %v", deal.Target)

	dealTarget, err := common.DealTargetFromMap(deal.Target)
	if err != nil {
		b.log.Errorf("Failed to parse target: %v", err)
		b.updateDealStatus(ctx, dealID, deal_status.ERR_BAD_TARGET)
		b.sendDealErrNotification(deal)
		return
	}
	postData := &dealTarget.PostData
	if postData.MessageText == "" && len(postData.MessageImages) == 0 {
		if flat, err := common.TgMsgDataFromMap(deal.Target); err == nil && (flat.MessageText != "" || len(flat.MessageImages) > 0) {
			postData = flat
		}
	}

	var msgsID []int
	channelRecipient := &tele.Chat{ID: deal.Edges.Channel.TgID}
	msgsID, err = b.sendContent(channelRecipient, postData)
	if err != nil {
		b.log.Errorf("failed to publish: %v", err)
		b.updateDealStatus(ctx, dealID, deal_status.ERR_PUBLISH)
		b.sendDealErrNotification(deal)
		return
	}

	now := time.Now()

	postDuration, err := parsePostDuration(deal.TargetType)
	if err != nil {
		b.log.Errorf("failed to parse post duration: %v", err)
	}

	_, err = b.deals.UpdateDeal(ctx, dealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.PUBLISHED))
		u.SetStatusUpdatedAt(now)
		u.SetExpiresAt(now.Add(postDuration))
		u.SetChannelPostIds(msgsID)
	})
	if err != nil {
		b.log.Errorf("Failed to update deal %s after publish: %v", dealID, err)
	}

	postLink := fmt.Sprintf("https://t.me/%s/%d", strings.TrimPrefix(deal.Edges.Channel.TgUsername, "@"), msgsID[0])
	channelName := deal.Edges.Channel.TgName
	if channelName == "" {
		channelName = deal.Edges.Channel.TgUsername
	}

	params := map[string]interface{}{
		"DealID":         util.EscapeMarkdownV2(dealID.String()),
		"ChannelName":    util.EscapeMarkdownV2(channelName),
		"CommonDeadline": util.EscapeMarkdownV2(deal.CommonDeadline.Format("02 Jan 2006, 15:04 UTC")),
		"TopDeadline":    util.EscapeMarkdownV2(deal.TopDeadline.Format("02 Jan 2006, 15:04 UTC")),
		"PostLink":       postLink,
	}

	b.sendDealNotification(deal, "deal", "published", params)
}

func (b *Bot) handleDeal(req common.ShowDealDataBotRequest) {
	ctx, cancel := context.WithTimeout(b.rootCtx, 15*time.Second)
	defer cancel()

	customer, err := b.customers.GetCustomer(ctx, req.UserID)
	if err != nil {
		b.log.Errorf("handleDeal: get customer %s: %v", req.UserID, err)
		return
	}

	deal, err := b.deals.GetDeal(ctx, req.DealID)
	if err != nil {
		b.log.Errorf("handleDeal: get deal %s: %v", req.DealID, err)
		return
	}

	locale := resolveLocale(customer.TgLanguage)
	channelName := ""
	if deal.Edges.Channel != nil {
		channelName = deal.Edges.Channel.TgName
		if channelName == "" {
			channelName = deal.Edges.Channel.TgUsername
		}
	}
	statusStr := deal_status.DealStatus(deal.Status).String()
	priceSpoiler := messages.WrapSpoiler(fmt.Sprintf("%.2f TON", deal.TonPrice))

	infoMsg, err := b.messages.Render(locale, "deal", "get_deal_data_info", map[string]interface{}{
		"DealID":       util.EscapeMarkdownV2(deal.ID.String()),
		"ChannelName":  util.EscapeMarkdownV2(channelName),
		"Status":       util.EscapeMarkdownV2(statusStr),
		"TargetType":   util.EscapeMarkdownV2(deal.TargetType),
		"PriceSpoiler": priceSpoiler,
	})
	if err != nil {
		b.log.Errorf("handleDeal: render info: %v", err)
		infoMsg = "*Deal Info*\n\nDeal ID: `" + deal.ID.String() + "`"
	}

	recipient := &tele.User{ID: customer.TgID}
	_, _ = b.tg.Send(recipient, infoMsg, tele.ModeMarkdownV2)

	dealTarget, err := common.DealTargetFromMap(deal.Target)
	if err != nil {
		b.log.Errorf("handleDeal: parse target: %v", err)
		noPostMsg, _ := b.messages.Render(locale, "deal", "get_deal_data_no_post", nil)
		_, _ = b.tg.Send(recipient, noPostMsg, tele.ModeMarkdownV2)
		return
	}
	postData := &dealTarget.PostData
	if postData.MessageText == "" && len(postData.MessageImages) == 0 {
		if flat, err := common.TgMsgDataFromMap(deal.Target); err == nil && (flat.MessageText != "" || len(flat.MessageImages) > 0) {
			postData = flat
		}
	}

	if postData.MessageText == "" && len(postData.MessageImages) == 0 {
		noPostMsg, _ := b.messages.Render(locale, "deal", "get_deal_data_no_post", nil)
		_, _ = b.tg.Send(recipient, noPostMsg, tele.ModeMarkdownV2)
		return
	}

	_, err = b.sendContent(recipient, postData)
	if err != nil {
		b.log.Errorf("handleDeal: send content: %v", err)
	}
}

// updateDealStatus updates deal status in the database.
func (b *Bot) updateDealStatus(ctx context.Context, dealID uuid.UUID, status deal_status.DealStatus) {
	_, err := b.deals.UpdateDeal(ctx, dealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(status))
		u.SetStatusUpdatedAt(time.Now())
	})
	if err != nil {
		b.log.Errorf("Failed to update deal %s status to %s: %v", dealID, status, err)
	}
}

func (b *Bot) sendDealErrNotification(deal *ent.Deal) {
	localeAdv := resolveLocale(deal.Edges.Advertiser.TgLanguage)
	localeMgr := resolveLocale(deal.Edges.ChannelManager.TgLanguage)

	msgAdv, err := b.messages.Render(localeAdv, "deal", "err_publish_advertiser", nil)
	if err != nil {
		b.log.Errorf("render err_publish_advertiser: %v", err)
		return
	}

	msgMgr, err := b.messages.Render(localeMgr, "deal", "err_publish_manager", nil)
	if err != nil {
		b.log.Errorf("render err_publish_manager: %v", err)
		return
	}

	openDealLabelAdv := b.messages.RenderOrDefault(localeAdv, "buttons", "open_deal", nil, "📋 Open Deal")
	openDealLabelMgr := b.messages.RenderOrDefault(localeMgr, "buttons", "open_deal", nil, "📋 Open Deal")
	dealWeb := &tele.WebApp{URL: b.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()}
	markupAdv := &tele.ReplyMarkup{}
	markupAdv.Inline(markupAdv.Row(markupAdv.WebApp(openDealLabelAdv, dealWeb)))
	markupMgr := &tele.ReplyMarkup{}
	markupMgr.Inline(markupMgr.Row(markupMgr.WebApp(openDealLabelMgr, dealWeb)))

	_, err = b.tg.Send(&tele.User{ID: deal.Edges.Advertiser.TgID}, msgAdv, markupAdv, tele.ModeMarkdownV2)
	if err != nil {
		b.log.Errorln(err)
	}

	_, err = b.tg.Send(&tele.User{ID: deal.Edges.ChannelManager.TgID}, msgMgr, markupMgr, tele.ModeMarkdownV2)
	if err != nil {
		b.log.Errorln(err)
	}
}

func (b *Bot) sendDealNotification(deal *ent.Deal, section, key string, params map[string]interface{}) {
	// Check if edges are loaded
	if deal.Edges.Advertiser == nil || deal.Edges.ChannelManager == nil {
		b.log.Errorf("sendDealNotification: deal edges not loaded for deal %s", deal.ID)
		return
	}

	localeAdv := resolveLocale(deal.Edges.Advertiser.TgLanguage)
	localeMgr := resolveLocale(deal.Edges.ChannelManager.TgLanguage)

	msgAdv, err := b.messages.Render(localeAdv, section, key, params)
	if err != nil {
		b.log.Errorf("render %s.%s for advertiser: %v", section, key, err)
		return
	}

	msgMgr, err := b.messages.Render(localeMgr, section, key, params)
	if err != nil {
		b.log.Errorf("render %s.%s for manager: %v", section, key, err)
		return
	}

	openDealLabelAdv := b.messages.RenderOrDefault(localeAdv, "buttons", "open_deal", nil, "📋 Open Deal")
	openDealLabelMgr := b.messages.RenderOrDefault(localeMgr, "buttons", "open_deal", nil, "📋 Open Deal")
	dealWeb := &tele.WebApp{URL: b.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()}
	markupAdv := &tele.ReplyMarkup{}
	markupAdv.Inline(markupAdv.Row(markupAdv.WebApp(openDealLabelAdv, dealWeb)))
	markupMgr := &tele.ReplyMarkup{}
	markupMgr.Inline(markupMgr.Row(markupMgr.WebApp(openDealLabelMgr, dealWeb)))

	_, err = b.tg.Send(&tele.User{ID: deal.Edges.Advertiser.TgID}, msgAdv, markupAdv, tele.ModeMarkdownV2)
	if err != nil {
		b.log.Errorln(err)
	}

	_, err = b.tg.Send(&tele.User{ID: deal.Edges.ChannelManager.TgID}, msgMgr, markupMgr, tele.ModeMarkdownV2)
	if err != nil {
		b.log.Errorln(err)
	}
}

func (b *Bot) handleNotification(req common.Notification) {
	if req.WebAppURL != "" {
		btnKey := "open_deals"
		defaultLabel := "📋 Open Deals"
		if strings.Contains(req.WebAppURL, "/deal/") {
			btnKey = "open_deal"
			defaultLabel = "📋 Open Deal"
		}
		btnLabel := b.messages.RenderOrDefault("en", "buttons", btnKey, nil, defaultLabel)
		dealsWeb := &tele.WebApp{URL: req.WebAppURL}
		markup := &tele.ReplyMarkup{}
		markup.Inline(
			markup.Row(
				markup.WebApp(btnLabel, dealsWeb),
			),
		)
		_, err := b.tg.Send(&tele.User{ID: req.UserID}, req.Text, markup, tele.ModeMarkdownV2)
		if err != nil {
			b.log.Errorln(err)
		}
		return
	}

	_, err := b.tg.Send(&tele.User{ID: req.UserID}, req.Text, tele.ModeMarkdownV2)
	if err != nil {
		b.log.Errorln(err)
	}
}

// resolveLocale returns locale with fallback to "en"
func resolveLocale(locale string) string {
	if locale == "" {
		return "en"
	}
	return locale
}
