package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"go.uber.org/zap"
)

const (
	ctxUser         = "userID"
	ctxDeal         = "deal"
	ctxDealID       = "deal_id"
	ctxIsAdvertiser = "is_advertiser"
	ctxIsManager    = "is_manager"
	ctxChannelID    = "channel_id"
)

// authMiddleware validates JWT from Authorization header.
func (s *Server) authMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, "Missing Authorization header")
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, "Invalid Authorization format. Expected 'Bearer <token>'")
			}

			tokenString := parts[1]

			customer, err := s.validateJWT(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(ctxUser, customer.ID)

			return next(c)
		}
	}
}

func (s *Server) GetContextUser(c echo.Context) (uuid.UUID, error) {
	userID := c.Get(ctxUser)
	if userID == nil {
		return uuid.Nil, errors.New("userID not found in context")
	}

	uID, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid userID type in context")
	}

	return uID, nil
}

func (s *Server) requestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)
			req := c.Request()
			res := c.Response()

			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", res.Status),
				zap.Duration("latency", latency),
				zap.String("ip", c.RealIP()),
			}

			if req.URL.RawQuery != "" {
				fields = append(fields, zap.String("query", req.URL.RawQuery))
			}

			if res.Size > 0 {
				fields = append(fields, zap.Int64("size", res.Size))
			}

			if userID := c.Get(ctxUser); userID != nil {
				if uid, ok := userID.(uuid.UUID); ok {
					fields = append(fields, zap.String("user_id", uid.String()))
				}
			}

			if reqID := res.Header().Get(echo.HeaderXRequestID); reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}

			logger := s.log.Desugar()

			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error("HTTP request failed", fields...)
				return err
			}

			switch {
			case res.Status >= 500:
				logger.Error("HTTP request", fields...)
			case res.Status >= 400:
				logger.Warn("HTTP request", fields...)
			default:
				logger.Info("HTTP request", fields...)
			}

			return nil
		}
	}
}

// RequireDealParticipant ensures the user is a participant of the deal.
func (s *Server) RequireDealParticipant() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			customerID, err := s.GetContextUser(c)
			if err != nil {
				status := common.GetHTTPStatus(common.ErrUnauthorized)
				message := s.GetUserMessage(common.ErrUnauthorized)
				return c.JSON(status, map[string]string{"error": message})
			}

			ctx := c.Request().Context()

			var dealID uuid.UUID

			if c.Request().Method == http.MethodGet {
				dealIDStr := c.Param("id")
				dealID, err = uuid.Parse(dealIDStr)
				if err != nil {
					status := common.GetHTTPStatus(common.ErrInvalidUUID)
					message := s.GetUserMessage(common.ErrInvalidUUID)
					return c.JSON(status, map[string]string{"error": message})
				}
			} else {
				var bodyBytes []byte
				if c.Request().Body != nil {
					bodyBytes, err = io.ReadAll(c.Request().Body)
					if err != nil {
						status := common.GetHTTPStatus(common.ErrInvalidRequest)
						message := s.GetUserMessage(common.ErrInvalidRequest)
						return c.JSON(status, map[string]string{"error": message})
					}
				}

				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				var req struct {
					DealID uuid.UUID `json:"deal_id"`
				}
				if err = json.Unmarshal(bodyBytes, &req); err != nil {
					status := common.GetHTTPStatus(common.ErrInvalidRequest)
					message := s.GetUserMessage(common.ErrInvalidRequest)
					return c.JSON(status, map[string]string{"error": message})
				}
				dealID = req.DealID
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			deal, err := s.deals.GetDeal(ctx, dealID)
			if err != nil {
				if ent.IsNotFound(err) {
					status := common.GetHTTPStatus(common.ErrDealNotFound)
					message := s.GetUserMessage(common.ErrDealNotFound)
					return c.JSON(status, map[string]string{"error": message})
				}
				s.log.Errorf("get deal: %+v", err)
				status := common.GetHTTPStatus(common.ErrInternalError)
				message := s.GetUserMessage(common.ErrInternalError)
				return c.JSON(status, map[string]string{"error": message})
			}

			isAdvertiser := deal.AdvertiserCustomerID == customerID
			isManager := deal.ChannelManagerID.String() != "" && deal.ChannelManagerID == customerID

			if !isAdvertiser && !isManager {
				status := common.GetHTTPStatus(common.ErrNotDealParticipant)
				message := s.GetUserMessage(common.ErrNotDealParticipant)
				return c.JSON(status, map[string]string{"error": message})
			}

			c.Set(ctxDeal, deal)
			c.Set(ctxDealID, dealID)
			c.Set(ctxIsAdvertiser, isAdvertiser)
			c.Set(ctxIsManager, isManager)

			return next(c)
		}
	}
}

func (s *Server) RequireDealAdvertiserOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAdvertiser, _ := c.Get(ctxIsAdvertiser).(bool)
			if !isAdvertiser {
				status := common.GetHTTPStatus(common.ErrForbidden)
				message := s.GetUserMessage(common.ErrForbidden)
				return c.JSON(status, map[string]string{"error": message})
			}
			return next(c)
		}
	}
}

func (s *Server) RequireDealChannelManagerOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isManager, _ := c.Get(ctxIsManager).(bool)
			if !isManager {
				status := common.GetHTTPStatus(common.ErrForbidden)
				message := s.GetUserMessage(common.ErrForbidden)
				return c.JSON(status, map[string]string{"error": message})
			}
			return next(c)
		}
	}
}

func (s *Server) RequireChannelAccess() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			customerID, err := s.GetContextUser(c)
			if err != nil {
				status := common.GetHTTPStatus(common.ErrUnauthorized)
				message := s.GetUserMessage(common.ErrUnauthorized)
				return c.JSON(status, map[string]string{"error": message})
			}

			ctx := c.Request().Context()

			var channelID uuid.UUID

			if c.Request().Method == http.MethodGet || c.Request().Method == http.MethodDelete {
				channelIDStr := c.Param("id")
				channelID, err = uuid.Parse(channelIDStr)
				if err != nil {
					status := common.GetHTTPStatus(common.ErrInvalidUUID)
					message := s.GetUserMessage(common.ErrInvalidUUID)
					return c.JSON(status, map[string]string{"error": message})
				}
			} else {
				var bodyBytes []byte
				if c.Request().Body != nil {
					bodyBytes, err = io.ReadAll(c.Request().Body)
					if err != nil {
						status := common.GetHTTPStatus(common.ErrInvalidRequest)
						message := s.GetUserMessage(common.ErrInvalidRequest)
						return c.JSON(status, map[string]string{"error": message})
					}
				}

				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				var req struct {
					ChannelID uuid.UUID `json:"channel_id"`
				}
				if err = json.Unmarshal(bodyBytes, &req); err != nil {
					status := common.GetHTTPStatus(common.ErrInvalidRequest)
					message := s.GetUserMessage(common.ErrInvalidRequest)
					return c.JSON(status, map[string]string{"error": message})
				}
				channelID = req.ChannelID
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			custCh, err := s.customers.GetCustomerChannel(ctx, customerID, channelID)
			s.log.Debugf("customer channel access: %+v", custCh)
			if err != nil {
				if ent.IsNotFound(err) {
					status := common.GetHTTPStatus(common.ErrNoChannelAccess)
					message := s.GetUserMessage(common.ErrNoChannelAccess)
					return c.JSON(status, map[string]string{"error": message})
				}
				s.log.Errorf("get customer channel: %+v", err)
				status := common.GetHTTPStatus(common.ErrInternalError)
				message := s.GetUserMessage(common.ErrInternalError)
				return c.JSON(status, map[string]string{"error": message})
			}

			c.Set(ctxChannelID, channelID)

			return next(c)
		}
	}
}
