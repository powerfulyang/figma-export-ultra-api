package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ExportRecord holds the schema definition for the ExportRecord entity.
type ExportRecord struct{ ent.Schema }

// Fields of the ExportRecord.
func (ExportRecord) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Comment("导出记录唯一标识"),
		field.String("figma_file_id").NotEmpty().Comment("Figma文件ID"),
		field.String("figma_file_name").Optional().Comment("Figma文件名称"),
		field.String("figma_file_url").Optional().Comment("Figma文件URL"),
		field.Enum("export_format").Values("png", "jpg", "svg", "pdf").Comment("导出格式"),
		field.String("export_scale").Comment("导出缩放比例"),
		field.JSON("export_settings", map[string]interface{}{}).
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).Comment("导出设置JSON"),
		field.JSON("selected_nodes", []string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).Comment("选中的节点ID列表"),
		field.Int("total_assets").Comment("总资源数量"),
		field.Int("exported_assets").Comment("已导出资源数量"),
		field.Enum("status").Values("pending", "processing", "completed", "failed", "cancelled").
			Comment("导出状态"),
		field.String("download_url").Optional().Comment("下载链接"),
		field.String("error_message").Optional().Comment("错误信息"),
		field.String("ip_address").Optional().Comment("操作IP地址"),
		field.String("user_agent").Optional().Comment("用户代理"),
		field.Time("started_at").Optional().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp",
			dialect.Postgres: "timestamptz",
		}).Comment("开始导出时间"),
		field.Time("completed_at").Optional().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp",
			dialect.Postgres: "timestamptz",
		}).Comment("完成导出时间"),
		field.Time("expires_at").Optional().SchemaType(map[string]string{
			dialect.MySQL:    "timestamp",
			dialect.Postgres: "timestamptz",
		}).Comment("下载链接过期时间"),
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

// Edges of the ExportRecord.
func (ExportRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("export_records").Unique(),
		edge.From("anonymous_user", AnonymousUser.Type).Ref("export_records").Unique(),
	}
}
