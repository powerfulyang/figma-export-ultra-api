package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Group represents a user group for sharing configs.
type Group struct{ ent.Schema }

// Fields defines the fields for the Group entity.
func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").Optional().MaxLen(255),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges defines the relationships for the Group entity.
func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		// many-to-many members
		edge.From("members", User.Type).Ref("groups"),
		// many-to-many configs shared to this group
		edge.From("configs", ConfigItem.Type).Ref("shared_groups"),
	}
}
