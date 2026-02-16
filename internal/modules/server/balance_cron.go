package server

import (
	"context"
	"fmt"
	"time"

	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
)

func (s *Server) startRefunder(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	s.log.Info("balance refunder started")

	for {
		select {
		case <-ctx.Done():
			s.log.Info("balance refunder stopped")
			return

		case <-ticker.C:
			s.checkAndRefundTerminalDeals(ctx)
		}
	}
}

func (s *Server) checkAndRefundTerminalDeals(ctx context.Context) {
	deals, err := s.deals.GetDealsForRefund(ctx)
	if err != nil {
		s.log.Errorf("Failed to fetch terminal deals for refund: %v", err)
		return
	}
	var notification common.Notification

	for _, d := range deals {
		if err = s.customers.UnlockBalanceForDeal(ctx, d); err != nil {
			s.log.Errorf("Failed to refund deal %s: %v", d.ID, err)
			continue
		}
		if _, err = s.deals.UpdateDeal(ctx, d.ID, func(u *ent.DealUpdateOne) {
			u.SetBalanceRefunded(true)
		}); err != nil {
			s.log.Errorf("Failed to mark deal %s as refunded: %v", d.ID, err)
			continue
		}

		// Render refund message with spoiler
		locale := "en"
		if d.Edges.Advertiser != nil && d.Edges.Advertiser.TgLanguage != "" {
			locale = d.Edges.Advertiser.TgLanguage
		}

		// Escape parameters manually since we're using Render() not RenderMDV2()
		dealIDEscaped := util.EscapeMarkdownV2(d.ID.String())
		statusStr := util.EscapeMarkdownV2(deal_status.DealStatus(d.Status).String())
		amountSpoiler := messages.WrapSpoiler(fmt.Sprintf("%.2f TON", d.TonPrice))

		var text string
		text, err = s.messages.Render(locale, "notification", "refund", map[string]interface{}{
			"DealID":        dealIDEscaped,
			"Status":        statusStr,
			"AmountSpoiler": amountSpoiler,
		})
		if err != nil {
			s.log.Errorf("render refund message: %v", err)
			text = fmt.Sprintf("Deal %s refunded: %.2f TON", d.ID, d.TonPrice)
		}

		notification.UserID = d.Edges.Advertiser.TgID
		notification.Text = text
		notification.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + d.ID.String()
		s.bridge.Notification <- notification

		s.log.Debugf("Refunded balance for deal %s (status=%s)", d.ID, deal_status.DealStatus(d.Status).String())
	}
}
