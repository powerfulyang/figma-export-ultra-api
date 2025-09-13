package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Identity holds authentication identifier for a user (e.g., password login).
type Identity struct{ ent.Schema }

// Fields of the Identity.
func (Identity) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Enum("provider").Values("password").Default("password"),
		field.String("identifier").NotEmpty().MaxLen(320),
		field.String("secret_hash").Optional().Nillable().MaxLen(200),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Identity.
func (Identity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Required(),
	}
}

// Indexes defines indexes for the Identity entity.
func (Identity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "identifier").Unique(),
	}
}
