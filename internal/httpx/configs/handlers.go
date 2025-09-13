// Package configs provides HTTP handlers for managing configuration items.
package configs

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/configitem"
	"fiber-ent-apollo-pg/ent/group"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/httpx/kit"
	"fiber-ent-apollo-pg/internal/httpx/mw"
)

// CreateConfigRequest is the request body for creating a config item
// swagger:model CreateConfigRequest
type CreateConfigRequest struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

// ShareToGroupsRequest is the request body for sharing a config to groups
// swagger:model ShareToGroupsRequest
type ShareToGroupsRequest struct {
	GroupIDs []uuid.UUID `json:"group_ids"`
}

// ListConfigsHandler lists configs owned by the current user.
//
//	@Summary      List my configs
//	@Description  Returns configs owned by the current user
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        limit       query   int     false  "page size"      default(20)
//	@Param        offset      query   int     false  "offset"         default(0)
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Router       /api/v1/configs [get]
func ListConfigsHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		pg, err := kit.ParsePaging(c)
		if err != nil {
			return err
		}

		q := client.ConfigItem.Query().Where(configitem.HasOwnerWith(user.IDEQ(uid))).Order(ent.Desc(configitem.FieldUpdatedAt))
		items, err := q.Limit(pg.Limit).Offset(pg.Offset).All(ctx)
		if err != nil {
			return kit.InternalError("query configs failed", err.Error())
		}

		nextOff := pg.Offset + len(items)
		meta := kit.PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(items), NextOffset: &nextOff, HasMore: len(items) == pg.Limit, Mode: "offset"}
		return kit.List(c, items, meta)
	}
}

// CreateConfigHandler creates a new config owned by the current user.
//
//	@Summary      Create config
//	@Description  Create a config owned by the current user
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        body  body  configs.CreateConfigRequest  true  "config payload"
//	@Success      201   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Router       /api/v1/configs [post]
func CreateConfigHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		var req CreateConfigRequest
		if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			return kit.BadRequest("name required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		created, err := client.ConfigItem.Create().SetName(req.Name).SetData(req.Data).SetOwnerID(uid).Save(ctx)
		if err != nil {
			return kit.InternalError("create config failed", err.Error())
		}
		return kit.Created(c, created)
	}
}

// DeleteConfigHandler deletes a config owned by the current user.
//
//	@Summary      Delete config
//	@Description  Delete a config (owner only)
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        id   path  string  true  "Config UUID"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/configs/{id} [delete]
func DeleteConfigHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		cfgID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid config id", c.Params("id"))
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(cfgID)).WithOwner().Only(ctx)
		if err != nil || cfg.Edges.Owner == nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}
		if err := client.ConfigItem.DeleteOneID(cfgID).Exec(ctx); err != nil {
			return kit.InternalError("delete failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// ShareToGroupsHandler shares a config to given groups.
//
//	@Summary      Share to groups
//	@Description  Share a config to specified groups (owner only)
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        id     path  string                    true  "Config UUID"
//	@Param        body   body  configs.ShareToGroupsRequest  true  "group ids"
//	@Success      200    {object}  map[string]string
//	@Failure      400    {object}  map[string]interface{}
//	@Failure      401    {object}  map[string]interface{}
//	@Failure      404    {object}  map[string]interface{}
//	@Router       /api/v1/configs/{id}/share/groups [post]
func ShareToGroupsHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		idStr := c.Params("id")
		cfgID, err := uuid.Parse(idStr)
		if err != nil {
			return kit.BadRequest("invalid config id", idStr)
		}
		var req ShareToGroupsRequest
		if err := c.BodyParser(&req); err != nil || len(req.GroupIDs) == 0 {
			return kit.BadRequest("group_ids required", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(cfgID)).WithOwner().Only(ctx)
		if err != nil || cfg.Edges.Owner == nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner.ID != uid {
			return fiber.ErrForbidden
		}

		upd := client.ConfigItem.UpdateOneID(cfgID).AddSharedGroupIDs(req.GroupIDs...)
		if err := upd.Exec(ctx); err != nil {
			return kit.InternalError("share failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// UnshareFromGroupsHandler removes sharing of a config from specified groups.
//
//	@Summary      Unshare from groups
//	@Description  Remove group sharing (owner only)
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        id     path  string                    true  "Config UUID"
//	@Param        body   body  configs.ShareToGroupsRequest  true  "group ids"
//	@Success      200    {object}  map[string]string
//	@Failure      400    {object}  map[string]interface{}
//	@Failure      401    {object}  map[string]interface{}
//	@Failure      403    {object}  map[string]interface{}
//	@Failure      404    {object}  map[string]interface{}
//	@Router       /api/v1/configs/{id}/unshare/groups [post]
func UnshareFromGroupsHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		cfgID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid config id", c.Params("id"))
		}
		var req ShareToGroupsRequest
		if err := c.BodyParser(&req); err != nil || len(req.GroupIDs) == 0 {
			return kit.BadRequest("group_ids required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(cfgID)).WithOwner().Only(ctx)
		if err != nil || cfg.Edges.Owner == nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner.ID != uid {
			return fiber.ErrForbidden
		}
		if err := client.ConfigItem.UpdateOneID(cfgID).RemoveSharedGroupIDs(req.GroupIDs...).Exec(ctx); err != nil {
			return kit.InternalError("unshare failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// ShareToUserHandler shares a config to a single user by creating/finding a 2-person group.
//
//	@Summary      Share to user
//	@Description  Share a config to a user (creates a 2-person group if needed)
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        id        path  string  true  "Config UUID"
//	@Param        user_id   path  string  true  "Target User UUID"
//	@Success      200       {object}  map[string]interface{}
//	@Failure      400       {object}  map[string]interface{}
//	@Failure      401       {object}  map[string]interface{}
//	@Failure      403       {object}  map[string]interface{}
//	@Failure      404       {object}  map[string]interface{}
//	@Router       /api/v1/configs/{id}/share/user/{user_id} [post]
func ShareToUserHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		cfgID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid config id", c.Params("id"))
		}
		targetID, err := uuid.Parse(c.Params("user_id"))
		if err != nil {
			return kit.BadRequest("invalid user id", c.Params("user_id"))
		}

		if targetID == ownerID {
			return kit.BadRequest("cannot share to self", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		// Ensure owner has this config
		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(cfgID)).WithOwner().Only(ctx)
		if err != nil || cfg.Edges.Owner == nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Try find an existing group that has exactly two members: owner and target.
		cand, err := client.Group.Query().
			Where(group.HasMembersWith(user.IDIn(ownerID, targetID))).
			All(ctx)
		if err != nil {
			return kit.InternalError("query groups failed", err.Error())
		}
		var dm *ent.Group
		for _, g := range cand {
			// fetch members count and ids
			members, err := g.QueryMembers().All(ctx)
			if err != nil {
				continue
			}
			if len(members) == 2 {
				ids := map[uuid.UUID]bool{members[0].ID: true, members[1].ID: true}
				if ids[ownerID] && ids[targetID] {
					dm = g
					break
				}
			}
		}
		if dm == nil {
			// create a new 2-person group
			name := "dm:" + ownerID.String() + ":" + targetID.String()
			var err error
			dm, err = client.Group.Create().SetName(name).Save(ctx)
			if err != nil {
				return kit.InternalError("create group failed", err.Error())
			}
			if err := client.Group.UpdateOne(dm).AddMemberIDs(ownerID, targetID).Exec(ctx); err != nil {
				return kit.InternalError("add members failed", err.Error())
			}
		}

		// Share config to the group
		if err := client.ConfigItem.UpdateOneID(cfgID).AddSharedGroupIDs(dm.ID).Exec(ctx); err != nil {
			return kit.InternalError("share failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok", "group_id": dm.ID})
	}
}

// UnshareFromUserHandler removes sharing of a config from a 2-person group with the target user (if exists).
//
//	@Summary      Unshare from user
//	@Description  Remove sharing from the DM group (owner only)
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        id        path  string  true  "Config UUID"
//	@Param        user_id   path  string  true  "Target User UUID"
//	@Success      200       {object}  map[string]interface{}
//	@Failure      400       {object}  map[string]interface{}
//	@Failure      401       {object}  map[string]interface{}
//	@Failure      403       {object}  map[string]interface{}
//	@Failure      404       {object}  map[string]interface{}
//	@Router       /api/v1/configs/{id}/unshare/user/{user_id} [post]
func UnshareFromUserHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		cfgID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid config id", c.Params("id"))
		}
		targetID, err := uuid.Parse(c.Params("user_id"))
		if err != nil {
			return kit.BadRequest("invalid user id", c.Params("user_id"))
		}

		if targetID == ownerID {
			return kit.BadRequest("cannot unshare from self", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(cfgID)).WithOwner().Only(ctx)
		if err != nil || cfg.Edges.Owner == nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		cand, err := client.Group.Query().Where(group.HasMembersWith(user.IDIn(ownerID, targetID))).All(ctx)
		if err != nil {
			return kit.InternalError("query groups failed", err.Error())
		}
		var dm *ent.Group
		for _, g := range cand {
			members, err := g.QueryMembers().All(ctx)
			if err != nil {
				continue
			}
			if len(members) == 2 {
				ids := map[uuid.UUID]bool{members[0].ID: true, members[1].ID: true}
				if ids[ownerID] && ids[targetID] {
					dm = g
					break
				}
			}
		}
		if dm == nil {
			return kit.NotFound("dm group not found")
		}
		if err := client.ConfigItem.UpdateOneID(cfgID).RemoveSharedGroupIDs(dm.ID).Exec(ctx); err != nil {
			return kit.InternalError("unshare failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok", "group_id": dm.ID})
	}
}

// VisibleConfigsHandler lists configs the current user can see (owned or shared via groups).
//
//	@Summary      List visible configs
//	@Description  Configs owned by me or shared to my groups
//	@Tags         configs
//	@Accept       json
//	@Produce      json
//	@Param        limit       query   int     false  "page size"      default(20)
//	@Param        offset      query   int     false  "offset"         default(0)
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Router       /api/v1/configs/visible [get]
func VisibleConfigsHandler(client *ent.Client) fiber.Handler {
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
		// gather group IDs where current user is a member
		gids, err := client.Group.Query().Where(group.HasMembersWith(user.IDEQ(uid))).IDs(ctx)
		if err != nil {
			return kit.InternalError("query groups failed", err.Error())
		}
		q := client.ConfigItem.Query().
			Where(
				configitem.Or(
					configitem.HasOwnerWith(user.IDEQ(uid)),
					configitem.HasSharedGroupsWith(group.IDIn(gids...)),
				),
			).
			Order(ent.Desc(configitem.FieldUpdatedAt))
		items, err := q.Limit(pg.Limit).Offset(pg.Offset).All(ctx)
		if err != nil {
			return kit.InternalError("query configs failed", err.Error())
		}
		nextOff := pg.Offset + len(items)
		meta := kit.PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(items), NextOffset: &nextOff, HasMore: len(items) == pg.Limit, Mode: "offset"}
		return kit.List(c, items, meta)
	}
}
