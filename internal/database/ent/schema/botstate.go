package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type BotState struct {
	ent.Schema
}

func (BotState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique(),

		field.UUID("deal_id", uuid.UUID{}),

		field.Int("state").Default(0),

		field.JSON("data", map[string]interface{}{}).
			Optional(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (BotState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}
