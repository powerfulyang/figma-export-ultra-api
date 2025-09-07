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

// AnonymousUser holds the schema definition for the AnonymousUser entity.
type AnonymousUser struct{ ent.Schema }

// Fields of the AnonymousUser.
func (AnonymousUser) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("匿名用户唯一标识"),
		field.String("browser_fingerprint").NotEmpty().Unique().Comment("浏览器指纹唯一标识"),
		field.String("user_agent").Optional().Comment("用户代理字符串"),
		field.String("ip_address").Optional().Comment("IP地址"),
		field.String("timezone").Optional().Comment("时区信息"),
		field.String("language").Optional().Comment("语言设置"),
		field.String("screen_resolution").Optional().Comment("屏幕分辨率"),
		field.Bool("is_active").Default(true).Comment("是否活跃"),
		field.Time("last_activity_at").Default(time.Now()).Comment("最后活动时间"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").UpdateDefault(time.Now),
	}
}

// Edges of the AnonymousUser.
func (AnonymousUser) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("configs", UserConfig.Type),
		edge.To("export_records", ExportRecord.Type),
	}
}
