package tg_api

import (
	"context"
	"os"
	"path/filepath"
	"time"

	pebbledb "github.com/cockroachdb/pebble"
	boltstor "github.com/gotd/contrib/bbolt"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/contrib/pebble"
	mtStorage "github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"github.com/samber/do/v2"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/logger"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	lj "gopkg.in/natefinch/lumberjack.v2"
)

type Client struct {
	api             *tg.Client
	waiter          *floodwait.Waiter
	client          *telegram.Client
	Flow            *auth.Flow
	lg              *zap.Logger
	UpdatesRecovery *updates.Manager
	peerDB          *pebble.PeerStorage
	db              *pebbledb.DB
	boltdb          *bbolt.DB
	dispatcher      *tg.UpdateDispatcher

	cfg *config.Config
	log *zap.SugaredLogger
}

func New(i do.Injector) (*Client, error) {
	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*logger.Logger](i).Named("stats_bot").Named("telegram_client")

	var t Client
	t.log = log
	t.cfg = cfg

	path := "/app/session"
	if cfg.AppMode == "local" {
		path = "session"
	}

	if cfg.AppMode == "local-scheduler" {
		path = "session"
	}

	// This is needed to reuse session and not login every time.
	sessionDir := filepath.Join(path, sessionFolder(cfg.UserBot.PhoneNumber))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, err
	}

	logFilePath := filepath.Join(sessionDir, "log.jsonl")

	t.log.Debugf("storing session in %s, logs in %s\n", sessionDir, logFilePath)

	// Setting up logging to file with rotation.
	// Log to file, so we don't interfere with prompts and messages to user.
	logWriter := zapcore.AddSync(&lj.Logger{
		Filename:   logFilePath,
		MaxBackups: 3,
		MaxSize:    1, // megabytes
		MaxAge:     7, // days
	})
	logCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		logWriter,
		zap.DebugLevel,
	)
	lg := zap.New(logCore)

	defer func() { _ = lg.Sync() }()

	// So, we are storing session information in current directory, under subdirectory "session/phone_hash"
	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}
	// Peer storage, for resolve caching and short updates handling.

	db, err := pebbledb.Open(filepath.Join(sessionDir, "peers.pebble.db"), &pebbledb.Options{})
	if err != nil {
		return nil, err
	}
	peerDB := pebble.NewPeerStorage(db)

	// Setting up client.
	//
	// Dispatcher is used to register handlers for events.
	dispatcher := tg.NewUpdateDispatcher()
	// Setting up update handler that will fill peer storage before
	// calling dispatcher handlers.
	updateHandler := mtStorage.UpdateHook(dispatcher, peerDB)
	// Setting up persistent storage for qts/pts to be able to
	// recover after restart.
	boltdb, err := bbolt.Open(filepath.Join(sessionDir, "updates.bolt.db"), 0666, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create bolt storage")
	}
	boltstore := boltstor.NewStateStorage(boltdb)

	updatesRecovery := updates.New(updates.Config{
		Handler: updateHandler, // using previous handler with peerDB
		Logger:  lg.Named("updates_recovery"),
		Storage: boltstore,
	})

	// Handler of FLOOD_WAIT that will automatically retry request.
	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		t.log.Warn("Flood wait", zap.Duration("wait", wait.Duration))
	})

	// Filling client options.
	options := telegram.Options{
		Logger:         lg.Named("options"), // Passing logger for observability.
		SessionStorage: sessionStorage,      // Setting up session sessionStorage to store auth data.
		UpdateHandler:  updatesRecovery,     // Setting up handler for updates from server.
		Middlewares: []telegram.Middleware{
			// Setting up FLOOD_WAIT handler to automatically wait and retry request.
			waiter,
			// Setting up general rate limits to less likely get flood wait errors.
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
		// Disable automatic DC migration to handle it manually and avoid Waiter deadlock
		// Manual migration allows sequential processing: request -> MIGRATE error -> AuthExport -> retry
		DisableAutoMigration: true,
	}
	client := telegram.NewClient(cfg.UserBot.AppID, cfg.UserBot.AppHash, options)
	api := client.API()

	// Setting up resolver cache that will use peer storage.
	resolver := mtStorage.NewResolverCache(peer.Plain(api), peerDB)
	// Usage:
	//
	//	  if _, err := resolver.ResolveDomain(ctx, "tdlibchat"); err != nil {
	//		   return errors.Wrap(err, "resolve")
	//	  }
	_ = resolver

	// Registering handler for new private messages.

	t.api = api
	t.waiter = waiter
	t.client = client
	t.lg = lg
	t.peerDB = peerDB
	t.UpdatesRecovery = updatesRecovery
	t.db = db
	t.boltdb = boltdb
	t.dispatcher = &dispatcher
	t.Flow = &auth.Flow{}

	t.dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		return nil
	})

	t.dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		return nil
	})

	return &t, nil
}
