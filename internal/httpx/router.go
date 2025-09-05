package httpx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	entlib "entgo.io/ent"
	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/post"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/esx"
	"fiber-ent-apollo-pg/internal/mqx"
)

type Providers struct {
	MQ mqx.Publisher
	ES *esx.Client
}

func Register(app *fiber.App, client *ent.Client, providers ...*Providers) {
	var p *Providers
	if len(providers) > 0 {
		p = providers[0]
	}
	app.Get("/health", func(c *fiber.Ctx) error { return OK(c, fiber.Map{"status": "ok"}) })

	app.Get("/users", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		nameFilter := c.Query("name")
		q := client.User.Query()
		if nameFilter != "" {
			q = q.Where(user.NameContains(nameFilter))
		}

		pg, err := parsePaging(c)
		if err != nil {
			return err
		}

		keyset := false
		switch pg.Mode {
		case "cursor":
			keyset = true
			s := pg.Sort
			if s == "" {
				s = "id:asc"
			}
			if s != "id:asc" {
				return BadRequest("cursor requires sort=id:asc", s)
			}
			if pg.CursorID != nil {
				q = q.Where(user.IDGT(*pg.CursorID))
			}
			q = q.Order(entlib.Asc(user.FieldID)).Limit(pg.Limit)
		case "snapshot":
			keyset = true
			q = q.Where(user.CreatedAtLTE(*pg.Snapshot)).Order(entlib.Desc(user.FieldCreatedAt), entlib.Desc(user.FieldID)).Limit(pg.Limit)
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
			var nextCursor *int
			var nextCursorTS string
			hasMore := len(users) == pg.Limit
			if len(users) > 0 {
				last := users[len(users)-1]
				id := last.ID
				nextCursor = &id
				nextCursorTS = last.CreatedAt.UTC().Format(time.RFC3339Nano)
			}
			meta := PageMeta{Limit: pg.Limit, Count: len(users), Cursor: pg.CursorID, NextCursor: nextCursor, HasMore: hasMore, Mode: "cursor"}
			if pg.CursorID != nil && pg.CursorTS != nil {
				meta.CursorEnc = encodeCursor(*pg.CursorID, *pg.CursorTS)
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

		nextOff := pg.Offset + len(users)
		meta := PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(users), NextOffset: &nextOff, HasMore: len(users) == pg.Limit, Mode: "offset"}
		if pg.WithTotal {
			tq := client.User.Query()
			if nameFilter != "" {
				tq = tq.Where(user.NameContains(nameFilter))
			}
			if total, err := tq.Count(ctx); err == nil {
				meta.Total = &total
			}
		}
		return List(c, users, meta)
	})

	app.Post("/users", func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&body); err != nil || body.Name == "" {
			return BadRequest("name required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		u, err := client.User.Create().SetName(body.Name).Save(ctx)
		if err != nil {
			return InternalError("create user failed", err.Error())
		}
		return Created(c, u)
	})

	// Posts APIs
	app.Get("/posts", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		uid := c.QueryInt("user_id", 0)
		search := c.Query("q")
		q := client.Post.Query().WithAuthor()
		if uid > 0 {
			q = q.Where(post.HasAuthorWith(user.IDEQ(uid)))
		}
		if search != "" {
			q = q.Where(post.TitleContains(search))
		}

		pg, err := parsePaging(c)
		if err != nil {
			return err
		}

		keyset := false
		switch pg.Mode {
		case "cursor":
			keyset = true
			s := pg.Sort
			if s == "created_at:desc" || s == "" {
				s = "id:asc"
			}
			if s != "id:asc" {
				return BadRequest("cursor requires sort=id:asc", s)
			}
			if pg.CursorID != nil {
				q = q.Where(post.IDGT(*pg.CursorID))
			}
			q = q.Order(entlib.Asc(post.FieldID)).Limit(pg.Limit)
		case "snapshot":
			keyset = true
			q = q.Where(post.CreatedAtLTE(*pg.Snapshot)).Order(entlib.Desc(post.FieldCreatedAt), entlib.Desc(post.FieldID)).Limit(pg.Limit)
			if pg.CursorID != nil {
				if pg.CursorTS != nil {
					curTS := pg.CursorTS.UTC()
					q = q.Where(post.Or(post.CreatedAtLT(curTS), post.And(post.CreatedAtEQ(curTS), post.IDLT(*pg.CursorID))))
				} else {
					q = q.Where(post.IDLT(*pg.CursorID))
				}
			}
		default:
			s := pg.Sort
			if s == "" {
				s = "created_at:desc"
			}
			var err error
			q, err = applyPostSort(q, s)
			if err != nil {
				return err
			}
			q = q.Limit(pg.Limit).Offset(pg.Offset)
		}

		posts, err := q.All(ctx)
		if err != nil {
			return InternalError("query posts failed", err.Error())
		}

		if keyset {
			var nextCursor *int
			var nextCursorTS string
			hasMore := len(posts) == pg.Limit
			if len(posts) > 0 {
				last := posts[len(posts)-1]
				id := last.ID
				nextCursor = &id
				nextCursorTS = last.CreatedAt.UTC().Format(time.RFC3339Nano)
			}
			meta := PageMeta{Limit: pg.Limit, Count: len(posts), Cursor: pg.CursorID, NextCursor: nextCursor, HasMore: hasMore, Mode: "cursor"}
			if pg.CursorID != nil && pg.CursorTS != nil {
				meta.CursorEnc = encodeCursor(*pg.CursorID, *pg.CursorTS)
			}
			if nextCursor != nil && len(posts) > 0 {
				meta.NextCursorEnc = encodeCursor(*nextCursor, posts[len(posts)-1].CreatedAt)
			}
			if pg.Snapshot != nil {
				meta.Snapshot = pg.Snapshot.UTC().Format(time.RFC3339Nano)
				if pg.CursorTS != nil {
					meta.CursorTS = pg.CursorTS.UTC().Format(time.RFC3339Nano)
				}
				meta.NextCursorTS = nextCursorTS
			}
			return List(c, posts, meta)
		}

		nextOff := pg.Offset + len(posts)
		meta := PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(posts), NextOffset: &nextOff, HasMore: len(posts) == pg.Limit, Mode: "offset"}
		if pg.WithTotal {
			tq := client.Post.Query()
			if uid > 0 {
				tq = tq.Where(post.HasAuthorWith(user.IDEQ(uid)))
			}
			if search != "" {
				tq = tq.Where(post.TitleContains(search))
			}
			if total, err := tq.Count(ctx); err == nil {
				meta.Total = &total
			}
		}
		return List(c, posts, meta)
	})

	app.Post("/posts", func(c *fiber.Ctx) error {
		var body struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			UserID  int    `json:"user_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.Title == "" || body.UserID == 0 {
			return BadRequest("title and user_id required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		p, err := client.Post.Create().SetTitle(body.Title).SetContent(body.Content).SetAuthorID(body.UserID).Save(ctx)
		if err != nil {
			return InternalError("create post failed", err.Error())
		}
		// Publish MQ event
		if pvd := p; pvd != nil { /* shadow fix */
		}
		if p != nil && pvd := providers; pvd == nil { /* noop */
		}
		if p != nil { /* nop */
		}
		if p := p; p != nil { /* nop */
		}
		if p != nil { /* shadow */
		}
		if pvd := providers; pvd != nil && len(pvd) > 0 && pvd[0] != nil && pvd[0].MQ != nil {
			evt := map[string]any{"type": "post.created", "id": p.ID, "user_id": p.UserID, "title": p.Title}
			b, _ := json.Marshal(evt)
			_ = pvd[0].MQ.Publish(ctx, "post.created", b)
		}
		// Index into ES
		if len(providers) > 0 && providers[0] != nil && providers[0].ES != nil {
			_ = esx.IndexPost(ctx, providers[0].ES, "posts", esx.PostDoc{ID: p.ID, Title: p.Title, Content: p.Content, UserID: p.UserID, CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano)})
		}
		return Created(c, p)
	})

	// ES search
	app.Get("/search/posts", func(c *fiber.Ctx) error {
		if len(providers) == 0 || providers[0] == nil || providers[0].ES == nil {
			return OK(c, fiber.Map{"hits": []any{}})
		}
		q := c.Query("q")
		if q == "" {
			return BadRequest("q required", nil)
		}
		from := c.QueryInt("offset", 0)
		size := clamp(c.QueryInt("limit", 20), 1, 100)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		res, err := esx.SearchPosts(ctx, providers[0].ES, "posts", q, from, size)
		if err != nil {
			return InternalError("es search failed", err.Error())
		}
		return OK(c, res)
	})
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
