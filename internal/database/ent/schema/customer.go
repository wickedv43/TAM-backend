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

const customersTableName = "customers"

// Customer holds the schema definition for the Customer entity.
type Customer struct {
	ent.Schema
}

func (Customer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: customersTableName},
	}
}

// Fields of the Customer.
func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Int64("tg_id").Unique(),
		field.String("tg_username").Optional(),
		field.String("tg_firstname").Optional(),
		field.String("tg_lastname").Optional(),
		field.String("tg_language").Optional(),
		field.String("tg_picture").Optional(),
		field.Bool("tg_is_premium").Default(false),
		field.Bool("tg_is_allow_pm").Default(false),
		field.Int64("referrer_tg_id").Optional(),
		field.Int64("wallet_hd_id").Unique().Optional(),
		field.String("address_bounceable").Optional(),
		field.String("address_nonbounceable").Optional(),
		field.Float("ton_balance").Default(0),
		field.Float("ton_balance_locked").Default(0),
		field.Int("status").Default(0)}
}

// Edges of the Customer.
func (Customer) Edges() []ent.Edge {
	return []ent.Edge{
		// Customer has many CustomerChannel rows (memberships).
		edge.To("customer_channels", CustomerChannel.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("customer_briefs", CustomerBrief.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("advertiser_deals", Deal.Type),
		edge.To("managed_deals", Deal.Type),
	}
}
