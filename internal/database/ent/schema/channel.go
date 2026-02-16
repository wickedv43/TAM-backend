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

const channelsTableName = "channels"

// Channel holds the schema definition for the Channel entity.
type Channel struct {
	ent.Schema
}

func (Channel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: channelsTableName},
	}
}

// Fields of the Channel.
func (Channel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUID).Immutable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Int64("tg_id").Unique(),
		field.String("tg_username").Optional(),
		field.String("tg_name").Optional(),
		field.String("tg_description").Optional(),
		field.String("tg_picture").Optional(),
		field.Text("commentary").Optional(),
		field.Bool("is_listed").Default(false),
		field.Strings("tags").Optional(),
		field.Time("stats_updated_at").Default(time.Now),
		field.JSON("stats", map[string]interface{}{}).Optional(),
		field.JSON("prices", map[string]float64{}).Optional(),
		field.Int("subscribers").Optional(),
		field.Int("premium_subscribers").Optional(),
		field.Int("median_post_views").Optional(),
		field.Int("avg_post_views").Optional(),
		field.Int("status").Default(0),
		field.String("main_language").Optional(),
		field.Time("restricted_from").Default(func() time.Time { return time.Unix(0, 0) }),
		field.Time("restricted_till").Default(func() time.Time { return time.Unix(0, 0) }),
		field.Enum("restriction_reason").Values("unspecified", "cant_verify").Default("unspecified"),
		field.Float("notifications_on").Default(0),
		field.Time("first_post_date").Optional(),
		field.Int("total_posts").Optional(),
	}
}

// Edges of the Channel.
func (Channel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("customer_channels", CustomerChannel.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("deals", Deal.Type),
	}
}
