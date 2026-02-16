package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/samber/do/v2"
)

const (
	MinChannelSubscribers = 50 // Min subscribers count for stats access
)

type Config struct {
	AppMode  string
	Indexer  bool
	StatsBot bool

	Bot      Bot
	UserBot  UserBot
	Postgres Postgres
	Server   Server

	InitDataExpire time.Duration

	JWTSecret []byte
	JWTExpire time.Duration

	Ton Ton
}

// Echo server port
type Server struct {
	Port           string `env:"SERVER_PORT" envDefault:"8080"`
	AllowedOrigins string `env:"ALLOWED_ORIGINS"`
}

type Bot struct {
	ID        int64
	Username  string
	AdminsID  []int64
	Token     string `env:"BOT_TOKEN"`
	WebAppURL string `env:"WEB_APP_URL"`
}

type UserBot struct {
	UserID      int64
	Username    string
	AppHash     string `env:"APP_HASH"`
	AppID       int    `env:"APP_ID"`
	PhoneNumber string `env:"PHONE_NUMBER"`
}

type Postgres struct {
	DSN string `env:"DATABASE_DSN"`
}

type Ton struct {
	MasterMnemonic string `env:"TON_MASTER_MNEMONIC"`
	TestNet        string `env:"TON_TEST_NET"`
	MainNet        string `env:"TON_MAIN_NET"`
}

func NewConfig(_ do.Injector) (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Printf("running without load env")
	}

	h := os.Getenv("INIT_DATA_EXPIRE")
	if h == "" {
		h = "6h"
	}
	hours, err := time.ParseDuration(h)
	if err != nil {
		return nil, err
	}

	t := os.Getenv("JWT_TIME")
	if t == "" {
		t = "6h"
	}
	jwtTime, err := time.ParseDuration(t)

	var aID int
	appID := os.Getenv("APP_ID")
	if appID != "" {
		aID, err = strconv.Atoi(appID)
		if err != nil {
			return nil, err
		}
	}

	var (
		indexer  bool
		statsBot bool
	)
	indxr := os.Getenv("INDEXER")
	if indxr == "true" {
		indexer = true
	}

	sBot := os.Getenv("STATS_BOT")
	if sBot == "true" {
		statsBot = true
	}

	admins := make([]int64, 0)
	adminsID := os.Getenv("ADMINS_ID")
	if adminsID != "" {
		adminsList := strings.Split(adminsID, ",")
		for _, admin := range adminsList {
			var adminID int
			adminID, err = strconv.Atoi(admin)
			if err != nil {
				return nil, err
			}
			admins = append(admins, int64(adminID))
		}
	}

	return &Config{
		AppMode:  os.Getenv("APP_MODE"),
		Indexer:  indexer,
		StatsBot: statsBot,
		//echo
		Server: Server{
			Port:           os.Getenv("SERVER_PORT"),
			AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		},

		Bot: Bot{
			AdminsID:  admins,
			Token:     os.Getenv("BOT_TOKEN"),
			WebAppURL: os.Getenv("WEB_APP_URL"),
		},
		UserBot: UserBot{
			AppHash:     os.Getenv("APP_HASH"),
			AppID:       aID,
			PhoneNumber: os.Getenv("PHONE_NUMBER"),
		},

		//database
		Postgres: Postgres{
			DSN: os.Getenv("DATABASE_DSN"),
		},

		InitDataExpire: hours,
		JWTSecret:      []byte(os.Getenv("JWT_SECRET")),
		JWTExpire:      jwtTime,

		Ton: Ton{
			MasterMnemonic: os.Getenv("TON_MASTER_MNEMONIC"),
			TestNet:        os.Getenv("TON_TEST_NET"),
			MainNet:        os.Getenv("TON_MAIN_NET"),
		},
	}, nil
}
