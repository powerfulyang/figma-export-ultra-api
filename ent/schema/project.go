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

// Project represents a project that can have multiple config items.
type Project struct{ ent.Schema }

// Fields defines the fields for the Project entity.
func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").NotEmpty().MaxLen(255),
		field.String("url").NotEmpty().MaxLen(255),
		field.String("description").Optional().MaxLen(1000),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges defines the relationships for the Project entity.
func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		// owner user (required)
		edge.To("owner", User.Type).Unique().Required(),
		// project configs (one-to-many)
		edge.From("project_configs", ProjectConfig.Type).Ref("project"),
	}
}

// Indexes defines indexes for the Project entity.
func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("owner"),
		index.Fields("updated_at"),
		// url is unique per owner
		index.Edges("owner").Fields("url").Unique(),
	}
}
