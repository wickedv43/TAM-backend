package common

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common/constants/channel_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/channel"
)

type AddChannelRequest struct {
	ChannelUsername string `json:"channel_username"`
}

type AddChannelResponse struct {
	TgID      int64
	Username  string
	Title     string
	About     string
	PhotoHash string

	Subscribers        int
	PremiumSubscribers int
	PremiumPercent     float64

	FirstPostDate   time.Time
	TotalPosts      int
	MedianViews     int
	MedianForwards  int
	MedianReactions int
	AverageViews    int

	BoostLevel int
	Boosts     int

	Stats *ChannelStats

	StatsAvailable bool

	NotificationsOn float64

	UserRole string
}

type ChannelResponse struct {
	ID                 uuid.UUID                 `json:"id"`
	TgID               int64                     `json:"tg_id"`
	TgUsername         string                    `json:"tg_username"`
	TgName             string                    `json:"tg_name"`
	TgDescription      string                    `json:"tg_description"`
	TgPicture          string                    `json:"tg_picture"`
	Commentary         string                    `json:"commentary"`
	IsListed           bool                      `json:"is_listed"`
	Tags               []string                  `json:"tags"`
	StatsUpdatedAt     time.Time                 `json:"stats_updated_at"`
	Stats              *ChannelStats             `json:"stats,omitempty"`
	Prices             ChannelPrices             `json:"prices"`
	Subscribers        int                       `json:"subscribers"`
	PremiumSubscribers int                       `json:"premium_subscribers"`
	MedianPostViews    int                       `json:"median_post_views"`
	AvgPostViews       int                       `json:"avg_post_views"`
	Status             int                       `json:"status"`
	MainLanguage       string                    `json:"main_language"`
	RestrictedFrom     time.Time                 `json:"restricted_from"`
	RestrictedTill     time.Time                 `json:"restricted_till"`
	RestrictionReason  channel.RestrictionReason `json:"restriction_reason"`
	NotificationsOn    float64                   `json:"notifications_on"`
	FirstPostDate      time.Time                 `json:"first_post_date"`
	TotalPosts         int                       `json:"total_posts"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

func ChannelStatsToMap(stats *ChannelStats) map[string]interface{} {
	if stats == nil {
		return nil
	}

	var result map[string]interface{}
	if jsonData, err := json.Marshal(stats); err == nil {
		_ = json.Unmarshal(jsonData, &result)
	}

	return result
}

func mapStatsToChannelStats(statsMap map[string]interface{}) *ChannelStats {
	if statsMap == nil || len(statsMap) == 0 {
		return nil
	}

	var stats ChannelStats
	if jsonData, err := json.Marshal(statsMap); err == nil {
		if err := json.Unmarshal(jsonData, &stats); err == nil {
			return &stats
		}
	}

	return nil
}

func MapChannel(c *ent.Channel) *ChannelResponse {
	if c == nil {
		return nil
	}
	return &ChannelResponse{
		ID:                 c.ID,
		TgID:               c.TgID,
		TgUsername:         c.TgUsername,
		TgName:             c.TgName,
		TgDescription:      c.TgDescription,
		TgPicture:          c.TgPicture,
		Commentary:         c.Commentary,
		IsListed:           c.IsListed,
		Tags:               c.Tags,
		StatsUpdatedAt:     c.StatsUpdatedAt,
		Stats:              mapStatsToChannelStats(c.Stats),
		Prices:             *ChannelPricesFromMap(c.Prices),
		Subscribers:        c.Subscribers,
		PremiumSubscribers: c.PremiumSubscribers,
		MedianPostViews:    c.MedianPostViews,
		AvgPostViews:       c.AvgPostViews,
		Status:             c.Status,
		MainLanguage:       c.MainLanguage,
		RestrictedFrom:     c.RestrictedFrom,
		RestrictedTill:     c.RestrictedTill,
		RestrictionReason:  c.RestrictionReason,
		NotificationsOn:    c.NotificationsOn,
		FirstPostDate:      c.FirstPostDate,
		TotalPosts:         c.TotalPosts,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func MapChannels(channels []*ent.Channel) []*ChannelResponse {
	result := make([]*ChannelResponse, len(channels))
	for i, c := range channels {
		result[i] = MapChannel(c)
	}
	return result
}

type ChannelResponseWithoutStats struct {
	ID                 uuid.UUID                    `json:"id"`
	TgID               int64                        `json:"tg_id"`
	TgUsername         string                       `json:"tg_username"`
	TgName             string                       `json:"tg_name"`
	TgDescription      string                       `json:"tg_description"`
	TgPicture          string                       `json:"tg_picture"`
	Commentary         string                       `json:"commentary"`
	IsListed           bool                         `json:"is_listed"`
	Tags               []string                     `json:"tags"`
	StatsUpdatedAt     time.Time                    `json:"stats_updated_at"`
	Prices             ChannelPrices                `json:"prices"`
	Subscribers        int                          `json:"subscribers"`
	PremiumSubscribers int                          `json:"premium_subscribers"`
	MedianPostViews    int                          `json:"median_post_views"`
	AvgPostViews       int                          `json:"avg_post_views"`
	Status             channel_status.ChannelStatus `json:"status"`
	MainLanguage       string                       `json:"main_language"`
	RestrictedFrom     time.Time                    `json:"restricted_from"`
	RestrictedTill     time.Time                    `json:"restricted_till"`
	RestrictionReason  channel.RestrictionReason    `json:"restriction_reason"`
	NotificationsOn    float64                      `json:"notifications_on"`
	FirstPostDate      time.Time                    `json:"first_post_date"`
	TotalPosts         int                          `json:"total_posts"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`
}

// MapChannelWithoutStats maps ent.Channel to ChannelResponseWithoutStats.
func MapChannelWithoutStats(c *ent.Channel) *ChannelResponseWithoutStats {
	if c == nil {
		return nil
	}
	return &ChannelResponseWithoutStats{
		ID:                 c.ID,
		TgID:               c.TgID,
		TgUsername:         c.TgUsername,
		TgName:             c.TgName,
		TgDescription:      c.TgDescription,
		TgPicture:          c.TgPicture,
		Commentary:         c.Commentary,
		IsListed:           c.IsListed,
		Tags:               c.Tags,
		StatsUpdatedAt:     c.StatsUpdatedAt,
		Prices:             *ChannelPricesFromMap(c.Prices),
		Subscribers:        c.Subscribers,
		PremiumSubscribers: c.PremiumSubscribers,
		MedianPostViews:    c.MedianPostViews,
		AvgPostViews:       c.AvgPostViews,
		Status:             channel_status.ChannelStatus(c.Status),
		MainLanguage:       c.MainLanguage,
		RestrictedFrom:     c.RestrictedFrom,
		RestrictedTill:     c.RestrictedTill,
		RestrictionReason:  c.RestrictionReason,
		NotificationsOn:    c.NotificationsOn,
		FirstPostDate:      c.FirstPostDate,
		TotalPosts:         c.TotalPosts,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

// MapChannelsWithoutStats maps a slice of channels to ChannelResponseWithoutStats.
func MapChannelsWithoutStats(channels []*ent.Channel) []*ChannelResponseWithoutStats {
	result := make([]*ChannelResponseWithoutStats, len(channels))
	for i, c := range channels {
		result[i] = MapChannelWithoutStats(c)
	}
	return result
}

type GetChannelRequest struct {
	ChannelID uuid.UUID `json:"channel_id"`
}

type ProvideChannelDataRequest struct {
	ChannelID   uuid.UUID     `json:"channel_id"`
	Description string        `json:"channel_description"`
	Tags        []string      `json:"tags"`
	Language    string        `json:"language"`
	Prices      ChannelPrices `json:"prices"`
}

type ListChannelsRequest struct {
	PageOptions PageOptions `json:"page_options"`
	MyChannels  bool        `query:"my_channels" json:"my_channels"`
}

// @TODO: To generic. Current types parser can parse generics
// type ListChannelsResponse = Pagination[[]*ChannelResponseWithoutStats]
type ListChannelsResponse struct {
	Data  []*ChannelResponseWithoutStats `json:"data"`
	Count int                            `json:"count"`
}

type GetChannelResponse struct {
	CanEdit bool            `json:"canEdit"`
	Channel ChannelResponse `json:"channel"`
}

type CreateChannelResponse = ChannelResponseWithoutStats
