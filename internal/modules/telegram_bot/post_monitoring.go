package telegram_bot

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
)

// startPostMonitoring runs a cron that checks published posts for edits/deletions.
func (b *Bot) startPostMonitoring(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	b.log.Info("Post monitoring started")

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Post monitoring stopped")
			return
		case <-ticker.C:
			b.checkPublishedPosts(ctx)
		}
	}
}

func (b *Bot) checkPublishedPosts(ctx context.Context) {
	deals, err := b.deals.GetDealsPublishedActive(ctx)
	if err != nil {
		b.log.Errorf("Failed to fetch published deals: %v", err)
		return
	}

	for _, d := range deals {
		violated, reason := b.checkPostIntegrity(ctx, d)
		if violated {
			b.updateDealStatus(ctx, d.ID, deal_status.TERMS_VIOLATED)
			// Escape parameters for formatted message
			b.sendDealNotification(d, "deal", "violated_edited", map[string]interface{}{
				"DealID": util.EscapeMarkdownV2(d.ID.String()),
			})

			b.log.Infof("Deal %s: TERMS_VIOLATED (%s)", d.ID, reason)
		}
	}
}

// checkPostIntegrity verifies the post exists and was not edited.
func (b *Bot) checkPostIntegrity(ctx context.Context, deal *ent.Deal) (bool, string) {
	if deal.Edges.Channel == nil {
		return false, ""
	}
	if b.resty == nil {
		return false, ""
	}

	ch := deal.Edges.Channel
	if ch.TgUsername == "" {
		return false, ""
	}

	if len(deal.ChannelPostIds) == 0 {
		return false, ""
	}

	exists, edited := b.checkPostViaTMe(ctx, ch.TgUsername, deal.ChannelPostIds[0])
	if !exists {
		return true, "post deleted or inaccessible"
	}
	if edited {
		return true, "post was edited"
	}
	return false, ""
}

// checkPostViaTMe fetches t.me/s/channel/postID and returns (exists, edited).
func (b *Bot) checkPostViaTMe(ctx context.Context, channelUsername string, postID int) (exists bool, edited bool) {
	username := strings.TrimPrefix(channelUsername, "@")
	url := fmt.Sprintf("https://t.me/s/%s/%d", username, postID)

	resp, err := b.resty.R().SetContext(ctx).Get(url)
	if err != nil {
		b.log.Warnf("checkPostViaTMe: %v", err)
		return false, false
	}
	body := resp.Body()

	postDataPost := []byte(`data-post="` + username + `/` + strconv.Itoa(postID) + `"`)
	idx := bytes.Index(body, postDataPost)
	if idx < 0 {
		return false, false
	}
	exists = true

	blockStart := idx
	nextBlock := bytes.Index(body[blockStart+len(postDataPost):], []byte(`data-post="`))
	blockEnd := len(body)
	if nextBlock >= 0 {
		blockEnd = blockStart + len(postDataPost) + nextBlock
	}
	block := body[blockStart:blockEnd]

	metaSpan := []byte(`tgme_widget_message_meta">`)
	metaIdx := bytes.Index(block, metaSpan)
	if metaIdx < 0 {
		return exists, false
	}
	afterMeta := block[metaIdx+len(metaSpan):]
	if len(afterMeta) < 6 {
		return exists, false
	}
	edited = bytes.HasPrefix(afterMeta, []byte("edited"))
	return exists, edited
}
