package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Device represents a client device instance (browser/app).
type Device struct{ ent.Schema }

// Fields of the Device.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("device_id").NotEmpty().MaxLen(128).Unique(),
		field.JSON("meta", map[string]any{}).Optional(),
		field.Time("first_seen_at").Default(time.Now),
		field.Time("last_seen_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Device.
func (Device) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique(),
		edge.To("visitor", Visitor.Type).Unique(),
	}
}

// Indexes defines indexes for the Device entity.
func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("user"),
		index.Edges("visitor"),
		index.Fields("last_seen_at"),
	}
}
