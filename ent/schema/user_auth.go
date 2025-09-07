// Package schema provides database schema definitions for the figma export ultra API.
// This package contains entity schemas including AnonymousUser, User, UserAuth,
// UserConfig, ConfigHistory, and ExportRecord for managing user data and export operations.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// UserAuth holds the schema definition for the UserAuth entity.
type UserAuth struct{ ent.Schema }

// Fields of the UserAuth.
func (UserAuth) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("认证记录唯一标识"),
		field.Enum("auth_type").
			Values(
				"email_password", // 邮箱密码登录
				"google",         // Google登录
				"github",         // GitHub登录
				"wechat",         // 微信登录
				"qq",             // QQ登录
				"apple",          // Apple登录
				"facebook",       // Facebook登录
				"twitter",        // Twitter登录
				"linkedin",       // LinkedIn登录
			).
			Comment("认证方式类型"),

		field.String("identifier").NotEmpty().Comment("认证标识符，如邮箱、第三方用户ID等"),
		field.String("credential").Optional().Comment("认证凭据，如密码哈希、token等"),
		field.String("provider_user_id").Optional().Comment("第三方平台的用户ID"),
		field.String("provider_username").Optional().Comment("第三方平台的用户名"),
		field.String("provider_email").Optional().Comment("第三方平台的邮箱"),
		field.String("provider_avatar").Optional().Comment("第三方平台的头像"),

		field.JSON("provider_data", map[string]interface{}{}).
			Optional().
			Comment("第三方平台的完整用户信息"),

		field.String("access_token").Optional().Comment("第三方平台的访问令牌"),
		field.String("refresh_token").Optional().Comment("第三方平台的刷新令牌"),
		field.Time("token_expires_at").Optional().Comment("令牌过期时间"),

		field.Bool("is_primary").Comment("是否为主要认证方式"),
		field.Bool("is_enabled").Comment("是否启用"),

		field.Time("last_used_at").Optional().Comment("最后使用时间"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").UpdateDefault(time.Now),
	}
}

// Edges of the UserAuth.
func (UserAuth) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("auth_methods").
			Unique().
			Required().
			Comment("所属用户"),
	}
}

// Indexes of the UserAuth.
func (UserAuth) Indexes() []ent.Index {
	return []ent.Index{
		// 确保同一认证类型+标识符全局唯一（防止同一邮箱注册多次）
		index.Fields("auth_type", "identifier").Unique(),
		// 第三方用户ID的唯一索引（防止同一第三方账号绑定多次）
		index.Fields("auth_type", "provider_user_id").Unique(),
		// 提高查询性能的索引
		index.Fields("identifier"),
		index.Fields("is_enabled"),
		index.Fields("last_used_at"),
	}
}
