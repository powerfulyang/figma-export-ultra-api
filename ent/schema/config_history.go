package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
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
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).Comment("变更前的配置数据"),
		field.JSON("new_config_data", map[string]interface{}{}).
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).Comment("变更后的配置数据"),
		field.String("change_type").NotEmpty().Comment("变更类型：create, update, delete"),
		field.String("change_description").Optional().Comment("变更描述"),
		field.String("old_version").Optional().Comment("旧版本号"),
		field.String("new_version").NotEmpty().Comment("新版本号"),
		field.String("ip_address").Optional().Comment("操作IP地址"),
		field.String("user_agent").Optional().Comment("用户代理"),
		field.Time("created_at").Immutable().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp DEFAULT CURRENT_TIMESTAMP",
			dialect.Postgres: "timestamptz DEFAULT CURRENT_TIMESTAMP",
		}),
	}
}

// Edges of the ConfigHistory.
func (ConfigHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("config", UserConfig.Type).Ref("history").Unique().Required(),
	}
}
