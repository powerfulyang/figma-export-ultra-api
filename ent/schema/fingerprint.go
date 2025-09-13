package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Fingerprint represents a browser/device fingerprint snapshot (hash only).
type Fingerprint struct{ ent.Schema }

// Fields of the Fingerprint.
func (Fingerprint) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("fp_hash").NotEmpty().Unique().MaxLen(128),
		field.String("ua_hash").Optional().Nillable().MaxLen(128),
		field.String("ip_hash").Optional().Nillable().MaxLen(128),
		field.Time("last_seen_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Fingerprint.
func (Fingerprint) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("visitor", Visitor.Type).Unique(),
	}
}

// Indexes defines indexes for the Fingerprint entity.
func (Fingerprint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("last_seen_at"),
		index.Edges("visitor"),
	}
}
