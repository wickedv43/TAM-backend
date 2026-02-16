package common

import "github.com/google/uuid"

type AutoPostRequest struct {
	DealID uuid.UUID `json:"deal_id"`
}

type StatsBotAddChannelRequest struct {
	ChannelUsername string `json:"channel_username"`
	UserTgID        int64
}
