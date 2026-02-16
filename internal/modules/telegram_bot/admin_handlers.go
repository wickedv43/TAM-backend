package telegram_bot

import (
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleAdminMode(c tele.Context) error {
	b.log.Debugf("Admin mode message")

	msg := c.Message().Text

	b.bridge.CodeMessage <- msg

	return b.states.ClearState(c.Sender().ID)
}
