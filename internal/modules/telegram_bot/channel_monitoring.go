package telegram_bot

import (
	"fmt"

	"github.com/wickedv43/TAM-backend/internal/common/constants/channel_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_target_type"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	tele "gopkg.in/telebot.v4"
)

// onMyChatMember handles bot add/remove events in chats.
func (b *Bot) onMyChatMember(c tele.Context) error {
	c.Sender()
	member := c.ChatMember()
	if member == nil {
		return nil
	}

	chat := member.Chat
	newStatus := member.NewChatMember.Role
	oldStatus := member.OldChatMember.Role

	if (oldStatus == tele.Left || oldStatus == tele.Kicked) &&
		(newStatus == tele.Administrator || newStatus == tele.Creator || newStatus == tele.Member) {
		b.log.Debugw("Bot added to chat", "chat_id", chat.ID, "chat_type", chat.Type, "title", chat.Title)
	}

	if (oldStatus == tele.Administrator || oldStatus == tele.Creator || oldStatus == tele.Member) &&
		(newStatus == tele.Left || newStatus == tele.Kicked) {
		b.log.Infow("Bot removed from chat", "chat_id", chat.ID, "title", chat.Title)
		b.handleBotRemovedFromChat(chat.ID)
	}

	return nil
}

func (b *Bot) handleBotRemovedFromChat(chatID int64) {
	ctx := b.rootCtx

	ch, err := b.channels.GetChannelByTgID(ctx, chatID)
	if err != nil {
		if ent.IsNotFound(err) {
			return
		}
		b.log.Errorf("handleBotRemovedFromChat: get channel: %v", err)
		return
	}

	_, err = b.channels.UpdateChannel(ctx, ch.ID, func(u *ent.ChannelUpdateOne) {
		u.SetStatus(int(channel_status.UNACTIVATED))
	})
	if err != nil {
		b.log.Errorf("handleBotRemovedFromChat: update channel status: %v", err)
	}

	deals, err := b.deals.GetDealsByChannelExcludingStatuses(ctx, ch.ID)
	if err != nil {
		b.log.Errorf("handleBotRemovedFromChat: get deals: %v", err)
		return
	}

	for _, d := range deals {
		b.updateDealStatus(ctx, d.ID, deal_status.TERMS_VIOLATED)
	}

	b.log.Debugf("handleBotRemovedFromChat: channel %s (tg_id=%d), status=UNACTIVATED, %d deals marked as TERMS_VIOLATED",
		ch.ID, chatID, len(deals))
}

func (b *Bot) onChannelPost(c tele.Context) error {
	defer func() {
		if r := recover(); r != nil {
			b.log.Errorf("PANIC in onChannelPost: %v", r)
		}
	}()

	msg := c.Message()
	if msg == nil {
		return nil
	}

	chat := c.Chat()
	if chat == nil {
		return nil
	}

	ctx := b.rootCtx
	ch, err := b.channels.GetChannelByTgID(ctx, chat.ID)
	if err != nil || ch == nil {
		return nil
	}

	deals, err := b.deals.GetPublishedDealsWithinTopHoursByChannel(ctx, ch.ID)
	if err != nil {
		return nil
	}

	ourPostIDs := make(map[int]bool)
	for _, deal := range deals {
		if deal == nil {
			continue
		}
		for _, id := range deal.ChannelPostIds {
			ourPostIDs[id] = true
		}
	}

	if ourPostIDs[msg.ID] {
		b.log.Debugf("onChannelPost: msg.ID=%d is our post, ignoring", msg.ID)
		return nil
	}

	for _, deal := range deals {
		if deal == nil || deal.Edges.Advertiser == nil || deal.Edges.ChannelManager == nil {
			b.log.Warnf("onChannelPost: deal %v has nil edges, skipping", deal)
			continue
		}

		var topHours int
		_, topHours, _, err = deal_target_type.ParseDealTargetTypeToParams(deal.TargetType)
		if err != nil {
			continue
		}

		b.log.Infof("Deal %s: TERMS_VIOLATED (new post within %d hours)", deal.ID, topHours)

		b.updateDealStatus(ctx, deal.ID, deal_status.TERMS_VIOLATED)

		b.sendDealNotification(deal, "deal", "violated_posted", map[string]interface{}{
			"DealID":    util.EscapeMarkdownV2(deal.ID.String()),
			"LockHours": util.EscapeMarkdownV2(fmt.Sprintf("%d", topHours)),
		})
	}
	return nil
}
