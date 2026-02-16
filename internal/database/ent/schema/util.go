package schema

import "github.com/google/uuid"

func newUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 if V7 fails (unlikely) or panic
		return uuid.New()
	}
	return id
}
