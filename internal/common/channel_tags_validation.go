package common

import (
	_ "embed"
	"encoding/json"
)

//go:embed channel-tags.json
var tagsJSON []byte

const (
	MIN_TAGS_COUNT = 1
	MAX_TAGS_COUNT = 5
)

type ChannelTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	tags   []ChannelTag
	tagIDs map[string]bool
)

func init() {
	json.Unmarshal(tagsJSON, &tags) //nolint:errcheck

	tagIDs = make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagIDs[tag.ID] = true
	}
}

func IsValidTag(tagID string) bool {
	return tagIDs[tagID]
}
