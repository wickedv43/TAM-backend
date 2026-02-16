package common

// ChannelPrices represents pricing options for a channel.
// Post_1_24 is required, others are optional.
type ChannelPrices struct {
	Post_1_24 float64 `json:"post_1_24"`
	Post_2_48 float64 `json:"post_2_48,omitempty"`
	Post_3_72 float64 `json:"post_3_72,omitempty"`
}

// GetPrice returns the price for the given target type.
// Returns 0 if target type is invalid or price is not set.
func (p *ChannelPrices) GetPrice(targetType string) float64 {
	if p == nil {
		return 0
	}
	switch targetType {
	case "post_1_24":
		return p.Post_1_24
	case "post_2_48":
		return p.Post_2_48
	case "post_3_72":
		return p.Post_3_72
	default:
		return 0
	}
}

// ToMap converts ChannelPrices to map[string]float64 for storing in ent Channel.Prices.
func (p *ChannelPrices) ToMap() map[string]float64 {
	if p == nil {
		return nil
	}
	result := make(map[string]float64)
	result["post_1_24"] = p.Post_1_24
	if p.Post_2_48 > 0 {
		result["post_2_48"] = p.Post_2_48
	}
	if p.Post_3_72 > 0 {
		result["post_3_72"] = p.Post_3_72
	}
	return result
}

// ChannelPricesFromMap parses Channel.Prices (map[string]float64) into ChannelPrices.
func ChannelPricesFromMap(m map[string]float64) *ChannelPrices {
	if m == nil {
		return &ChannelPrices{}
	}
	return &ChannelPrices{
		Post_1_24: m["post_1_24"],
		Post_2_48: m["post_2_48"],
		Post_3_72: m["post_3_72"],
	}
}
