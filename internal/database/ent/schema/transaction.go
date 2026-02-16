package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

const transactionsTableName = "transactions"

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

func (Transaction) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: transactionsTableName},
	}
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("sender").Comment("Sender address"),
		field.String("wallet_address").Comment("Recipient address (from customers.wallet_address)"),
		field.Float("amount").Comment("Amount in TON"),
		field.String("tx_hash").Unique().Comment("Transaction hash"),
		field.Uint64("seqno").Comment("Block sequence number"),
		field.Uint64("lt").Comment("Logical time"),
		field.Int("status").Default(0),
	}
}

// Indexes of the Transaction.
func (Transaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("wallet_address"),
		index.Fields("status"),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return nil
}
