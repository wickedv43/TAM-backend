package common

import (
	"errors"
	"net/http"
)

// Domain Errors (Graphs Bot)
var (
	ErrChannelNotFound         = errors.New("channel not found or user bot not added")
	ErrChatAdminRequired       = errors.New("user bot needs admin rights in channel")
	ErrUserNoRights            = errors.New("user is not an admin or creator of the channel")
	ErrTextBotNotAdded         = errors.New("text bot is not added to channel")
	ErrTextBotNoRights         = errors.New("text bot lacks required permissions")
	ErrUserBotNotAdded         = errors.New("user bot not added to channel")
	ErrChannelPrivate          = errors.New("channel is private or inaccessible")
	ErrStatsUnavailable        = errors.New("channel statistics are not available")
	ErrTextBotCannotPost       = errors.New("text bot cannot post messages")
	ErrTextBotCannotEdit       = errors.New("text bot cannot edit messages")
	ErrTextBotCannotDelete     = errors.New("text bot cannot delete messages")
	ErrUserBotCannotPost       = errors.New("user bot cannot post messages")
	ErrUserBotCannotEdit       = errors.New("user bot cannot edit messages")
	ErrUserBotCannotDelete     = errors.New("user bot cannot delete messages")
	ErrChannelNotBroadcast     = errors.New("only broadcast channels are supported")
	ErrChannelNotPublic        = errors.New("only public channels are supported")
	ErrChannelNotEnoughMembers = errors.New("channel must have at least 50 subscribers")
)

// HTTP Layer Errors
var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrChannelExists        = errors.New("channel already exists")
	ErrChannelStatusInvalid = errors.New("invalid channel status for operation")
	ErrInternalError        = errors.New("internal server error")
	ErrNotFound             = errors.New("not found")
	ErrDealNotFound         = errors.New("deal not found")
	ErrNotDealParticipant   = errors.New("you are not a participant of this deal")
	ErrNoChannelAccess      = errors.New("you don't have access to this channel")
	ErrInvalidUUID          = errors.New("invalid UUID format")
)

// Transaction Errors
var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrWalletNotLinked     = errors.New("wallet not linked")
)

// GetHTTPStatus maps domain errors to HTTP status codes
func GetHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrChannelNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrChatAdminRequired):
		return http.StatusForbidden
	case errors.Is(err, ErrUserNoRights):
		return http.StatusForbidden
	case errors.Is(err, ErrTextBotNotAdded):
		return http.StatusBadRequest
	case errors.Is(err, ErrTextBotNoRights):
		return http.StatusBadRequest
	case errors.Is(err, ErrUserBotNotAdded):
		return http.StatusBadRequest
	case errors.Is(err, ErrChannelPrivate):
		return http.StatusForbidden
	case errors.Is(err, ErrStatsUnavailable):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrChannelExists):
		return http.StatusConflict
	case errors.Is(err, ErrChannelStatusInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrInternalError):
		return http.StatusInternalServerError
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrTextBotCannotPost):
		return http.StatusBadRequest
	case errors.Is(err, ErrTextBotCannotEdit):
		return http.StatusBadRequest
	case errors.Is(err, ErrTextBotCannotDelete):
		return http.StatusBadRequest
	case errors.Is(err, ErrChannelNotBroadcast):
		return http.StatusBadRequest
	case errors.Is(err, ErrChannelNotPublic):
		return http.StatusBadRequest
	case errors.Is(err, ErrChannelNotEnoughMembers):
		return http.StatusBadRequest
	case errors.Is(err, ErrDealNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrNotDealParticipant):
		return http.StatusForbidden
	case errors.Is(err, ErrNoChannelAccess):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidUUID):
		return http.StatusBadRequest
	case errors.Is(err, ErrInsufficientBalance):
		return http.StatusForbidden
	case errors.Is(err, ErrWalletNotLinked):
		return http.StatusBadRequest
	case errors.Is(err, ErrUserBotCannotPost):
		return http.StatusBadRequest
	case errors.Is(err, ErrUserBotCannotEdit):
		return http.StatusBadRequest
	case errors.Is(err, ErrUserBotCannotDelete):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
