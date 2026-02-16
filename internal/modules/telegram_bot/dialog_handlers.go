package telegram_bot

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/wickedv43/TAM-backend/internal/common/constants/bot_state"
	"github.com/wickedv43/TAM-backend/internal/common/constants/post_data"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	tele "gopkg.in/telebot.v4"
)

// handleDialogMessage processes messages sent during a dialog session.
func (b *Bot) handleDialogMessage(c tele.Context, state *ent.BotState) error {
	targetIDRaw, ok := state.Data[recipientID].(float64)
	if !ok {
		b.log.Errorf("state data error: missing recipientID")
		return nil
	}
	targetID := int64(targetIDRaw)

	sID, ok := state.Data[senderID].(float64)
	if !ok {
		b.log.Errorf("state data error: missing senderID")
		return nil
	}

	sendID := int64(sID)

	sender := strconv.Itoa(int(sendID))

	state, err := b.states.GetState(sendID)
	if err != nil {
		b.log.Errorf("state data error: get state: %v", err)
		return nil
	}

	dealID := state.DealID.String()

	ctx := b.rootCtx
	deal, err := b.deals.GetDeal(ctx, state.DealID)
	if err != nil {
		b.log.Errorf("get deal for dialog message: %v", err)
		replyLabel := b.messages.RenderOrDefault("en", "buttons", "reply", nil, "Reply")
		menu := &tele.ReplyMarkup{}
		btnAnswer := menu.Data(replyLabel, answerMessage.Unique(), sender+" "+dealID)
		menu.Inline(menu.Row(btnAnswer))
		_, err = b.tg.Send(&tele.User{ID: targetID}, c.Message().Text, menu)
		if err != nil {
			return errors.Wrap(err, "telegram send message")
		}
		_ = b.states.ClearState(state.UserID)
		sentMsg := b.messages.RenderOrDefault("en", "dialog", "message_sent", nil, "✅ Message sent")
		_, _ = b.tg.Send(&tele.User{ID: sendID}, sentMsg)
		_ = c.Notify(tele.Typing)
		return nil
	}

	var recipientLocale string
	if deal.Edges.Advertiser != nil && deal.Edges.Advertiser.TgID == targetID {
		recipientLocale = resolveLocale(deal.Edges.Advertiser.TgLanguage)
	} else if deal.Edges.ChannelManager != nil && deal.Edges.ChannelManager.TgID == targetID {
		recipientLocale = resolveLocale(deal.Edges.ChannelManager.TgLanguage)
	} else {
		recipientLocale = "en"
	}

	var messageKey string
	var channelName string
	if deal.Edges.ChannelManager != nil && deal.Edges.ChannelManager.TgID == sendID {
		messageKey = "received_from_manager"
		if deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
			channelName = "@" + deal.Edges.Channel.TgUsername
		} else {
			channelName = "—"
		}
	} else {
		messageKey = "received_from_advertiser"
	}

	params := map[string]interface{}{
		"MessageText": util.EscapeMarkdownV2(c.Message().Text),
	}
	if channelName != "" {
		params["ChannelName"] = util.EscapeMarkdownV2(channelName)
	}

	formattedMsg, err := b.messages.Render(recipientLocale, "dialog", messageKey, params)
	if err != nil {
		b.log.Errorf("render dialog message: %v", err)
		formattedMsg = c.Message().Text
	}

	replyLabel := b.messages.RenderOrDefault(recipientLocale, "buttons", "reply", nil, "Reply")
	menu := &tele.ReplyMarkup{}
	btnAnswer := menu.Data(replyLabel, answerMessage.Unique(), sender+" "+dealID)

	menu.Inline(
		menu.Row(btnAnswer),
	)

	_, err = b.tg.Send(&tele.User{ID: targetID}, formattedMsg, menu, tele.ModeMarkdownV2)
	if err != nil {
		return errors.Wrap(err, "telegram send message")
	}

	err = b.states.ClearState(state.UserID)
	if err != nil {
		return errors.Wrap(err, "clear state")
	}

	senderLocale := "en"
	if deal.Edges.ChannelManager != nil && deal.Edges.ChannelManager.TgID == sendID {
		senderLocale = resolveLocale(deal.Edges.ChannelManager.TgLanguage)
	} else if deal.Edges.Advertiser != nil && deal.Edges.Advertiser.TgID == sendID {
		senderLocale = resolveLocale(deal.Edges.Advertiser.TgLanguage)
	}
	sentMsg := b.messages.RenderOrDefault(senderLocale, "dialog", "message_sent", nil, "✅ Message sent")
	_, _ = b.tg.Send(&tele.User{ID: sendID}, sentMsg)

	_ = c.Notify(tele.Typing)

	return nil
}

// handleAnswer initiates a reply dialog when the "Answer" button is clicked.
func (b *Bot) handleAnswer(c tele.Context) error {
	queryData := strings.Split(c.Data(), " ")

	rID, err := strconv.Atoi(queryData[0])
	if err != nil {
		return errors.Wrap(err, "telegram answer message")
	}
	recipient := int64(rID)

	dealID, err := uuid.Parse(queryData[1])
	if err != nil {
		return errors.Wrap(err, "telegram answer message")
	}

	data := map[string]interface{}{
		senderID:    c.Sender().ID,
		recipientID: recipient,
	}

	_, err = b.states.SetState(c.Sender().ID, bot_state.AWAITING_DIALOG_MESSAGE, dealID, data)
	if err != nil {
		b.log.Errorln(err)
	}

	deal, err := b.deals.GetDeal(b.rootCtx, dealID)
	channelName := ""
	if err == nil && deal != nil && deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
		channelName = "@" + deal.Edges.Channel.TgUsername
	} else if err == nil && deal != nil {
		channelName = "—"
	}

	locale := resolveLocale(c.Sender().LanguageCode)

	params := map[string]interface{}{
		"DealID":      util.EscapeMarkdownV2(dealID.String()),
		"ChannelName": util.EscapeMarkdownV2(channelName),
	}
	msg, err := b.messages.Render(locale, "dialog", "send_message", params)
	if err != nil {
		b.log.Errorf("render send_message: %v", err)
		return c.Reply("Please send a message") // fallback
	}

	return c.Reply(msg, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
}

// handleCancel cancels the current operation, clears state, and deletes preview messages.
func (b *Bot) handleCancel(c tele.Context) error {
	state, err := b.states.GetState(c.Sender().ID)
	if err != nil {
		b.log.Errorf("failed to get state in handleCancel: %v", err)
	}

	if state != nil && state.Data != nil {
		if idsInterface, ok := state.Data[post_data.MessagePreviewIDs].([]interface{}); ok {
			for _, idVal := range idsInterface {
				oldID := getInt(idVal)
				if oldID > 0 {
					_ = b.tg.Delete(&tele.Message{
						ID:   oldID,
						Chat: &tele.Chat{ID: state.UserID},
					})
				}
			}
		}
	}

	if state != nil && bot_state.BotState(state.State) == bot_state.AWAITING_POST_CONTENT && state.DealID != uuid.Nil {
		_, _ = b.states.SetState(c.Sender().ID, bot_state.AWAITING_POST_CONTENT, state.DealID, nil)

		locale := resolveLocale(c.Sender().LanguageCode)
		msg, err := b.messages.Render(locale, "dialog", "send_another_post", nil)
		if err != nil {
			b.log.Errorf("render send_another_post: %v", err)
		} else {
			_ = c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
		}
	} else {
		err = b.states.ClearState(c.Sender().ID)
		if err != nil {
			return errors.Wrap(err, "clear state")
		}
	}

	if c.Message() != nil {
		_ = c.Delete()
	}

	return b.onStart(c)
}
