package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Visitor represents an anonymous visitor identity bound by anon_id and fp hash.
type Visitor struct{ ent.Schema }

// Fields of the Visitor.
func (Visitor) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("anon_id").NotEmpty().Unique().MaxLen(64),
		field.String("primary_fp_hash").Optional().Nillable().MaxLen(128),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Visitor.
func (Visitor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("devices", Device.Type).Ref("visitor"),
		edge.From("fingerprints", Fingerprint.Type).Ref("visitor"),
	}
}

// Indexes defines indexes for the Visitor entity.
func (Visitor) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("primary_fp_hash"),
	}
}
