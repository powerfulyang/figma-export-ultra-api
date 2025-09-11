# device 表设计与使用说明

## 定位
- 目标：标识“设备实例”（浏览器/APP 安装），贯穿匿名期与登录期，用于行为归因、风控与多设备管理。
- 位置：`ent/schema/device.go`（已升级为 UUID 主键）。

## 表结构（MVP）
- `id`：UUID 主键（服务端生成）。
- `device_id`：字符串，客户端生成的稳定设备实例 ID（唯一，建议 UUID/安装 ID，<=128）。
- `meta`：JSON 低敏元数据（可选），如：ua family、os、app 版本、语言、时区、屏幕分辨率摘要等。
- `first_seen_at` / `last_seen_at`：首次/最近活跃时间。
- 关系：
  - `user`（唯一反向）：登录期所有权归属到某个用户。
  - `visitor`（唯一反向）：匿名期挂在对应的访客上。
- 不变式：任一时刻仅建议关联 `user` 或 `visitor` 之一；迁移时执行“解绑 → 绑定”。

## 索引与约束建议
- 唯一键：`UNIQUE(device_id)`（已在 Schema 中约束）。
- 二级索引：`(user_id)`、`(visitor_id)`、`(last_seen_at)` 便于查询统计与清理。
- 清理策略：定期删除长期不活跃且无业务引用的设备记录（例如 180 天）。

## 典型流程
### 1) 匿名初始化（/api/v1/auth/anonymous/init）
- 入参：`device_id`（必填），`fp_hash`（可选），`meta`（可选）。
- 服务端：
  1. 命中/创建 `Visitor`（参见 anon_id 文档）。
  2. 对 `Device` 执行 Upsert：按 `device_id` 查找；
     - 若不存在：创建并 `SetVisitor(v)`，`SetMeta(meta)`，`SetFirstSeenAt/LastSeenAt(now)`；
     - 若存在：`UpdateOne`，`SetLastSeenAt(now)`，必要时更新 `meta` 与关联（若当前未绑定或绑定不同 Visitor，则按规则绑定当前 Visitor）。
  3. 返回匿名 Access JWT，并设置 Refresh Cookie。

### 2) 指纹/环境同步（/api/v1/auth/fp/sync）
- 目的：补充/更新 `Fingerprint` 与 `Device.meta`，刷新 `last_seen_at`。
- 幂等：按 `device_id`、`fp_hash` 做去重，避免多次写入同一指纹。

### 3) 登录合并（/api/v1/auth/login）
- 验证身份成功后，执行“软合并”：
  - 在事务中将当前匿名期资源补充 `user_id`；`Device` 从 `visitor -> user`：
    - 仅当设备属于当前匿名会话（或携带的 `anon_id` 指向的 Visitor）时迁移；
    - 保留历史 `visitor_id` 在其他表中以便可追溯（如 Event）。
  - 幂等：重复合并不应重复写入或报错。

### 4) 登出/吊销（可选增强）
- 纯 JWT：不必写库；可选将 `device_id` 放入 JWT Claims 用于审计。
- 若引入刷新令牌黑名单：按 `jti + device_id` 维度撤销，实现“按设备登出”。

## Mermaid 时序图
```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant API as API Server
  participant DB as Postgres (Ent)

  Note over C: 初次匿名
  C->>API: POST /auth/anonymous/init { device_id, fp_hash?, meta? }
  API->>DB: Upsert Device by device_id (bind Visitor)
  API-->>C: 200 OK { access_token, anon_id }

  Note over C: 登录
  C->>API: POST /auth/login { identifier, password }
  API->>DB: Tx: merge Visitor resources; Device visitor->user
  API-->>C: 200 OK { access_token(user) }
```

## Upsert 与迁移示例
### SQL 伪代码
```sql
-- Upsert Device（Postgres 可用 ON CONFLICT）
INSERT INTO devices (device_id, meta, visitor_id, first_seen_at, last_seen_at)
VALUES ($DEVICE_ID, $META, $VISITOR_ID, NOW(), NOW())
ON CONFLICT (device_id)
DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at,
              meta = COALESCE(EXCLUDED.meta, devices.meta),
              visitor_id = COALESCE(EXCLUDED.visitor_id, devices.visitor_id);
```

### Ent 代码片段（Go）
```go
// Upsert by device_id
ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
defer cancel()

d, err := client.Device.
    Query().
    Where(device.DeviceIDEQ(req.DeviceID)).
    First(ctx)
if ent.IsNotFound(err) {
    _, err = client.Device.Create().
        SetDeviceID(req.DeviceID).
        SetVisitor(v).
        SetMeta(req.Meta). // map[string]any
        Save(ctx)
} else if err == nil {
    err = client.Device.UpdateOne(d).
        SetLastSeenAt(time.Now()).
        SetVisitor(v). // 若当前匿名态
        SetMeta(req.Meta). // 可按字段级合并
        Exec(ctx)
}
```

### 登录迁移（Visitor -> User）
```go
err := client.Tx(ctx, func(tx *ent.Tx) error {
    // 将设备归属到用户
    if err := tx.Device.
        Update().
        Where(device.HasVisitorWith(visitor.IDEQ(curVisitorID))).
        SetVisitor(nil).
        SetUser(u). // 登录用户
        Exec(ctx); err != nil {
        return err
    }
    // 可继续迁移 Fingerprint/Event 等补充 user_id
    return nil
})
```

## 隐私与安全
- 最小化：`meta` 只保存低敏、非可逆信息；不要存精准位置信息、硬件序列号。
- 合规：设备标识可能被视为个人数据，需在隐私政策/同意管理中披露，提供退出/删除能力。
- DNT：尊重 Do Not Track；可仅保留 `device_id` 与必要时间戳。

## 风控与限流
- 建议限流键：`ip + device_id + (anon_id|fp_hash)`，分别用于匿名初始化、刷新、登录等接口。
- 异常监控：单用户/单 IP 设备数量阈值、短时间设备切换频率等。

## 测试清单
- Upsert 幂等：相同 `device_id` 多次调用不重复创建。
- 迁移幂等：同一 Visitor 合并到同一 User 多次不报错。
- 关联正确：匿名期挂 Visitor，登录后切换到 User。
- 清理安全：长期不活跃设备清理不影响活跃用户。

## 迁移注意
- 本仓库已将 `device.id` 升级为 UUID；若数据库中已有 int 主键，需评估迁移方案（新表重建/中间表映射/导出导入）。

## 与 JWT 的关系
- 设备不是授权主体，但可放入 JWT claims（如 `device_id`）用于审计与风控；不要依赖其作为授权判据。

**参考**：匿名访客与 `anon_id` 说明见 `prd/anon_id.md`。

