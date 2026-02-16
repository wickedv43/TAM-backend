package stats_bot

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/modules/stats_bot/tg_api"
)

// validateChannelAccess checks channel access rights and returns the user role.
func (s *StatsBot) validateChannelAccess(ctx context.Context, channel *tg.InputChannel, userTgID int64) (string, error) {
	participants, err := s.tgClient.GetChannelAdmins(ctx, channel)
	if err != nil {
		return "", fmt.Errorf("get admins: %w", common.ErrChatAdminRequired)
	}

	admins := tg_api.ExtractAdminInfo(participants)

	var userRole string
	for _, admin := range admins {
		if admin.UserID == userTgID {
			if admin.IsCreator {
				userRole = "creator"
			} else {
				userRole = "administrator"
			}
			break
		}
	}

	if userRole == "" {
		return "", common.ErrUserNoRights
	}

	if s.cfg.Bot.ID != 0 {
		var found bool
		for _, admin := range admins {
			if admin.UserID == s.cfg.Bot.ID {
				found = true
				if !admin.CanPost {
					return "", common.ErrTextBotCannotPost
				}
				if !admin.CanEdit {
					return "", common.ErrTextBotCannotEdit
				}
				if !admin.CanDelete {
					return "", common.ErrTextBotCannotDelete
				}
				break
			}
		}
		if !found {
			return "", common.ErrTextBotNotAdded
		}
	}

	if s.cfg.UserBot.UserID != 0 {
		var found bool
		for _, admin := range admins {
			if admin.UserID == s.cfg.UserBot.UserID {
				found = true
				if !admin.CanPost {
					return "", common.ErrUserBotCannotPost
				}
				if !admin.CanEdit {
					return "", common.ErrUserBotCannotEdit
				}
				if !admin.CanDelete {
					return "", common.ErrUserBotCannotDelete
				}
				break
			}
		}
		if !found {
			return "", common.ErrUserBotNotAdded
		}
	}

	return userRole, nil
}

// validateChannelRequirements checks channel type, visibility, and subscriber count.
func (s *StatsBot) validateChannelRequirements(channel *tg.Channel, subscribers int) error {
	if channel == nil {
		return fmt.Errorf("channel object is nil")
	}

	if !tg_api.ValidateChannelType(channel) {
		s.log.Warn("Channel type is not broadcast, but only broadcast channels are supported")
		return common.ErrChannelNotBroadcast
	}

	if !tg_api.ValidateChannelIsPublic(channel) {
		s.log.Warn("Channel is private, but only public channels are supported")
		return common.ErrChannelNotPublic
	}

	if subscribers < config.MinChannelSubscribers {
		s.log.Warnf("Channel has %d subscribers, but minimum %d required", subscribers, config.MinChannelSubscribers)
		return common.ErrChannelNotEnoughMembers
	}

	return nil
}
