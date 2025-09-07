package httpx

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/user"
)

// GetUsersHandler 获取用户列表处理器
//
//	@Summary		获取用户列表
//	@Description	支持分页、排序和筛选的用户列表查询
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			name		query		string	false	"用户名筛选"
//	@Param			limit		query		int		false	"每页数量"	default(20)
//	@Param			offset		query		int		false	"偏移量"	default(0)
//	@Param			sort		query		string	false	"排序字段"
//	@Param			mode		query		string	false	"分页模式: offset, cursor, snapshot"	default(offset)
//	@Param			cursor		query		string	false	"游标位置（cursor模式）"
//	@Param			snapshot	query		string	false	"快照时间（snapshot模式）"
//	@Param			with_total	query		bool	false	"是否返回总数"	default(false)
//	@Success		200			{object}	map[string]interface{}	"用户列表"
//	@Failure		400			{object}	map[string]interface{}	"请求参数错误"
//	@Failure		500			{object}	map[string]interface{}	"内部服务器错误"
//	@Router			/api/v1/users [get]
func GetUsersHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		nameFilter := c.Query("name")
		q := client.User.Query()
		if nameFilter != "" {
			q = q.Where(user.UsernameContains(nameFilter))
		}

		pg, err := parsePaging(c)
		if err != nil {
			return err
		}

		keyset := false
		switch pg.Mode {
		case "cursor":
			keyset = true
			s := lo.Ternary(pg.Sort != "", pg.Sort, "id:asc")
			if s != "id:asc" {
				return BadRequest("cursor requires sort=id:asc", s)
			}
			if pg.CursorID != nil {
				q = q.Where(user.IDGT(*pg.CursorID))
			}
			q = q.Order(ent.Asc(user.FieldID)).Limit(pg.Limit)
		case "snapshot":
			keyset = true
			q = q.Where(user.CreatedAtLTE(*pg.Snapshot)).Order(ent.Desc(user.FieldCreatedAt), ent.Desc(user.FieldID)).Limit(pg.Limit)
			if pg.CursorID != nil {
				if pg.CursorTS != nil {
					curTS := pg.CursorTS.UTC()
					q = q.Where(user.Or(user.CreatedAtLT(curTS), user.And(user.CreatedAtEQ(curTS), user.IDLT(*pg.CursorID))))
				} else {
					q = q.Where(user.IDLT(*pg.CursorID))
				}
			}
		default: // offset
			if s := pg.Sort; s != "" {
				var err error
				q, err = applyUserSort(q, s)
				if err != nil {
					return err
				}
			}
			q = q.Limit(pg.Limit).Offset(pg.Offset)
		}

		users, err := q.All(ctx)
		if err != nil {
			return InternalError("query users failed", err.Error())
		}

		if keyset {
			return buildKeysetResponse(c, users, &pg)
		}

		return buildOffsetResponse(c, users, &pg, client, nameFilter, "user")
	}
}

// CreateUserHandler 创建用户处理器
//
//	@Summary		创建新用户
//	@Description	创建一个新的用户
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		map[string]string	true	"用户信息"	example({"username": "john_doe"})
//	@Success		201		{object}	map[string]interface{}	"创建成功"
//	@Failure		400		{object}	map[string]interface{}	"请求参数错误"
//	@Failure		500		{object}	map[string]interface{}	"内部服务器错误"
//	@Router			/api/v1/users [post]
func CreateUserHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Username string `json:"username"`
		}
		if err := c.BodyParser(&body); err != nil || body.Username == "" {
			return BadRequest("username required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		u, err := client.User.Create().SetUsername(body.Username).Save(ctx)
		if err != nil {
			return InternalError("create user failed", err.Error())
		}
		return Created(c, u)
	}
}

// buildKeysetResponse 构建基于游标的响应
func buildKeysetResponse(c *fiber.Ctx, users []*ent.User, pg *PagingParams) error {
	var nextCursor *string
	var nextCursorTS string
	hasMore := len(users) == pg.Limit
	if len(users) > 0 {
		last := users[len(users)-1]
		nextCursor = lo.ToPtr(last.ID.String())
		nextCursorTS = last.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	meta := PageMeta{
		Limit:      pg.Limit,
		Count:      len(users),
		Cursor:     lo.TernaryF(pg.CursorID != nil, func() *string { s := pg.CursorID.String(); return &s }, func() *string { return nil }),
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Mode:       "cursor",
	}
	if pg.CursorID != nil && pg.CursorTS != nil {
		meta.CursorEnc = encodeCursor(pg.CursorID.String(), *pg.CursorTS)
	}
	if nextCursor != nil && len(users) > 0 {
		meta.NextCursorEnc = encodeCursor(*nextCursor, users[len(users)-1].CreatedAt)
	}
	if pg.Snapshot != nil {
		meta.Snapshot = pg.Snapshot.UTC().Format(time.RFC3339Nano)
		if pg.CursorTS != nil {
			meta.CursorTS = pg.CursorTS.UTC().Format(time.RFC3339Nano)
		}
		meta.NextCursorTS = nextCursorTS
	}
	return List(c, users, meta)
}

// buildOffsetResponse 构建基于偏移的响应
func buildOffsetResponse(c *fiber.Ctx, users []*ent.User, pg *PagingParams, client *ent.Client, nameFilter, _ string) error {
	nextOff := pg.Offset + len(users)
	meta := PageMeta{
		Limit:      pg.Limit,
		Offset:     pg.Offset,
		Count:      len(users),
		NextOffset: lo.ToPtr(nextOff),
		HasMore:    len(users) == pg.Limit,
		Mode:       "offset",
	}
	if pg.WithTotal {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		tq := client.User.Query()
		if nameFilter != "" {
			tq = tq.Where(user.UsernameContains(nameFilter))
		}
		if total, err := tq.Count(ctx); err == nil {
			meta.Total = lo.ToPtr(total)
		}
	}
	return List(c, users, meta)
}
