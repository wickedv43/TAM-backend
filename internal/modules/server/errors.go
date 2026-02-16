package server

import (
	"errors"
	"fmt"

	"github.com/wickedv43/TAM-backend/internal/common"
)

// GetUserMessage returns user-friendly messages for errors, with bot nicknames from config
func (s *Server) GetUserMessage(err error) string {
	textBot := s.cfg.Bot.Username
	if textBot == "" {
		textBot = "Text bot"
	}
	userBot := s.cfg.UserBot.Username
	if userBot == "" {
		userBot = "User bot"
	}

	switch {
	case errors.Is(err, common.ErrChannelNotFound):
		return fmt.Sprintf("Channel not found or %s not added", userBot)
	case errors.Is(err, common.ErrChatAdminRequired):
		return fmt.Sprintf("%s needs admin rights in channel", userBot)
	case errors.Is(err, common.ErrUserNoRights):
		return "You must be an admin or creator of this channel"
	case errors.Is(err, common.ErrTextBotNotAdded):
		return fmt.Sprintf("%s is not added to the channel", textBot)
	case errors.Is(err, common.ErrTextBotNoRights):
		return fmt.Sprintf("%s lacks required permissions", textBot)
	case errors.Is(err, common.ErrUserBotNotAdded):
		return fmt.Sprintf("%s is not added to the channel", userBot)
	case errors.Is(err, common.ErrChannelPrivate):
		return "Channel is private or inaccessible"
	case errors.Is(err, common.ErrStatsUnavailable):
		return "Channel statistics are not available"
	case errors.Is(err, common.ErrUnauthorized):
		return "Unauthorized"
	case errors.Is(err, common.ErrForbidden):
		return "Forbidden"
	case errors.Is(err, common.ErrInvalidRequest):
		return "Invalid request"
	case errors.Is(err, common.ErrChannelExists):
		return "Channel already exists"
	case errors.Is(err, common.ErrChannelStatusInvalid):
		return "Invalid channel status for operation"
	case errors.Is(err, common.ErrInternalError):
		return "Internal server error"
	case errors.Is(err, common.ErrTextBotCannotPost):
		return fmt.Sprintf("%s needs permission to post messages", textBot)
	case errors.Is(err, common.ErrTextBotCannotEdit):
		return fmt.Sprintf("%s needs permission to edit messages", textBot)
	case errors.Is(err, common.ErrTextBotCannotDelete):
		return fmt.Sprintf("%s needs permission to delete messages", textBot)
	case errors.Is(err, common.ErrChannelNotBroadcast):
		return "Only broadcast channels are supported. Groups and supergroups are not allowed."
	case errors.Is(err, common.ErrChannelNotPublic):
		return "Only public channels with username are supported"
	case errors.Is(err, common.ErrChannelNotEnoughMembers):
		return "Channel must have at least 50 subscribers to access statistics"
	case errors.Is(err, common.ErrDealNotFound):
		return "Deal not found"
	case errors.Is(err, common.ErrNotDealParticipant):
		return "You are not a participant of this deal"
	case errors.Is(err, common.ErrNoChannelAccess):
		return "You don't have access to this channel"
	case errors.Is(err, common.ErrInvalidUUID):
		return "Invalid UUID format"
	case errors.Is(err, common.ErrInsufficientBalance):
		return "Insufficient balance"
	case errors.Is(err, common.ErrWalletNotLinked):
		return "Wallet not linked"
	case errors.Is(err, common.ErrUserBotCannotPost):
		return fmt.Sprintf("%s needs permission to post messages", userBot)
	case errors.Is(err, common.ErrUserBotCannotEdit):
		return fmt.Sprintf("%s needs permission to edit messages", userBot)
	case errors.Is(err, common.ErrUserBotCannotDelete):
		return fmt.Sprintf("%s needs permission to delete messages", userBot)
	default:
		return "Internal server error"
	}
}
