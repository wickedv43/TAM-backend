package telegram_bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

func parsePostDuration(s string) (time.Duration, error) {
	data := strings.Split(s, "_")
	if len(data) != 3 {
		return 24 * time.Hour, fmt.Errorf("invalid post duration: %s", s)
	}

	hours, err := strconv.Atoi(data[2])
	if err != nil {
		return 24 * time.Hour, fmt.Errorf("invalid post duration: %s", s)
	}

	duration := time.Duration(hours) * time.Hour

	return duration, nil
}

// getInt safely retrieves an int from interface{} (handles int, float64, string).
func getInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case int64:
		return int(val)
	case int32:
		return int(val)
	default:
		return 0
	}
}

// toInterfaceSlice converts []tele.MessageEntity to []interface{} for JSON storage.
func toInterfaceSlice(entities []tele.MessageEntity) []interface{} {
	if entities == nil {
		return nil
	}
	result := make([]interface{}, len(entities))
	for i, e := range entities {
		result[i] = e
	}
	return result
}
