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

// ProjectConfig represents the relationship between project and config item with active status.
type ProjectConfig struct{ ent.Schema }

// Fields defines the fields for the ProjectConfig entity.
func (ProjectConfig) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Bool("active").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges defines the relationships for the ProjectConfig entity.
func (ProjectConfig) Edges() []ent.Edge {
	return []ent.Edge{
		// belongs to project (required)
		edge.To("project", Project.Type).Unique().Required(),
		// belongs to config item (required)
		edge.To("config_item", ConfigItem.Type).Unique().Required(),
	}
}

// Indexes defines indexes for the ProjectConfig entity.
func (ProjectConfig) Indexes() []ent.Index {
	return []ent.Index{
		// composite unique index for project and config_item
		index.Edges("project", "config_item").Unique(),
		// index for active configs
		index.Fields("active"),
		// index for project to find active config quickly
		index.Edges("project").Fields("active"),
	}
}
