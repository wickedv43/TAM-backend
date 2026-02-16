package telegram_bot

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/pkg/errors"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/bot_state"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	tele "gopkg.in/telebot.v4"
)

// utf16StringLen returns the length of s in UTF-16 code units (as required by Telegram API).
func utf16StringLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// handlePostContent processes text messages for post creation.
func (b *Bot) handlePostContent(c tele.Context, state *ent.BotState) error {
	msg := c.Message()

	msgData := &common.TgMsgData{
		MessageText:     msg.Text,
		MessageEntities: toInterfaceSlice(msg.Entities),
	}

	data, err := msgData.ToMap()
	if err != nil {
		return errors.Wrap(err, "msgData to map")
	}

	newState, err := b.states.SetState(state.UserID, bot_state.AWAITING_POST_CONTENT, state.DealID, data)
	if err != nil {
		return errors.Wrap(err, "set state")
	}

	return b.showPost(c.Sender(), newState)
}

// showPost displays the current post preview with action buttons.
func (b *Bot) showPost(recipient tele.Recipient, state *ent.BotState) error {
	msgData, err := common.TgMsgDataFromMap(state.Data)
	if err != nil {
		return errors.Wrap(err, "parse msg data")
	}

	for _, idVal := range msgData.MessagePreviewIDs {
		oldID := getInt(idVal)
		if oldID > 0 {
			_ = b.tg.Delete(&tele.Message{
				ID:   oldID,
				Chat: &tele.Chat{ID: state.UserID},
			})
		}
	}

	var entities []tele.MessageEntity
	if msgData.MessageEntities != nil {
		bytes, _ := json.Marshal(msgData.MessageEntities)
		_ = json.Unmarshal(bytes, &entities)
	}

	utf16Len := utf16StringLen(msgData.MessageText)

	var validEntities []tele.MessageEntity
	for _, e := range entities {
		if e.Offset >= 0 && e.Offset < utf16Len && e.Offset+e.Length <= utf16Len {
			validEntities = append(validEntities, e)
		} else {
			b.log.Warnf("showPost: skipping invalid entity: offset=%d, length=%d, text_utf16_len=%d",
				e.Offset, e.Length, utf16Len)
		}
	}

	// Sort media by MsgID
	items := make([]common.TgMsgMediaItem, len(msgData.MessageImages))
	copy(items, msgData.MessageImages)
	sort.Slice(items, func(i, j int) bool { return items[i].MsgID < items[j].MsgID })

	locale := "en"
	if u, ok := recipient.(*tele.User); ok && u.LanguageCode != "" {
		locale = resolveLocale(u.LanguageCode)
	}
	btnConfirm := b.messages.RenderOrDefault(locale, "buttons", "confirm", nil, "✅ Confirm")
	btnEdit := b.messages.RenderOrDefault(locale, "buttons", "edit", nil, "⬅️ Edit")

	menu := &tele.ReplyMarkup{}
	btnAccept := menu.Data(btnConfirm, acceptPost.Unique())
	btnDecline := menu.Data(btnEdit, cancelMessage.Unique())
	menu.Inline(menu.Row(btnDecline, btnAccept))

	var mdText string
	var opts *tele.SendOptions
	useEntities := len(validEntities) > 0 && msgData.MessageText != ""
	if useEntities {
		mdText = msgData.MessageText
		opts = &tele.SendOptions{Entities: validEntities}
	} else {
		mdText = util.RenderMarkdownV2(msgData.MessageText, nil)
		opts = &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}
	}

	var sentMessages []interface{}

	switch len(items) {
	case 0:
		if msgData.MessageText == "" {
			return nil
		}
		sentMsg, err := b.tg.Send(recipient, mdText, opts)
		if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
			b.log.Warnf("showPost: entity error on text, retrying with MarkdownV2: %v", err)
			opts.Entities = nil
			opts.ParseMode = tele.ModeMarkdownV2
			sentMsg, err = b.tg.Send(recipient, util.RenderMarkdownV2(msgData.MessageText, entities), opts)
		}
		if err != nil {
			return err
		}
		sentMessages = append(sentMessages, sentMsg.ID)

		locale := resolveLocale("en")
		previewMsg, err := b.messages.Render(locale, "post", "preview", nil)
		if err != nil || previewMsg == "" {
			previewMsg = "🧐 *Preview your post*\nThis is how the post will look in the channel\\."
		}
		menuMsg, err := b.tg.Send(recipient, previewMsg, &tele.SendOptions{
			ParseMode:   tele.ModeMarkdownV2,
			ReplyMarkup: menu,
		})
		if err != nil {
			b.log.Errorf("failed to send preview menu for text-only post: %v", err)
			return err
		}
		sentMessages = append(sentMessages, menuMsg.ID)

	case 1:
		var msg *tele.Message
		var err error
		item := items[0]

		switch item.Type {
		case "video":
			video := &tele.Video{
				File:         tele.File{FileID: item.FileID},
				Caption:      mdText,
				CaptionAbove: msgData.MessageCaptionAbove,
			}
			if item.HasSpoiler {
				opts.HasSpoiler = true
			}
			msg, err = b.tg.Send(recipient, video, opts)
			if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
				b.log.Warnf("showPost: entity error on video, retrying with MarkdownV2: %v", err)
				video.Caption = util.RenderMarkdownV2(msgData.MessageText, entities)
				opts.Entities = nil
				opts.ParseMode = tele.ModeMarkdownV2
				msg, err = b.tg.Send(recipient, video, opts)
			}
		default:
			photo := &tele.Photo{
				File:         tele.File{FileID: item.FileID},
				Caption:      mdText,
				CaptionAbove: msgData.MessageCaptionAbove,
			}
			if item.HasSpoiler {
				opts.HasSpoiler = true
			}
			msg, err = b.tg.Send(recipient, photo, opts)
			if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
				b.log.Warnf("showPost: entity error on photo, retrying with MarkdownV2: %v", err)
				photo.Caption = util.RenderMarkdownV2(msgData.MessageText, entities)
				opts.Entities = nil
				opts.ParseMode = tele.ModeMarkdownV2
				msg, err = b.tg.Send(recipient, photo, opts)
			}
		}

		if err != nil {
			return err
		}
		sentMessages = append(sentMessages, msg.ID)

		locale := resolveLocale("en")
		previewMsg, err := b.messages.Render(locale, "post", "preview", nil)
		if err != nil || previewMsg == "" {
			previewMsg = "🧐 *Preview your post*\nThis is how the post will look in the channel\\."
		}
		menuMsg, err := b.tg.Send(recipient, previewMsg, &tele.SendOptions{
			ParseMode:   tele.ModeMarkdownV2,
			ReplyMarkup: menu,
		})
		if err != nil {
			b.log.Errorf("failed to send preview menu for single media: %v", err)
			return err
		}
		sentMessages = append(sentMessages, menuMsg.ID)

	default:
		var album tele.Album
		for i, item := range items {
			var media tele.Inputtable
			if item.Type == "video" {
				v := &tele.Video{File: tele.File{FileID: item.FileID}}
				v.CaptionAbove = msgData.MessageCaptionAbove
				v.HasSpoiler = item.HasSpoiler
				if i == 0 {
					v.Caption = mdText
				}
				media = v
			} else {
				p := &tele.Photo{File: tele.File{FileID: item.FileID}}
				p.CaptionAbove = msgData.MessageCaptionAbove
				p.HasSpoiler = item.HasSpoiler
				if i == 0 {
					p.Caption = mdText
				}
				media = p
			}
			album = append(album, media)
		}

		msgs, err := b.tg.SendAlbum(recipient, album, opts)
		if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
			b.log.Warnf("showPost: entity error on album, retrying with MarkdownV2: %v", err)
			mdFallback := util.RenderMarkdownV2(msgData.MessageText, entities)
			if p, ok := album[0].(*tele.Photo); ok {
				p.Caption = mdFallback
			} else if v, ok := album[0].(*tele.Video); ok {
				v.Caption = mdFallback
			}
			opts.Entities = nil
			opts.ParseMode = tele.ModeMarkdownV2
			msgs, err = b.tg.SendAlbum(recipient, album, opts)
		}
		if err != nil {
			return err
		}
		for _, m := range msgs {
			sentMessages = append(sentMessages, m.ID)
		}

		locale = resolveLocale("en")
		previewMsg, err := b.messages.Render(locale, "post", "preview", nil)
		if err != nil || previewMsg == "" {
			previewMsg = "🧐 *Preview your post*\nThis is how the post will look in the channel\\."
		}
		menuMsg, err := b.tg.Send(recipient, previewMsg, &tele.SendOptions{
			ParseMode:   tele.ModeMarkdownV2,
			ReplyMarkup: menu,
		})
		if err != nil {
			b.log.Errorf("failed to send preview menu for album: %v", err)
			return err
		}
		sentMessages = append(sentMessages, menuMsg.ID)
	}

	msgData.MessagePreviewIDs = sentMessages
	newData, _ := msgData.ToMap()
	_, _ = b.states.SetState(state.UserID, bot_state.AWAITING_POST_CONTENT, state.DealID, newData)

	return nil
}

// acceptPost finalizes the post creation, sends the content, and clears the state.
func (b *Bot) acceptPost(c tele.Context) error {
	state, err := b.states.GetState(c.Sender().ID)
	if err != nil {
		b.log.Errorf("acceptPost: get state: %v", err)
		return nil
	}
	if state == nil || state.Data == nil {
		b.log.Errorf("acceptPost: state lost for user %d", c.Sender().ID)
		return nil
	}

	msgData, err := common.TgMsgDataFromMap(state.Data)
	if err != nil {
		return errors.Wrap(err, "parse msg data")
	}

	for _, idVal := range msgData.MessagePreviewIDs {
		oldID := getInt(idVal)
		if oldID > 0 {
			_ = b.tg.Delete(&tele.Message{
				ID:   oldID,
				Chat: &tele.Chat{ID: state.UserID},
			})
		}
	}

	_, err = b.sendContent(c.Sender(), msgData)
	if err != nil {
		return errors.Wrap(err, "send content")
	}

	msgData.MessagePreviewIDs = nil
	msgData.MessageAlbumID = ""

	b.log.Debugf("acceptPost: msgData for save: text=%q, images=%d", msgData.MessageText, len(msgData.MessageImages))

	dealTarget := common.DealTarget{PostData: *msgData}
	if deal, err := b.deals.GetDeal(b.rootCtx, state.DealID); err == nil && deal.Target != nil {
		existing, err := common.DealTargetFromMap(deal.Target)
		if err == nil {
			dealTarget.BriefData = existing.BriefData
			dealTarget.StoryData = existing.StoryData
			dealTarget.RepostData = existing.RepostData
		}
	}

	targetMap, err := dealTarget.ToMap()
	if err != nil {
		b.log.Errorf("DealTarget.ToMap: %v", err)
		return nil
	}

	b.log.Debugf("acceptPost: targetMap saved: %v", targetMap)

	_, err = b.deals.UpdateDeal(b.rootCtx, state.DealID, func(u *ent.DealUpdateOne) {
		u.SetTarget(targetMap)
	})
	if err != nil {
		b.log.Errorf("update deal target: %v", err)
		return nil
	}

	err = b.states.ClearState(state.UserID)
	if err != nil {
		return errors.Wrap(err, "clear state")
	}

	locale := resolveLocale(c.Sender().LanguageCode)
	msg, err := b.messages.Render(locale, "post", "saved", nil)
	if err != nil {
		b.log.Errorf("render post saved: %v", err)
		return c.Send("Post saved!")
	}

	openDealLabel := b.messages.RenderOrDefault(locale, "buttons", "open_deal", nil, "📋 Open Deal")
	dealWeb := &tele.WebApp{URL: b.cfg.Bot.WebAppURL + "/deal/" + state.DealID.String()}
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.WebApp(openDealLabel, dealWeb),
		),
	)

	return c.Send(msg, markup, tele.ModeMarkdownV2)
}

// sendContent sends the final post content to the recipient.
func (b *Bot) sendContent(recipient tele.Recipient, msgData *common.TgMsgData) ([]int, error) {
	var entities []tele.MessageEntity
	if msgData.MessageEntities != nil {
		bytes, _ := json.Marshal(msgData.MessageEntities)
		_ = json.Unmarshal(bytes, &entities)
	}

	items := make([]common.TgMsgMediaItem, len(msgData.MessageImages))
	copy(items, msgData.MessageImages)
	sort.Slice(items, func(i, j int) bool { return items[i].MsgID < items[j].MsgID })

	utf16Len := utf16StringLen(msgData.MessageText)

	var validEntities []tele.MessageEntity
	for _, e := range entities {
		if e.Offset >= 0 && e.Offset < utf16Len && e.Offset+e.Length <= utf16Len {
			validEntities = append(validEntities, e)
		} else {
			b.log.Warnf("sendContent: skipping invalid entity: offset=%d, length=%d, text_utf16_len=%d",
				e.Offset, e.Length, utf16Len)
		}
	}

	var mdText string
	var opts *tele.SendOptions

	useEntities := len(validEntities) > 0 && msgData.MessageText != ""
	if useEntities {
		mdText = msgData.MessageText
		opts = &tele.SendOptions{Entities: validEntities}
		b.log.Infof("sendContent: using entities (custom emoji) — entities=%d, text_len=%d", len(validEntities), utf16Len)
	} else {
		mdText = util.RenderMarkdownV2(msgData.MessageText, nil)
		opts = &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}
		b.log.Infof("sendContent: using MarkdownV2 (no custom emoji) — raw_entities=%d, valid=%d, text_len=%d", len(entities), len(validEntities), utf16Len)
	}

	b.log.Debug("target map post?: ", msgData)
	switch len(items) {
	case 0:
		if msgData.MessageText == "" {
			return []int{0}, errors.New("message text is empty")
		}
		msg, err := b.tg.Send(recipient, mdText, opts)
		if err != nil {
			return []int{0}, errors.Wrap(err, "send message")
		}
		return []int{msg.ID}, nil

	case 1:
		item := items[0]
		switch item.Type {
		case "video":
			video := &tele.Video{
				File:         tele.File{FileID: item.FileID},
				Caption:      mdText,
				CaptionAbove: msgData.MessageCaptionAbove,
			}
			if item.HasSpoiler {
				opts.HasSpoiler = true
			}
			msg, err := b.tg.Send(recipient, video, opts)
			if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
				b.log.Warnf("entity error on video, retrying with MarkdownV2: %v", err)
				video.Caption = util.RenderMarkdownV2(msgData.MessageText, entities)
				opts.Entities = nil
				opts.ParseMode = tele.ModeMarkdownV2
				msg, err = b.tg.Send(recipient, video, opts)
			}
			if err != nil {
				return []int{0}, err
			}

			return []int{msg.ID}, nil
		default:
			photo := &tele.Photo{
				File:         tele.File{FileID: item.FileID},
				Caption:      mdText,
				CaptionAbove: msgData.MessageCaptionAbove,
			}
			if item.HasSpoiler {
				opts.HasSpoiler = true
			}
			msg, err := b.tg.Send(recipient, photo, opts)
			if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
				b.log.Warnf("entity error on photo, retrying with MarkdownV2: %v", err)
				photo.Caption = util.RenderMarkdownV2(msgData.MessageText, entities)
				opts.Entities = nil
				opts.ParseMode = tele.ModeMarkdownV2
				msg, err = b.tg.Send(recipient, photo, opts)
			}
			if err != nil {
				return []int{0}, err
			}

			return []int{msg.ID}, nil
		}

	default:
		var album tele.Album
		for i, item := range items {
			var media tele.Inputtable
			if item.Type == "video" {
				v := &tele.Video{File: tele.File{FileID: item.FileID}}
				v.CaptionAbove = msgData.MessageCaptionAbove
				if i == 0 {
					v.Caption = mdText
				}
				media = v
			} else {
				p := &tele.Photo{File: tele.File{FileID: item.FileID}}
				p.CaptionAbove = msgData.MessageCaptionAbove
				if i == 0 {
					p.Caption = mdText
				}
				media = p
			}
			album = append(album, media)
		}
		msg, err := b.tg.SendAlbum(recipient, album, opts)
		if err != nil && useEntities && strings.Contains(err.Error(), "entity begins after") {
			b.log.Warnf("entity error on album, retrying with MarkdownV2: %v", err)
			mdFallback := util.RenderMarkdownV2(msgData.MessageText, entities)
			if p, ok := album[0].(*tele.Photo); ok {
				p.Caption = mdFallback
			} else if v, ok := album[0].(*tele.Video); ok {
				v.Caption = mdFallback
			}
			opts.Entities = nil
			opts.ParseMode = tele.ModeMarkdownV2
			msg, err = b.tg.SendAlbum(recipient, album, opts)
		}
		if err != nil {
			return []int{0}, err
		}

		ids := make([]int, len(msg))
		for i, m := range msg {
			ids[i] = m.ID
		}
		return ids, nil
	}
}
