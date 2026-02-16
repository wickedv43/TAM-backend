package common

import (
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

// --- Brief ---
type BriefResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func MapBrief(b *ent.Brief) *BriefResponse {
	if b == nil {
		return nil
	}
	return &BriefResponse{
		ID:        b.ID,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func MapBriefs(briefs []*ent.Brief) []*BriefResponse {
	result := make([]*BriefResponse, len(briefs))
	for i, b := range briefs {
		result[i] = MapBrief(b)
	}
	return result
}
