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

const briefsTableName = "briefs"

// Brief holds the schema definition for the Brief entity.
type Brief struct {
	ent.Schema
}

func (Brief) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: briefsTableName},
	}
}

// Fields of the Brief.
func (Brief) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Brief.
func (Brief) Edges() []ent.Edge {
	return []ent.Edge{
		// Brief has many CustomerBrief rows.
		edge.To("customer_briefs", CustomerBrief.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
