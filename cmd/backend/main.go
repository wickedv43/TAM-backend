package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/logger"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/channels"
	"github.com/wickedv43/TAM-backend/internal/modules/server"
	"github.com/wickedv43/TAM-backend/internal/modules/stats_bot"
	"github.com/wickedv43/TAM-backend/internal/modules/stats_bot/tg_api"
	"github.com/wickedv43/TAM-backend/internal/modules/storage"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot"
	"github.com/wickedv43/TAM-backend/internal/modules/ton"

	"github.com/samber/do/v2"
	"golang.org/x/sync/errgroup"

	_ "github.com/wickedv43/TAM-backend/docs"
)

// @title           TAM Backend API
// @version         1.0
// @description     Backend for Telegram WebApp (TAM).
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.email   support@swagger.io

// @host            localhost:8080
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	time.Local = time.UTC

	//provide part
	i := do.New()

	do.ProvideNamedValue(i, "root.context", ctx)
	do.Provide(i, logger.NewLogger)
	do.Provide(i, config.NewConfig)
	do.Provide(i, messages.New)
	do.Provide(i, telegram_bot.NewBot)
	do.Provide(i, storage.NewPostgres)
	do.Provide(i, server.NewServer)
	do.Provide(i, stats_bot.New)
	do.Provide(i, tg_api.New)
	do.Provide(i, ton.NewTon)
	do.Provide(i, channels.NewChannels)

	//zap
	log := do.MustInvoke[*logger.Logger](i)
	log.Infoln("starting application....")

	//init part
	var (
		tgBot *telegram_bot.Bot
		s     *server.Server
		t     *ton.Ton
	)

	eg := errgroup.Group{}
	eg.Go(func() (err error) {
		tgBot, err = do.Invoke[*telegram_bot.Bot](i)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})
	eg.Go(func() (err error) {
		s, err = do.Invoke[*server.Server](i)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})
	eg.Go(func() (err error) {
		t, err = do.Invoke[*ton.Ton](i)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})
	err := eg.Wait()
	if err != nil {
		log.Errorf("failed start: %s", err)
		return
	}

	// Start part
	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		tgBot.Start(ctx)
	}()
	go func() {
		defer wg.Done()
		s.Start(ctx)
	}()
	go func() {
		defer wg.Done()
		t.Start(ctx)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh

	cancel()
	wg.Wait()

	_ = i.ShutdownWithContext(ctx)

	log.Infoln("grace shutdown")
}
