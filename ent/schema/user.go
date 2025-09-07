package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct{ ent.Schema }

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("用户唯一标识"),
		field.String("username").NotEmpty().Unique().Comment("用户名，全局唯一"),
		field.String("display_name").Optional().Comment("显示名称"),
		field.String("email").Optional().Comment("主邮箱"),
		field.String("avatar_url").Optional().Comment("头像URL"),
		field.String("bio").Optional().Comment("个人简介"),
		field.String("timezone").Optional().Comment("时区设置"),
		field.String("language").Optional().Comment("语言偏好"),
		field.Bool("is_active").Comment("账号是否激活"),
		field.Time("last_login_at").Optional().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp",
			dialect.Postgres: "timestamptz",
		}).Comment("最后登录时间"),
		field.Time("created_at").Immutable().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp DEFAULT CURRENT_TIMESTAMP",
			dialect.Postgres: "timestamptz DEFAULT CURRENT_TIMESTAMP",
		}),
		field.Time("updated_at").UpdateDefault(time.Now).SchemaType(map[string]string{
			dialect.MySQL:    "timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
			dialect.Postgres: "timestamptz DEFAULT CURRENT_TIMESTAMP",
		}),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("auth_methods", UserAuth.Type).Comment("用户的认证方式"),
		edge.To("configs", UserConfig.Type),
		edge.To("export_records", ExportRecord.Type),
	}
}
