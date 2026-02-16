package common

import (
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_type"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

type DealMessageRequest struct {
	DealID uuid.UUID `json:"deal_id"`
}

type DealMessageBotRequest struct {
	DealID      uuid.UUID `json:"deal_id"`
	SenderID    uuid.UUID `json:"sender_id"`
	RecipientID uuid.UUID `json:"recipient_id"`
}

type SendDealDataRequest struct {
	DealID uuid.UUID `json:"deal_id"`
}

type SendDealDataBotRequest struct {
	DealID uuid.UUID `json:"deal_id"`
	UserID uuid.UUID `json:"user_id"`
}

type ShowDealDataBotRequest struct {
	DealID uuid.UUID `json:"deal_id"`
	UserID uuid.UUID `json:"user_id"`
}

type OfferDealRequest struct {
	ChannelID uuid.UUID `json:"channel_id"`
	// TODO: disabled on MVP. Always ADVERTISING_OFFER for now
	// Type             int                    `json:"type"`
	TargetType string `json:"target_type"`
}

type MakeDealRequest struct {
	DealID             uuid.UUID           `json:"deal_id"`
	PreferableDatetime *PreferableDatetime `json:"preferable_datetime"` // RFC3339
}

type ApproveAndProposeTimeRequest struct {
	DealID          uuid.UUID `json:"deal_id"`
	PublicationTime string    `json:"publication_time"` // RFC3339
}

type UpdateDealRequest struct {
	Status          *deal_status.DealStatus `json:"status"`
	ExpiresAt       *time.Time              `json:"expires_at"`
	Target          map[string]interface{}  `json:"target"`
	PublicationTime *time.Time              `json:"publication_time"`
}

type DealResponse struct {
	ID                   uuid.UUID              `json:"id"`
	ChannelID            uuid.UUID              `json:"channel_id"`
	AdvertiserCustomerID uuid.UUID              `json:"advertiser_customer_id"`
	ChannelManagerID     uuid.UUID              `json:"channel_manager_id"`
	Type                 deal_type.DealType     `json:"type"`
	Status               deal_status.DealStatus `json:"status"`
	StatusUpdatedAt      time.Time              `json:"status_updated_at"`
	ExpiresAt            time.Time              `json:"expires_at"`
	TargetType           string                 `json:"target_type"`
	Target               DealTarget             `json:"target"`
	PublicationTime      *time.Time             `json:"publication_time"`
	TopDeadline          *time.Time             `json:"top_deadline"`
	CommonDeadline       *time.Time             `json:"common_deadline"`
	TonPrice             float64                `json:"ton_price"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

type GetMyDealsResponse struct {
	AdvertiserDeals []*DealResponse `json:"advertiser_deals"`
	ManagerDeals    []*DealResponse `json:"manager_deals"`
}

func MapDeal(d *ent.Deal) *DealResponse {
	if d == nil {
		return nil
	}

	dTarget, err := DealTargetFromMap(d.Target)
	if err != nil {
		return nil
	}

	return &DealResponse{
		ID:                   d.ID,
		ChannelID:            d.ChannelID,
		AdvertiserCustomerID: d.AdvertiserCustomerID,
		ChannelManagerID:     d.ChannelManagerID,
		Type:                 deal_type.DealType(d.Type),
		Status:               deal_status.DealStatus(d.Status),
		StatusUpdatedAt:      d.StatusUpdatedAt,
		ExpiresAt:            d.ExpiresAt,
		TargetType:           d.TargetType,
		Target:               dTarget,
		PublicationTime:      d.PublicationTime,
		TopDeadline:          d.TopDeadline,
		CommonDeadline:       d.CommonDeadline,
		TonPrice:             d.TonPrice,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
}

func MapDealWithChannel(d *ent.Deal) *DealWithChannelResponse {
	if d == nil {
		return nil
	}

	dealResp := MapDeal(d)
	if dealResp == nil {
		return nil
	}

	channelResp := MapChannelWithoutStats(d.Edges.Channel)
	if channelResp == nil {
		return nil
	}

	return &DealWithChannelResponse{
		Deal:    *dealResp,
		Channel: *channelResp,
	}
}

func MapDealsWithChannel(deals []*ent.Deal) []*DealWithChannelResponse {
	result := make([]*DealWithChannelResponse, len(deals))
	for i, d := range deals {
		result[i] = MapDealWithChannel(d)
	}
	return result
}

func MapDeals(deals []*ent.Deal) []*DealResponse {
	result := make([]*DealResponse, len(deals))
	for i, d := range deals {
		result[i] = MapDeal(d)
	}
	return result
}

type DealRequest struct {
	DealID uuid.UUID `json:"deal_id"`
}

type ListDealsRequest struct {
	PageOptions PageOptions `json:"page_options"`
}

// @TODO: To generic. Current types parser can parse generics
// type ListDealsResponse = Pagination[[]*DealResponse]
type ListDealsResponse struct {
	Data  []*DealResponse `json:"data"`
	Count int             `json:"count"`
}

type DealWithChannelResponse struct {
	Deal    DealResponse                `json:"deal"`
	Channel ChannelResponseWithoutStats `json:"channel"`
}

type ListDealsWithChannelResponse struct {
	Data  []*DealWithChannelResponse `json:"data"`
	Count int                        `json:"count"`
}
