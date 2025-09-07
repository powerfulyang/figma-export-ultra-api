// Package schema provides database schema definitions for the figma export ultra API.
// This package contains entity schemas including AnonymousUser, User, UserAuth,
// UserConfig, ConfigHistory, and ExportRecord for managing user data and export operations.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UserConfig holds the schema definition for the UserConfig entity.
type UserConfig struct{ ent.Schema }

// Fields of the UserConfig.
func (UserConfig) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("配置唯一标识"),
		field.String("name").NotEmpty().Comment("配置名称"),
		field.JSON("config_data", map[string]interface{}{}).Comment("配置数据JSON"),
		field.String("version").Comment("配置版本"),
		field.String("description").Optional().Comment("配置描述"),
		field.Bool("is_default").Comment("是否为默认配置"),
		field.Bool("is_active").Comment("是否启用"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").UpdateDefault(time.Now),
	}
}

// Edges of the UserConfig.
func (UserConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("configs").Unique(),
		edge.From("anonymous_user", AnonymousUser.Type).Ref("configs").Unique(),
		edge.To("history", ConfigHistory.Type),
	}
}
