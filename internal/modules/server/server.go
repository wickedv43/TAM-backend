package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/logger"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/stats_bot"
	"github.com/wickedv43/TAM-backend/internal/modules/storage"
	"github.com/wickedv43/TAM-backend/internal/modules/ton"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/do/v2"

	"go.uber.org/zap"

	echoSwagger "github.com/swaggo/echo-swagger"
	chans "github.com/wickedv43/TAM-backend/internal/modules/channels"
)

var (
	// CORS configuration
	corsAllowMethods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
	}

	corsAllowHeaders = []string{
		echo.HeaderOrigin,
		echo.HeaderContentType,
		echo.HeaderAccept,
		echo.HeaderAuthorization,
		"X-Aspect-Ratio",
		"X-Auth-Login-Id",
	}

	corsExposeHeaders = []string{
		echo.HeaderContentLength,
		echo.HeaderContentType,
		echo.HeaderAuthorization,
		"X-Auth-Login-Id",
	}
)

type Server struct {
	echo     *echo.Echo
	messages *messages.Messages

	cfg *config.Config
	log *zap.SugaredLogger

	customers storage.CustomersInterface
	channels  storage.ChannelsInterface
	deals     storage.DealsInterface
	briefs    storage.BriefsInterface

	bridge *chans.Channels

	statsBot stats_bot.Interface

	db *storage.PostgresDB

	ton ton.WalletInterface
}

func NewServer(i do.Injector) (*Server, error) {
	s := &Server{
		echo:     echo.New(),
		messages: do.MustInvoke[*messages.Messages](i),
		log:      do.MustInvoke[*logger.Logger](i).Named("echo"),
		cfg:      do.MustInvoke[*config.Config](i),
		statsBot: do.MustInvoke[*stats_bot.StatsBot](i),

		customers: do.MustInvoke[*storage.PostgresDB](i),
		channels:  do.MustInvoke[*storage.PostgresDB](i),
		deals:     do.MustInvoke[*storage.PostgresDB](i),
		briefs:    do.MustInvoke[*storage.PostgresDB](i),

		ton:    do.MustInvoke[*ton.Ton](i),
		bridge: do.MustInvoke[*chans.Channels](i),
	}

	// CORS configuration
	corsConfig := middleware.CORSConfig{
		AllowMethods:     corsAllowMethods,
		AllowHeaders:     corsAllowHeaders,
		AllowCredentials: true,
		ExposeHeaders:    corsExposeHeaders,
	}

	allowedOrigins := []string{"http://localhost:5173"}
	allowedOriginsEnv := strings.TrimSpace(s.cfg.Server.AllowedOrigins)
	if allowedOriginsEnv != "" {
		if allowedOriginsEnv == "*" {
			// Allow all origins: with credentials Echo must reflect request origin, not "*"
			corsConfig.AllowOrigins = []string{"*"}
			corsConfig.UnsafeWildcardOriginWithAllowCredentials = true
		} else {
			// Parse comma-separated origins
			allowedOrigins = splitAndTrim(s.cfg.Server.AllowedOrigins, ",")
			corsConfig.AllowOrigins = allowedOrigins
		}
	} else {
		// Default origins
		corsConfig.AllowOrigins = allowedOrigins
	}

	s.echo.Use(middleware.CORSWithConfig(corsConfig))

	s.echo.Use(s.requestLoggerMiddleware())
	s.echo.Use(middleware.Recover())

	s.echo.Static("/uploads", "uploads")

	s.registerRoutes()

	return s, nil
}

func (s *Server) registerRoutes() {
	// API v1
	api := s.echo.Group("/api/v1")

	api.GET("/swagger/*", echoSwagger.WrapHandler)
	// Auth endpoint (without JWT middleware)
	api.POST("/auth", s.onAuth)

	// Apply JWT middleware to all routes below
	api.Use(s.authMiddleware())

	// Customers
	customers := api.Group("/customers")
	customers.GET("/me", s.GetMe)
	customers.PATCH("/me", s.UpdateMe)
	customers.GET("/me/deals", s.GetMyDeals)
	customers.POST("/withdraw", s.Withdraw)

	// Channels
	channels := api.Group("/channels")
	channels.POST("/add-channel", s.AddChannel)
	channels.GET("/list-channels", s.ListChannels)
	channels.GET("/:id", s.GetChannel)
	channelsProtected := channels.Group("")
	channelsProtected.Use(s.RequireChannelAccess())
	channelsProtected.POST("/provide-channel-data", s.ProvideChannelData)
	channelsProtected.DELETE("/:id", s.DeleteChannel)

	// Deals
	deals := api.Group("/deals")
	deals.GET("/list", s.ListDeals)
	deals.POST("/offer-deal", s.OfferDeal)
	dealsProtected := deals.Group("")
	dealsProtected.Use(s.RequireDealParticipant())

	dealAdvertiserOnly := dealsProtected.Group("")
	dealAdvertiserOnly.Use(s.RequireDealAdvertiserOnly())
	dealAdvertiserOnly.POST("/send-deal-data", s.SendDealData)
	dealAdvertiserOnly.POST("/make-deal", s.MakeDeal)
	dealAdvertiserOnly.POST("/approve-deal-data", s.ApproveDealData)
	// dealAdvertiserOnly.POST("/revise-deal-data", s.ReviseDealData)

	dealChannelOnly := dealsProtected.Group("")
	dealChannelOnly.Use(s.RequireDealChannelManagerOnly())
	dealChannelOnly.POST("/accept-deal", s.AcceptDeal)
	dealChannelOnly.POST("/approve-and-propose-time", s.ApproveAndProposeTime)

	dealBothSides := dealsProtected.Group("")
	dealBothSides.POST("/deal-message", s.DealMessage)
	dealBothSides.POST("/decline-deal", s.DeclineDeal)
	dealBothSides.POST("/cancel-deal", s.CancelDeal)
	dealBothSides.POST("/get-deal-data", s.GetDealData)
	dealBothSides.GET("/:id", s.GetDeal)

	// Briefs
	briefs := api.Group("/briefs")
	briefs.POST("/add-brief", s.AddBrief)
	briefs.GET("/:id", s.GetBrief)
	briefs.PATCH("/:id", s.UpdateBrief)
	briefs.GET("/list", s.ListBriefs)

	// Upload
	api.POST("/upload", s.Upload)
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.log.Infoln("shutting down API server...")

		err := s.echo.Shutdown(ctx)
		if err != nil {
			s.log.Errorf("failed to shutdown API server: %s", err)
		}
	}()

	//start part
	s.log.Info("echo started")
	port := fmt.Sprintf(":%s", s.cfg.Server.Port)

	go func() {
		err := s.statsBot.Start(ctx)
		if err != nil {
			s.log.Errorf("failed to start stats bot: %s", err)
		}
	}()
	go s.startRefunder(ctx)
	go s.startDealPostPublisher(ctx)

	return s.echo.Start(port)
}

// splitAndTrim splits a string by delimiter and trims whitespace from each part
func splitAndTrim(s, delimiter string) []string {
	parts := strings.Split(s, delimiter)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseChannelUsername(input string) string {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "t.me/") {
		parts := strings.Split(input, "t.me/")
		if len(parts) > 1 {
			username := strings.Split(parts[1], "/")[0]
			return strings.TrimPrefix(username, "@")
		}
	}
	return strings.TrimPrefix(input, "@")
}
