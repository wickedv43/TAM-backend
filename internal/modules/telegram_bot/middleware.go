package telegram_bot

import (
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

const (
	contextEndpointKey = "context_endpoint"
	contextLoggerKey   = "context_logger"
)

func ctxLog(c tele.Context) *zap.SugaredLogger {
	if logger, ok := c.Get(contextLoggerKey).(*zap.SugaredLogger); ok && logger != nil {
		return logger
	}
	return zap.NewNop().Sugar()
}

func (b *Bot) Logger() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			ctxLogger := b.log
			c.Set(contextLoggerKey, ctxLogger)

			start := time.Now()
			err := next(c)
			stop := time.Now()

			var usernameOrID string
			if s := c.Sender(); s != nil {
				usernameOrID = s.Username
				if usernameOrID == "" {
					usernameOrID = "id:" + strconv.FormatInt(s.ID, 10)
				}
			} else {
				usernameOrID = "channel_post"
			}

			ctxLogger.Infof(
				"update:%d user=%s endpoint=%s latency=%s",
				c.Update().ID,
				usernameOrID,
				strings.TrimSpace(GetCtxEndpoint(c)),
				stop.Sub(start),
			)

			return err
		}
	}
}

func (b *Bot) HandleWithEndpoint(e interface{}, h tele.HandlerFunc, m ...tele.MiddlewareFunc) {
	handler := func(c tele.Context) error {
		var textEndpoint string

		switch endpoint := e.(type) {
		case string:
			textEndpoint = endpoint
		case tele.CallbackEndpoint:
			textEndpoint = endpoint.CallbackUnique()
		default:
			textEndpoint = "unknown"
		}

		c.Set(contextEndpointKey, textEndpoint)

		return h(c)
	}

	b.tg.Handle(e, handler, m...)
}

func GetCtxEndpoint(c tele.Context) string {
	value, ok := c.Get(contextEndpointKey).(string)
	if !ok {
		return "unknown"
	}

	return value
}

func onError(err error, c tele.Context) {
	var usernameOrID string
	if s := c.Sender(); s != nil {
		usernameOrID = s.Username
		if usernameOrID == "" {
			usernameOrID = "id:" + strconv.FormatInt(s.ID, 10)
		}
	} else {
		usernameOrID = "channel_post"
	}

	errMsg := err.Error()

	if strings.Contains(errMsg, "message to edit not found") ||
		strings.Contains(errMsg, "message is not modified") ||
		strings.Contains(errMsg, "message can't be edited") {
		ctxLog(c).Warnf(
			"update:%d user=%s endpoint=%s error=%s",
			c.Update().ID,
			usernameOrID,
			GetCtxEndpoint(c),
			errMsg,
		)
		return
	}

	ctxLog(c).Errorf(
		"update:%d user=%s endpoint=%s error=%s",
		c.Update().ID,
		usernameOrID,
		GetCtxEndpoint(c),
		errMsg,
	)
}
