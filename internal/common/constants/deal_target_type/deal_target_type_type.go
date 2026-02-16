package deal_target_type

import (
	"fmt"
	"strconv"
	"strings"
)

// DealTargetType represents the type of deal target (post duration)
type DealTargetType string

// Enum is an alias for DealTargetType to maintain compatibility with generated enum code
type Enum = DealTargetType

const (
	POST_1_24 DealTargetType = "post_1_24"
	POST_2_48 DealTargetType = "post_2_48"
	POST_3_72 DealTargetType = "post_3_72"
)

// String returns the string representation of DealTargetType
func (t DealTargetType) String() string {
	return string(t)
}

// TSName returns the TypeScript enum name for typescriptify
func (t DealTargetType) TSName() string {
	return string(t)
}

// IsValid checks if the DealTargetType is valid
func (t DealTargetType) IsValid() bool {
	switch t {
	case POST_1_24, POST_2_48, POST_3_72:
		return true
	}
	return false
}

// DealTargetTypeValues returns all valid DealTargetType values
func DealTargetTypeValues() []DealTargetType {
	return []DealTargetType{POST_1_24, POST_2_48, POST_3_72}
}

// ParseDealTargetType converts a string to DealTargetType
func ParseDealTargetType(s string) (DealTargetType, bool) {
	t := DealTargetType(s)
	return t, t.IsValid()
}

// "<type>_<topHours>_<commonHours>" (e.g. "post_1_24").
func ParseDealTargetTypeToParams(targetType string) (string, int, int, error) {
	parts := strings.Split(targetType, "_")
	if len(parts) != 3 {
		return "", 0, 0, fmt.Errorf("expected 3 parts, got %d", len(parts))
	}
	typ := parts[0]
	topHours, err := strconv.Atoi(parts[1])
	if err != nil || topHours < 0 {
		return "", 0, 0, fmt.Errorf("invalid top hours %q", parts[1])
	}
	commonHours, err := strconv.Atoi(parts[2])
	if err != nil || commonHours < 0 {
		return "", 0, 0, fmt.Errorf("invalid common hours %q", parts[2])
	}
	return typ, topHours, commonHours, nil
}
