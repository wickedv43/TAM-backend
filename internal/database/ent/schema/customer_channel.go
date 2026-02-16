package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

const customersChannelsTableName = "customers_channels"

// CustomerChannel (Customer <-> Channel)
type CustomerChannel struct {
	ent.Schema
}

func (CustomerChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: customersChannelsTableName},
	}
}

func (CustomerChannel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),

		// Timestamps
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),

		// Side keys
		field.UUID("customer_id", uuid.UUID{}),
		field.UUID("channel_id", uuid.UUID{}),

		// Role
		field.Int("role").Default(0),

		// Permissions
		field.Ints("permissions").Optional(),
	}
}

// Edges of the CustomerChannel.
func (CustomerChannel) Edges() []ent.Edge {
	return []ent.Edge{
		// CustomerChannel belongs to a Customer.
		edge.From("customer", Customer.Type).
			Ref("customer_channels").
			Field("customer_id").
			Unique().
			Required(),

		// CustomerChannel belongs to a Channel.
		edge.From("channel", Channel.Type).
			Ref("customer_channels").
			Field("channel_id").
			Unique().
			Required(),
	}
}
