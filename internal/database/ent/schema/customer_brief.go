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

const customersBriefsTableName = "customers_briefs"

// CustomerBrief holds the schema definition for the CustomerBrief entity.
type CustomerBrief struct {
	ent.Schema
}

func (CustomerBrief) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: customersBriefsTableName},
	}
}

// Fields of the CustomerBrief.
func (CustomerBrief) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),

		// Timestamps
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),

		// Side keys
		field.UUID("customer_id", uuid.UUID{}),
		field.UUID("brief_id", uuid.UUID{}),
	}
}

// Edges of the CustomerBrief.
func (CustomerBrief) Edges() []ent.Edge {
	return []ent.Edge{
		// CustomerBrief belongs to a Customer.
		edge.From("customer", Customer.Type).
			Ref("customer_briefs").
			Field("customer_id").
			Unique().
			Required(),

		// CustomerBrief belongs to a Brief.
		edge.From("brief", Brief.Type).
			Ref("customer_briefs").
			Field("brief_id").
			Unique().
			Required(),
	}
}
