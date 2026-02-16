package stats_bot

import (
	"context"

	"github.com/gotd/td/telegram/auth"
	"github.com/samber/do/v2"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/logger"
	"github.com/wickedv43/TAM-backend/internal/modules/channels"
	"github.com/wickedv43/TAM-backend/internal/modules/stats_bot/tg_api"

	"go.uber.org/zap"
)

type Interface interface {
	Start(ctx context.Context) error

	GetChannel(req common.StatsBotAddChannelRequest) (common.AddChannelResponse, error)
}

type StatsBot struct {
	cfg *config.Config
	log *zap.SugaredLogger

	tgClient *tg_api.Client

	bridge *channels.Channels

	cancelFunc context.CancelFunc
}

func New(i do.Injector) (*StatsBot, error) {
	var s StatsBot
	s.cfg = do.MustInvoke[*config.Config](i)
	s.log = do.MustInvoke[*logger.Logger](i).Named("stats_bot")
	s.tgClient = do.MustInvoke[*tg_api.Client](i)
	s.bridge = do.MustInvoke[*channels.Channels](i)

	authenticator := ChanAuthenticator{
		PhoneValue:          s.cfg.UserBot.PhoneNumber,
		BotAdminMessageChan: s.bridge.AdminMessage,
		BotAdminCodeChan:    s.bridge.CodeMessage,
	}

	flow := auth.NewFlow(authenticator, auth.SendCodeOptions{})
	s.tgClient.Flow = &flow

	return &s, nil
}

func (s *StatsBot) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.log.Infoln("shutting down StatsBot...")

		s.cancelFunc()
	}()

	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	if !s.cfg.StatsBot {
		s.log.Warn("StatsBot is OFF")
		return nil
	}

	return s.tgClient.Run(ctx, false)
}
