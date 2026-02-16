package server

import (
	"context"
	"time"

	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
)

func (s *Server) startDealPostPublisher(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.log.Info("deal auto-post-publisher started")

	for {
		select {
		case <-ctx.Done():
			s.log.Info("deal auto-post-publisher stopped")
			return

		case <-ticker.C:
			s.checkAndExpireDeals(ctx)
			s.checkAndCompletePublishedDeals(ctx)
			s.checkAndPublishDeals(ctx)
		}
	}
}

func (s *Server) checkAndExpireDeals(ctx context.Context) {
	deals, err := s.deals.GetExpiredDeals(ctx)
	if err != nil {
		s.log.Errorf("Failed to fetch expired deals: %v", err)
		return
	}

	//todo: [POST-MVP] need concurrency
	now := time.Now()
	for _, d := range deals {
		_, err = s.deals.UpdateDeal(ctx, d.ID, func(u *ent.DealUpdateOne) {
			u.SetStatus(int(deal_status.EXPIRED))
			u.SetStatusUpdatedAt(now)
		})
		if err != nil {
			s.log.Errorf("Failed to mark deal %s as expired: %v", d.ID, err)
			continue
		}
		s.log.Infof("Deal %s marked as expired (expires_at was %s)", d.ID, d.ExpiresAt)
	}
}

func (s *Server) checkAndPublishDeals(ctx context.Context) {
	deals, err := s.deals.GetDealsByStatus(ctx, int(deal_status.AWAITING_PUBLICATION))
	if err != nil {
		s.log.Errorf("Failed to fetch pending deals: %v", err)
		return
	}

	//todo: [POST-MVP] need concurrency
	now := time.Now()
	for _, d := range deals {
		if d.PublicationTime == nil || d.PublicationTime.After(now) {
			continue
		}

		s.log.Infof("Scheduling auto-post for deal %s (publication time: %s)",
			d.ID, d.PublicationTime)

		_, err = s.deals.UpdateDeal(ctx, d.ID, func(u *ent.DealUpdateOne) {
			u.SetStatus(int(deal_status.PUBLISHED))
			u.SetStatusUpdatedAt(now)
			u.SetExpiresAt(now.Add(24 * time.Hour))
		})
		if err != nil {
			s.log.Errorf("Failed to claim deal %s for publish: %v", d.ID, err)
			continue
		}

		select {
		case s.bridge.AutoPost <- d.ID:
			s.log.Debugf("Deal %s sent to autopost channel", d.ID)
		default:
			s.log.Warnf("AutoPost channel is full, reverting deal %s to AWAITING_PUBLICATION", d.ID)
			_, _ = s.deals.UpdateDeal(ctx, d.ID, func(u *ent.DealUpdateOne) {
				u.SetStatus(int(deal_status.AWAITING_PUBLICATION))
				u.SetStatusUpdatedAt(now)
			})
		}
	}
}

func (s *Server) checkAndCompletePublishedDeals(ctx context.Context) {
	deals, err := s.deals.GetDealsNeedToComplete(ctx)
	if err != nil {
		s.log.Errorf("Failed to fetch published expired deals: %v", err)
		return
	}

	now := time.Now()
	for _, d := range deals {
		if err = s.customers.TransferBalanceToManagerForDeal(ctx, d); err != nil {
			s.log.Errorf("Failed to transfer balance for deal %s: %v", d.ID, err)
			continue
		}

		_, err = s.deals.UpdateDeal(ctx, d.ID, func(u *ent.DealUpdateOne) {
			u.SetStatus(int(deal_status.COMPLETED))
			u.SetStatusUpdatedAt(now)
		})

		if err != nil {
			s.log.Errorf("Failed to mark deal %s as completed: %v", d.ID, err)
			continue
		}
		s.log.Infof("Deal %s completed, balance transferred to manager", d.ID)

		if d.Edges.Advertiser != nil && d.Edges.ChannelManager != nil {
			channelName := "—"
			if d.Edges.Channel != nil && d.Edges.Channel.TgUsername != "" {
				channelName = "@" + d.Edges.Channel.TgUsername
			} else if d.Edges.Channel != nil {
				channelName = d.Edges.Channel.TgName
			}
			params := map[string]interface{}{
				"DealID":      util.EscapeMarkdownV2(d.ID.String()),
				"ChannelName": util.EscapeMarkdownV2(channelName),
			}
			webAppURL := s.cfg.Bot.WebAppURL + "/deal/" + d.ID.String()

			for _, r := range []struct {
				tgID   int64
				locale string
			}{
				{d.Edges.Advertiser.TgID, d.Edges.Advertiser.TgLanguage},
				{d.Edges.ChannelManager.TgID, d.Edges.ChannelManager.TgLanguage},
			} {
				locale := r.locale
				if locale == "" {
					locale = "en"
				}
				var text string
				text, err = s.messages.Render(locale, "notification", "deal_completed", params)
				if err != nil {
					s.log.Errorf("render deal_completed: %v", err)
					text = "Deal completed. Payment transferred. You can delete the post."
				}

				s.bridge.Notification <- common.Notification{
					UserID:    r.tgID,
					Text:      text,
					WebAppURL: webAppURL,
				}
			}
		}
	}
}
