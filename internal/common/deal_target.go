package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PreferableDatetime describes preferred publication window.
// Date format: DD.MM.YYYY. Time format: HH.MM (00:00 - 24:00).
type PreferableDatetime struct {
	DateFrom string `json:"date_from,omitempty"`
	DateTo   string `json:"date_to,omitempty"`
	TimeFrom string `json:"time_from,omitempty"`
	TimeTo   string `json:"time_to,omitempty"`
}

type TgMsgMediaItem struct {
	FileID       string `json:"file_id"`
	MsgID        int    `json:"msg_id"`
	Type         string `json:"type"`
	CaptionAbove bool   `json:"caption_above"`
	HasSpoiler   bool   `json:"has_spoiler,omitempty"`
}

type TgMsgData struct {
	MessageText         string           `json:"message_text,omitempty"`
	MessageEntities     []interface{}    `json:"message_entities,omitempty"`
	MessageImages       []TgMsgMediaItem `json:"message_images,omitempty"`
	MessageCaptionAbove bool             `json:"message_caption_above,omitempty"`
	MessageAlbumID      string           `json:"message_album_id,omitempty"`
	MessagePreviewIDs   []interface{}    `json:"message_preview_ids,omitempty"`
	MessageDealID       string           `json:"message_deal_id,omitempty"`
}

type DealTarget struct {
	BriefData          TgMsgData           `json:"brief_data,omitempty"`
	PostData           TgMsgData           `json:"post_data,omitempty"`
	PreferableDatetime *PreferableDatetime `json:"preferable_datetime,omitempty"`
	// @TODO: Story and Repost
	StoryData  map[string]interface{} `json:"story_data,omitempty"`
	RepostData map[string]interface{} `json:"repost_data,omitempty"`
}

func (t *DealTarget) ToMap() (map[string]interface{}, error) {
	if t == nil {
		return nil, nil
	}
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func DealTargetFromMap(m map[string]interface{}) (DealTarget, error) {
	if m == nil {
		return DealTarget{}, nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return DealTarget{}, err
	}
	var t DealTarget
	if err = json.Unmarshal(data, &t); err != nil {
		return DealTarget{}, err
	}
	return t, nil
}

// ToMap converts TgMsgData to map for storing in state.Data or deal.Target.
func (t *TgMsgData) ToMap() (map[string]interface{}, error) {
	if t == nil {
		return nil, nil
	}
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TgMsgDataFromMap parses state.Data or deal.Target (map from jsonb) into TgMsgData.
func TgMsgDataFromMap(m map[string]interface{}) (*TgMsgData, error) {
	if m == nil {
		return &TgMsgData{}, nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var t TgMsgData
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

const (
	dateLayout = "02.01.2006" // DD.MM.YYYY
)

// ValidatePreferableDatetime validates p. Returns nil if p is nil or empty.
// date_from: must be >= tomorrow; date_to: must be >= date_from; time_from/time_to: HH.MM 00:00-24:00, time_to >= time_from.
func ValidatePreferableDatetime(p *PreferableDatetime) error {
	if p == nil {
		return nil
	}
	if p.DateFrom == "" && p.DateTo == "" && p.TimeFrom == "" && p.TimeTo == "" {
		return nil
	}

	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrowStart := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)

	if p.DateFrom != "" {
		dateFrom, err := time.Parse(dateLayout, p.DateFrom)
		if err != nil {
			return fmt.Errorf("date_from: invalid format, use DD.MM.YYYY")
		}
		dateFromStart := time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, time.UTC)
		if dateFromStart.Before(tomorrowStart) {
			return fmt.Errorf("date_from: must be not earlier than tomorrow")
		}

		if p.DateTo != "" {
			dateTo, err := time.Parse(dateLayout, p.DateTo)
			if err != nil {
				return fmt.Errorf("date_to: invalid format, use DD.MM.YYYY")
			}
			dateToStart := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 0, 0, 0, 0, time.UTC)
			if dateToStart.Before(dateFromStart) {
				return fmt.Errorf("date_to: must be not earlier than date_from")
			}
		}
	} else if p.DateTo != "" {
		return fmt.Errorf("date_from is required when date_to is provided")
	}

	parseTimeMM := func(s string) (minutes int, err error) {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid time format, use HH:MM")
		}
		h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid time format, use HH:MM")
		}
		m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid time format, use HH:MM")
		}
		if h < 0 || h > 24 || m < 0 || m > 59 {
			return 0, fmt.Errorf("time must be in range 00:00 - 24:00")
		}
		if h == 24 && m != 0 {
			return 0, fmt.Errorf("time must be in range 00:00 - 24:00")
		}
		return h*60 + m, nil
	}

	if p.TimeFrom != "" {
		if _, err := parseTimeMM(p.TimeFrom); err != nil {
			return fmt.Errorf("time_from: %w", err)
		}
	}
	if p.TimeTo != "" {
		if _, err := parseTimeMM(p.TimeTo); err != nil {
			return fmt.Errorf("time_to: %w", err)
		}
	}
	if p.TimeFrom != "" && p.TimeTo != "" {
		fromMin, _ := parseTimeMM(p.TimeFrom)
		toMin, _ := parseTimeMM(p.TimeTo)
		if toMin < fromMin {
			return fmt.Errorf("time_to must be >= time_from")
		}
	}

	return nil
}

func ValidatePublicationTime(publicationTime string) (time.Time, error) {
	if publicationTime == "" {
		return time.Time{}, fmt.Errorf("publication_time is required")
	}
	t, err := time.Parse(time.RFC3339, publicationTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("publication_time: invalid format, use RFC3339 (e.g. 2006-01-02T15:04:05Z07:00)")
	}
	minTime := time.Now().Add(time.Hour)
	if t.Before(minTime) {
		return time.Time{}, fmt.Errorf("publication_time: must be at least 1 hour from now")
	}
	return t, nil
}
