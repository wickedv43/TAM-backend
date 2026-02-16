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

const dealsTableName = "deals"

// Deal holds the schema definition for the Deal entity.
type Deal struct {
	ent.Schema
}

// Annotations configures SQL table name for Deal.
func (Deal) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: dealsTableName},
	}
}

// Fields of the Deal.
func (Deal) Fields() []ent.Field {
	return []ent.Field{
		// Primary key
		field.UUID("id", uuid.UUID{}).
			Default(newUUID).
			Immutable(),

		// Timestamps
		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),

		// Relations (FKs)
		field.UUID("channel_id", uuid.UUID{}),

		field.UUID("advertiser_customer_id", uuid.UUID{}).
			Comment("Customer who buys advertising"),

		field.UUID("channel_manager_id", uuid.UUID{}).
			Comment("Customer managing the deal on channel side"),

		// Deal type
		field.Int("type").Default(0),

		// Deal status
		field.Int("status").Default(0),

		field.Time("status_updated_at").
			Default(time.Now),

		// Deadlines
		field.Time("expires_at"),

		// Target type: post_1_24, post_2_48, post_3_72
		field.String("target_type").Default("post_1_24"),

		field.JSON("target", map[string]any{}).
			Default(map[string]any{}),

		// Publication
		field.Time("publication_time").
			Optional().
			Nillable(),

		field.Time("top_deadline").
			Optional().
			Nillable(),

		field.Time("common_deadline").
			Optional().
			Nillable(),

		// Price
		field.Float("ton_price").
			Positive(),
		field.Bool("balance_refunded").Default(false),
		field.Ints("channel_post_ids").Optional(),
	}
}

// Edges of the Deal.
func (Deal) Edges() []ent.Edge {
	return []ent.Edge{
		// Deal belongs to Channel.
		edge.From("channel", Channel.Type).
			Ref("deals").
			Field("channel_id").
			Required().
			Unique(),

		// Advertiser (buyer).
		edge.From("advertiser", Customer.Type).
			Ref("advertiser_deals").
			Field("advertiser_customer_id").
			Required().
			Unique(),

		// Channel manager (admin).
		edge.From("channel_manager", Customer.Type).
			Ref("managed_deals").
			Field("channel_manager_id").
			Required().
			Unique(),
	}
}
