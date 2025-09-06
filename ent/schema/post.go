// Package schema contains ent entity schema definitions
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Post holds the schema definition for the Post entity.
type Post struct{ ent.Schema }

// Fields of the Post.
func (Post) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").NotEmpty(),
		field.String("content").Default(""),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Post.
func (Post) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("author", User.Type).Ref("posts").Unique().Required(),
	}
}
