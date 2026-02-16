package stats_bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// ChanAuthenticator implements auth.UserAuthenticator.
type ChanAuthenticator struct {
	PhoneValue string

	BotAdminMessageChan chan string
	BotAdminCodeChan    chan string
}

func (a ChanAuthenticator) Phone(_ context.Context) (string, error) {
	return a.PhoneValue, nil
}

func (a ChanAuthenticator) Password(_ context.Context) (string, error) {
	msg := fmt.Sprintf("Please send 2FA code")

	a.BotAdminMessageChan <- msg

	code := <-a.BotAdminCodeChan
	return strings.TrimSpace(code), nil
}

func (a ChanAuthenticator) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (a ChanAuthenticator) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signup not implemented")
}

func (a ChanAuthenticator) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	msg := fmt.Sprintf("Please send code")

	a.BotAdminMessageChan <- msg

	code := <-a.BotAdminCodeChan
	return strings.TrimSpace(code), nil
}
