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

// ConfigHistory holds the schema definition for the ConfigHistory entity.
type ConfigHistory struct{ ent.Schema }

// Fields of the ConfigHistory.
func (ConfigHistory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("历史记录唯一标识"),
		field.JSON("old_config_data", map[string]interface{}{}).
			Optional().
			Comment("变更前的配置数据"),
		field.JSON("new_config_data", map[string]interface{}{}).
			Comment("变更后的配置数据"),
		field.String("change_type").NotEmpty().Comment("变更类型：create, update, delete"),
		field.String("change_description").Optional().Comment("变更描述"),
		field.String("old_version").Optional().Comment("旧版本号"),
		field.String("new_version").NotEmpty().Comment("新版本号"),
		field.String("ip_address").Optional().Comment("操作IP地址"),
		field.String("user_agent").Optional().Comment("用户代理"),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Edges of the ConfigHistory.
func (ConfigHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("config", UserConfig.Type).Ref("history").Unique().Required(),
	}
}
