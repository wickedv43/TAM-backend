package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// IndexerState stores the blockchain indexing progress.
type IndexerState struct {
	ent.Schema
}

func (IndexerState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "indexer_state"},
	}
}

func (IndexerState) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),

		field.String("service_name").Unique(),
		field.Uint64("last_seqno").Default(0),
	}
}
