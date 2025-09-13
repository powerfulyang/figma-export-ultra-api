// Package schema defines Ent ORM schema types for the application.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ConfigItem represents a user-saved configuration that can be shared.
type ConfigItem struct{ ent.Schema }

// Fields defines the fields for the ConfigItem entity.
func (ConfigItem) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").NotEmpty().MaxLen(255),
		field.JSON("data", map[string]any{}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges defines the relationships for the ConfigItem entity.
func (ConfigItem) Edges() []ent.Edge {
	return []ent.Edge{
		// owner user (required)
		edge.To("owner", User.Type).Unique().Required(),
		// shared to groups (many-to-many)
		edge.To("shared_groups", Group.Type),
	}
}

// Indexes defines indexes for the ConfigItem entity.
func (ConfigItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("owner"),
		index.Fields("updated_at"),
	}
}
