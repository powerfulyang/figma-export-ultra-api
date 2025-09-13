// Package groups provides HTTP handlers for managing user groups.
package groups

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/group"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/httpx/kit"
	"fiber-ent-apollo-pg/internal/httpx/mw"
)

// CreateGroupRequest is the request payload to create a group.
// swagger:model CreateGroupRequest
type CreateGroupRequest struct {
	Name      string      `json:"name"`
	MemberIDs []uuid.UUID `json:"member_ids"`
}

// CreateGroupHandler creates a group and adds members (including caller if not present).
//
//	@Summary      Create group
//	@Description  Create a group and add members (caller auto-included)
//	@Tags         groups
//	@Accept       json
//	@Produce      json
//	@Param        body  body  groups.CreateGroupRequest  true  "group payload"
//	@Success      201   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Router       /api/v1/groups [post]
func CreateGroupHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		var req CreateGroupRequest
		if err := c.BodyParser(&req); err != nil {
			return kit.BadRequest("invalid body", nil)
		}
		hasCaller := false
		for _, id := range req.MemberIDs {
			if id == uid {
				hasCaller = true
				break
			}
		}
		if !hasCaller {
			req.MemberIDs = append(req.MemberIDs, uid)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		g, err := client.Group.Create().SetName(req.Name).Save(ctx)
		if err != nil {
			return kit.InternalError("create group failed", err.Error())
		}
		if len(req.MemberIDs) > 0 {
			if err := client.Group.UpdateOne(g).AddMemberIDs(req.MemberIDs...).Exec(ctx); err != nil {
				return kit.InternalError("add members failed", err.Error())
			}
		}
		return kit.Created(c, g)
	}
}

// ListMyGroupsHandler lists groups the current user is a member of.
//
//	@Summary      List my groups
//	@Description  Groups that include the current user as member
//	@Tags         groups
//	@Accept       json
//	@Produce      json
//	@Param        limit       query   int     false  "page size"      default(20)
//	@Param        offset      query   int     false  "offset"         default(0)
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Router       /api/v1/groups [get]
func ListMyGroupsHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		pg, err := kit.ParsePaging(c)
		if err != nil {
			return err
		}
		q := client.Group.Query().Where(group.HasMembersWith(user.IDEQ(uid))).Order(ent.Desc(group.FieldCreatedAt))
		items, err := q.Limit(pg.Limit).Offset(pg.Offset).All(ctx)
		if err != nil {
			return kit.InternalError("query groups failed", err.Error())
		}
		nextOff := pg.Offset + len(items)
		meta := kit.PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(items), NextOffset: &nextOff, HasMore: len(items) == pg.Limit, Mode: "offset"}
		return kit.List(c, items, meta)
	}
}

// DeleteGroupHandler deletes a group if the current user is a member.
//
//	@Summary      Delete group
//	@Description  Delete a group (member only)
//	@Tags         groups
//	@Accept       json
//	@Produce      json
//	@Param        id   path  string  true  "Group UUID"
//	@Success      200  {object}  map[string]string
//	@Failure      400  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/groups/{id} [delete]
func DeleteGroupHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		gid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid group id", c.Params("id"))
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		// ensure membership
		ok, err := client.Group.Query().Where(group.IDEQ(gid), group.HasMembersWith(user.IDEQ(uid))).Exist(ctx)
		if err != nil {
			return kit.InternalError("query group failed", err.Error())
		}
		if !ok {
			return fiber.ErrForbidden
		}
		if err := client.Group.DeleteOneID(gid).Exec(ctx); err != nil {
			return kit.InternalError("delete failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}
