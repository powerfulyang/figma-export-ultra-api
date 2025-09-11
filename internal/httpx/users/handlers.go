package users

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/httpx/kit"
)

// GetUsersHandler returns a paginated list of users.
//
//	@Summary      List users
//	@Description  Supports paging, sorting, and display_name filter
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        name        query   string  false  "display_name filter"
//	@Param        limit       query   int     false  "page size"      default(20)
//	@Param        offset      query   int     false  "offset"         default(0)
//	@Param        sort        query   string  false  "sort field"
//	@Param        mode        query   string  false  "paging mode: offset|cursor|snapshot" default(offset)
//	@Param        cursor      query   string  false  "cursor value (cursor mode)"
//	@Param        snapshot    query   string  false  "snapshot time (snapshot mode)"
//	@Param        with_total  query   bool    false  "return total in offset mode" default(false)
//	@Success      200  {object}  map[string]interface{}  "user list"
//	@Failure      400  {object}  map[string]interface{}  "bad request"
//	@Failure      500  {object}  map[string]interface{}  "internal error"
//	@Router       /api/v1/users [get]
//
// GetUsersHandler returns a paginated list of users
func GetUsersHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		nameFilter := c.Query("name")
		q := client.User.Query()
		if nameFilter != "" {
			q = q.Where(user.DisplayNameContains(nameFilter))
		}

		pg, err := kit.ParsePaging(c)
		if err != nil {
			return err
		}

		keyset := false
		switch pg.Mode {
		case "cursor":
			keyset = true
			s := lo.Ternary(pg.Sort != "", pg.Sort, "id:asc")
			if s != "id:asc" {
				return kit.BadRequest("cursor requires sort=id:asc", s)
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
		default:
			if s := pg.Sort; s != "" {
				var err error
				q, err = kit.ApplyUserSort(q, s)
				if err != nil {
					return err
				}
			}
			q = q.Limit(pg.Limit).Offset(pg.Offset)
		}

		users, err := q.All(ctx)
		if err != nil {
			return kit.InternalError("query users failed", err.Error())
		}
		if keyset {
			return buildKeysetResponse(c, users, &pg)
		}
		return buildOffsetResponse(c, users, &pg, client, nameFilter, "user")
	}
}

// CreateUserHandler creates a new user (display_name only)
//
//	@Summary      Create user
//	@Description  Create a user with display_name
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        user  body  users.UserCreateRequest  true  "{display_name}"
//	@Success      201   {object}  map[string]interface{}  "created"
//	@Failure      400   {object}  map[string]interface{}  "bad request"
//	@Failure      500   {object}  map[string]interface{}  "internal error"
//	@Router       /api/v1/users [post]
func CreateUserHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body UserCreateRequest
		if err := c.BodyParser(&body); err != nil || body.DisplayName == "" {
			return kit.BadRequest("display_name required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		u, err := client.User.Create().SetDisplayName(body.DisplayName).Save(ctx)
		if err != nil {
			return kit.InternalError("create user failed", err.Error())
		}
		return kit.Created(c, u)
	}
}

func buildKeysetResponse(c *fiber.Ctx, users []*ent.User, pg *kit.PagingParams) error {
	var nextCursor *string
	var nextCursorTS string
	hasMore := len(users) == pg.Limit
	if len(users) > 0 {
		last := users[len(users)-1]
		nextCursor = lo.ToPtr(last.ID.String())
		nextCursorTS = last.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	meta := kit.PageMeta{
		Limit:      pg.Limit,
		Count:      len(users),
		Cursor:     lo.TernaryF(pg.CursorID != nil, func() *string { s := pg.CursorID.String(); return &s }, func() *string { return nil }),
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Mode:       "cursor",
	}
	if pg.CursorID != nil && pg.CursorTS != nil {
		meta.CursorEnc = kit.EncodeCursor(pg.CursorID.String(), *pg.CursorTS)
	}
	if nextCursor != nil && len(users) > 0 {
		meta.NextCursorEnc = kit.EncodeCursor(*nextCursor, users[len(users)-1].CreatedAt)
	}
	if pg.Snapshot != nil {
		meta.Snapshot = pg.Snapshot.UTC().Format(time.RFC3339Nano)
		if pg.CursorTS != nil {
			meta.CursorTS = pg.CursorTS.UTC().Format(time.RFC3339Nano)
		}
		meta.NextCursorTS = nextCursorTS
	}
	return kit.List(c, users, meta)
}

func buildOffsetResponse(c *fiber.Ctx, users []*ent.User, pg *kit.PagingParams, client *ent.Client, nameFilter, _ string) error {
	nextOff := pg.Offset + len(users)
	meta := kit.PageMeta{
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
			tq = tq.Where(user.DisplayNameContains(nameFilter))
		}
		if total, err := tq.Count(ctx); err == nil {
			meta.Total = lo.ToPtr(total)
		}
	}
	return kit.List(c, users, meta)
}
