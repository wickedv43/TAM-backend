package telegram_bot

import (
	"context"
	"sync"
	"time"

	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/bot_state"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	tele "gopkg.in/telebot.v4"
)

const (
	answerMessage util.CallbackUnique = "answer"
	cancelMessage util.CallbackUnique = "cancel"
	acceptPost    util.CallbackUnique = "accept_post"
)

// onStart handles the /start command, creates or updates the customer, and generates a wallet if needed.
func (b *Bot) onStart(c tele.Context) error {
	ctx, cancel := context.WithTimeout(b.rootCtx, 10*time.Second)
	defer cancel()

	existingCustomer, err := b.customers.GetCustomerByTgID(ctx, c.Sender().ID)
	if err != nil {
		if !ent.IsNotFound(err) {
			b.log.Errorf("Error while checking if customer exists: %v", err)
			return err
		}

		customer := &ent.Customer{
			TgID:        c.Sender().ID,
			TgUsername:  c.Sender().Username,
			TgFirstname: c.Sender().FirstName,
			TgLastname:  c.Sender().LastName,
			TgLanguage:  c.Sender().LanguageCode,
			TgIsPremium: c.Sender().IsPremium,
			Status:      int(customer_status.PRE_REGISTRATION),
		}

		createdCustomer, err := b.customers.CreateCustomer(ctx, customer)
		if err != nil {
			b.log.Errorf("Error creating customer: %v", err)
			return err
		}

		existingCustomer, err = b.customers.GetCustomer(ctx, createdCustomer.ID)
		if err != nil {
			b.log.Errorf("Error fetching created customer: %v", err)
			return err
		}

		var addrBounceable, addrNonBounceable string
		addrBounceable, addrNonBounceable, err = b.ton.GetAddress(uint32(existingCustomer.WalletHdID))
		if err != nil {
			b.log.Errorf("Error getting address: %v", err)
			return err
		}

		existingCustomer, err = b.customers.UpdateCustomer(ctx, existingCustomer.ID, func(u *ent.CustomerUpdateOne) {
			u.SetAddressBounceable(addrBounceable)
			u.SetAddressNonbounceable(addrNonBounceable)
		})

		if err != nil {
			b.log.Errorf("Error updating customer addresses: %v", err)
			return err
		}
	}

	locale := resolveLocale(c.Sender().LanguageCode)
	msg, err := b.messages.Render(locale, "general", "start", nil)
	if err != nil {
		b.log.Errorf("render start message: %v", err)
		return err
	}

	openWebAppLabel := b.messages.RenderOrDefault(locale, "buttons", "open_web_app", nil, "Open Web App")
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.WebApp(openWebAppLabel, b.web),
		),
	)

	b.log.Debugf("\n\t\tUser: %d \n\t\tUUID: %s \n\t\tWebAppURL: %s", c.Sender().ID, existingCustomer.ID, b.web.URL)

	return c.EditOrSend(msg, markup, tele.ModeMarkdownV2)
}

// onText handles text messages and routes them based on the user's state.
func (b *Bot) onText(c tele.Context) error {
	stateEntity, err := b.states.GetState(c.Sender().ID)
	if err != nil {
		b.log.Errorf("get state error: %v", err)
		return err
	}

	currentState := bot_state.IDLE
	if stateEntity != nil {
		currentState = bot_state.BotState(stateEntity.State)
	}

	b.log.Debugf("USER: %d STATE: %s", c.Sender().ID, currentState.String())

	switch currentState {
	case bot_state.AWAITING_DIALOG_MESSAGE:
		if stateEntity == nil {
			b.log.Errorf("onText: stateEntity nil but currentState AWAITING_DIALOG_MESSAGE")
			break
		}
		return b.handleDialogMessage(c, stateEntity)

	case bot_state.AWAITING_POST_CONTENT:
		if stateEntity == nil {
			b.log.Errorf("onText: stateEntity nil but currentState AWAITING_POST_CONTENT")
			break
		}
		return b.handlePostContent(c, stateEntity)

	case bot_state.ADMIN_MODE:
		if stateEntity == nil {
			b.log.Errorf("onText: stateEntity nil but currentState ADMIN_MODE")
			break
		}

		return b.handleAdminMode(c)

	default:
		locale := resolveLocale(c.Sender().LanguageCode)
		var msg string
		msg, err = b.messages.Render(locale, "general", "default_menu", nil)
		if err != nil {
			b.log.Errorf("render default_menu message: %v", err)
			return err
		}

		openWebAppLabel := b.messages.RenderOrDefault(locale, "buttons", "open_web_app", nil, "Open Web App")
		markup := &tele.ReplyMarkup{}
		markup.Inline(
			markup.Row(
				markup.WebApp(openWebAppLabel, b.web),
			),
		)

		return c.EditOrSend(msg, markup, tele.ModeMarkdownV2)
	}

	return nil
}

// onMedia handles photo and video messages, collecting them into albums if needed.
func (b *Bot) onMedia(c tele.Context) error {
	userID := c.Sender().ID

	muInterface, _ := b.userMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInterface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	stateEntity, err := b.states.GetState(userID)
	if err != nil {
		return err
	}

	if stateEntity == nil || bot_state.BotState(stateEntity.State) != bot_state.AWAITING_POST_CONTENT {
		b.log.Warnf("unexpected media from user %d (state: %v)", userID, stateEntity)
		return nil
	}

	msgData, err := common.TgMsgDataFromMap(stateEntity.Data)
	if err != nil {
		msgData = &common.TgMsgData{}
	}

	msg := c.Message()
	albumID := msg.AlbumID

	var fileID string
	var mediaType string

	if msg.Photo != nil {
		fileID = msg.Photo.FileID
		mediaType = "photo"
	} else if msg.Video != nil {
		fileID = msg.Video.FileID
		mediaType = "video"
	} else {
		b.log.Warnf("unsupported media type from user %d", userID)
		return nil
	}

	mediaItem := common.TgMsgMediaItem{
		FileID:       fileID,
		MsgID:        msg.ID,
		Type:         mediaType,
		CaptionAbove: msg.CaptionAbove,
		HasSpoiler:   msg.HasMediaSpoiler,
	}

	if albumID != "" {
		if msgData.MessageAlbumID == albumID {
			msgData.MessageImages = append(msgData.MessageImages, mediaItem)
		} else {
			// New album, start fresh — clear caption/entities, preserve preview IDs for deletion
			msgData.MessageAlbumID = albumID
			msgData.MessageImages = []common.TgMsgMediaItem{mediaItem}
			msgData.MessageText = ""
			msgData.MessageEntities = nil
		}
	} else {
		msgData.MessageAlbumID = ""
		msgData.MessageImages = []common.TgMsgMediaItem{mediaItem}
	}

	if msg.Caption != "" {
		msgData.MessageText = msg.Caption
		msgData.MessageEntities = toInterfaceSlice(msg.CaptionEntities)
	}

	if msg.Caption != "" || len(msgData.MessageImages) == 1 {
		msgData.MessageCaptionAbove = msg.CaptionAbove
	}

	data, _ := msgData.ToMap()
	newState, err := b.states.SetState(userID, bot_state.AWAITING_POST_CONTENT, stateEntity.DealID, data)
	if err != nil {
		b.log.Errorf("failed to set state in onMedia: %v", err)
		return err
	}

	if albumID != "" {
		if t, loaded := b.albumTimers.Load(albumID); loaded {
			t.(*time.Timer).Stop()
		}

		timer := time.AfterFunc(1*time.Second, func() {
			b.albumTimers.Delete(albumID)
			finalState, err := b.states.GetState(userID)
			if err != nil {
				b.log.Errorf("failed to get state in album timer: %v", err)
				return
			}
			if finalState != nil {
				if err = b.showPost(&tele.User{ID: userID}, finalState); err != nil {
					b.log.Errorf("failed to show post in album timer: %v", err)
				}
			}
		})
		b.albumTimers.Store(albumID, timer)
		return nil
	}

	return b.showPost(c.Sender(), newState)
}

func (b *Bot) onMe(c tele.Context) error {
	userID := c.Sender().ID

	user, err := b.customers.GetCustomerByTgID(b.rootCtx, userID)
	if err != nil {
		return err
	}

	return c.Send(user.ID.String())
}
