// Package projects provides HTTP handlers for managing projects.
package projects

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/configitem"
	"fiber-ent-apollo-pg/ent/project"
	"fiber-ent-apollo-pg/ent/projectconfig"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/httpx/kit"
	"fiber-ent-apollo-pg/internal/httpx/mw"
)

// CreateProjectRequest is the request body for creating a project
// swagger:model CreateProjectRequest
type CreateProjectRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// UpdateProjectRequest is the request body for updating a project
// swagger:model UpdateProjectRequest
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	Description *string `json:"description,omitempty"`
}

// AddConfigToProjectRequest is the request body for adding config to project
// swagger:model AddConfigToProjectRequest
type AddConfigToProjectRequest struct {
	ConfigID uuid.UUID `json:"config_id"`
	Active   bool      `json:"active,omitempty"`
}

// SetActiveConfigRequest is the request body for setting active config
// swagger:model SetActiveConfigRequest
type SetActiveConfigRequest struct {
	ConfigID uuid.UUID `json:"config_id"`
}

// ListProjectsHandler lists projects owned by the current user.
//
//	@Summary      List my projects
//	@Description  Returns projects owned by the current user
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        limit       query   int     false  "page size"      default(20)
//	@Param        offset      query   int     false  "offset"         default(0)
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Router       /api/v1/projects [get]
func ListProjectsHandler(client *ent.Client) fiber.Handler {
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

		q := client.Project.Query().Where(project.HasOwnerWith(user.IDEQ(uid))).Order(ent.Desc(project.FieldUpdatedAt))
		items, err := q.Limit(pg.Limit).Offset(pg.Offset).All(ctx)
		if err != nil {
			return kit.InternalError("query projects failed", err.Error())
		}

		nextOff := pg.Offset + len(items)
		meta := kit.PageMeta{Limit: pg.Limit, Offset: pg.Offset, Count: len(items), NextOffset: &nextOff, HasMore: len(items) == pg.Limit, Mode: "offset"}
		return kit.List(c, items, meta)
	}
}

// CreateProjectHandler creates a new project owned by the current user.
//
//	@Summary      Create project
//	@Description  Create a project owned by the current user
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        body  body  projects.CreateProjectRequest  true  "project payload"
//	@Success      201   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      409   {object}  map[string]interface{}
//	@Router       /api/v1/projects [post]
func CreateProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		var req CreateProjectRequest
		if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.URL) == "" {
			return kit.BadRequest("name and url required", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		// Check if URL already exists for this user
		exists, err := client.Project.Query().
			Where(project.And(project.HasOwnerWith(user.IDEQ(uid)), project.URLEQ(req.URL))).
			Exist(ctx)
		if err != nil {
			return kit.InternalError("check project url failed", err.Error())
		}
		if exists {
			return kit.BadRequest("project url already exists for this user", req.URL)
		}

		created, err := client.Project.Create().
			SetName(req.Name).
			SetURL(req.URL).
			SetNillableDescription(&req.Description).
			SetOwnerID(uid).
			Save(ctx)
		if err != nil {
			return kit.InternalError("create project failed", err.Error())
		}
		return kit.Created(c, created)
	}
}

// GetProjectHandler gets a single project by ID.
//
//	@Summary      Get project
//	@Description  Get project details (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id   path  string  true  "Project UUID"
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id} [get]
func GetProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		proj, err := client.Project.Query().
			Where(project.IDEQ(projID)).
			WithOwner().
			WithProjectConfigs(func(q *ent.ProjectConfigQuery) {
				q.WithConfigItem()
			}).
			Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		return kit.OK(c, proj)
	}
}

// UpdateProjectHandler updates a project owned by the current user.
//
//	@Summary      Update project
//	@Description  Update project details (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id    path  string                        true  "Project UUID"
//	@Param        body  body  projects.UpdateProjectRequest true  "project payload"
//	@Success      200   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      403   {object}  map[string]interface{}
//	@Failure      404   {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id} [put]
func UpdateProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		var req UpdateProjectRequest
		if err := c.BodyParser(&req); err != nil {
			return kit.BadRequest("invalid request body", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		upd := client.Project.UpdateOneID(projID)
		if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
			upd = upd.SetName(*req.Name)
		}
		if req.URL != nil && strings.TrimSpace(*req.URL) != "" {
			// Check if new URL conflicts with existing projects for this user
			exists, err := client.Project.Query().
				Where(project.And(
					project.HasOwnerWith(user.IDEQ(ownerID)),
					project.URLEQ(*req.URL),
					project.IDNEQ(projID),
				)).
				Exist(ctx)
			if err != nil {
				return kit.InternalError("check project url failed", err.Error())
			}
			if exists {
				return kit.BadRequest("project url already exists for this user", *req.URL)
			}
			upd = upd.SetURL(*req.URL)
		}
		if req.Description != nil {
			upd = upd.SetDescription(*req.Description)
		}

		updated, err := upd.Save(ctx)
		if err != nil {
			return kit.InternalError("update project failed", err.Error())
		}
		return kit.OK(c, updated)
	}
}

// DeleteProjectHandler deletes a project owned by the current user.
//
//	@Summary      Delete project
//	@Description  Delete a project (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id   path  string  true  "Project UUID"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id} [delete]
func DeleteProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Delete project configs first (cascade delete)
		_, err = client.ProjectConfig.Delete().Where(projectconfig.HasProjectWith(project.IDEQ(projID))).Exec(ctx)
		if err != nil {
			return kit.InternalError("delete project configs failed", err.Error())
		}

		if err := client.Project.DeleteOneID(projID).Exec(ctx); err != nil {
			return kit.InternalError("delete failed", err.Error())
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// AddConfigToProjectHandler adds a config to a project.
//
//	@Summary      Add config to project
//	@Description  Add a config item to project (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id    path  string                             true  "Project UUID"
//	@Param        body  body  projects.AddConfigToProjectRequest true  "config payload"
//	@Success      200   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      403   {object}  map[string]interface{}
//	@Failure      404   {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id}/configs [post]
func AddConfigToProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		var req AddConfigToProjectRequest
		if err := c.BodyParser(&req); err != nil {
			return kit.BadRequest("invalid request body", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		// Verify project ownership
		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Verify config ownership
		cfg, err := client.ConfigItem.Query().Where(configitem.IDEQ(req.ConfigID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("config not found")
		}
		if cfg.Edges.Owner == nil || cfg.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Check if config already associated with this project
		exists, err := client.ProjectConfig.Query().
			Where(projectconfig.And(
				projectconfig.HasProjectWith(project.IDEQ(projID)),
				projectconfig.HasConfigItemWith(configitem.IDEQ(req.ConfigID)),
			)).
			Exist(ctx)
		if err != nil {
			return kit.InternalError("check project config failed", err.Error())
		}
		if exists {
			return kit.BadRequest("config already associated with this project", nil)
		}

		// If setting this as active, deactivate others first
		if req.Active {
			_, err = client.ProjectConfig.Update().
				Where(projectconfig.HasProjectWith(project.IDEQ(projID))).
				SetActive(false).
				Save(ctx)
			if err != nil {
				return kit.InternalError("deactivate existing configs failed", err.Error())
			}
		}

		// Create the association
		projConfig, err := client.ProjectConfig.Create().
			SetProjectID(projID).
			SetConfigItemID(req.ConfigID).
			SetActive(req.Active).
			Save(ctx)
		if err != nil {
			return kit.InternalError("create project config failed", err.Error())
		}

		return kit.Created(c, projConfig)
	}
}

// RemoveConfigFromProjectHandler removes a config from a project.
//
//	@Summary      Remove config from project
//	@Description  Remove a config item from project (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id         path  string  true  "Project UUID"
//	@Param        config_id  path  string  true  "Config UUID"
//	@Success      200        {object}  map[string]string
//	@Failure      401        {object}  map[string]interface{}
//	@Failure      403        {object}  map[string]interface{}
//	@Failure      404        {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id}/configs/{config_id} [delete]
func RemoveConfigFromProjectHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		configID, err := uuid.Parse(c.Params("config_id"))
		if err != nil {
			return kit.BadRequest("invalid config id", c.Params("config_id"))
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		// Verify project ownership
		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Delete the association
		deleted, err := client.ProjectConfig.Delete().
			Where(projectconfig.And(
				projectconfig.HasProjectWith(project.IDEQ(projID)),
				projectconfig.HasConfigItemWith(configitem.IDEQ(configID)),
			)).
			Exec(ctx)
		if err != nil {
			return kit.InternalError("remove project config failed", err.Error())
		}
		if deleted == 0 {
			return kit.NotFound("project config association not found")
		}

		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// SetActiveConfigHandler sets the active config for a project.
//
//	@Summary      Set active config
//	@Description  Set which config is active for a project (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id    path  string                           true  "Project UUID"
//	@Param        body  body  projects.SetActiveConfigRequest  true  "config payload"
//	@Success      200   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      403   {object}  map[string]interface{}
//	@Failure      404   {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id}/active-config [put]
func SetActiveConfigHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		var req SetActiveConfigRequest
		if err := c.BodyParser(&req); err != nil {
			return kit.BadRequest("invalid request body", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		// Verify project ownership
		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Verify the config is associated with this project
		projConfig, err := client.ProjectConfig.Query().
			Where(projectconfig.And(
				projectconfig.HasProjectWith(project.IDEQ(projID)),
				projectconfig.HasConfigItemWith(configitem.IDEQ(req.ConfigID)),
			)).
			Only(ctx)
		if err != nil {
			return kit.NotFound("project config association not found")
		}

		// Deactivate all configs for this project
		_, err = client.ProjectConfig.Update().
			Where(projectconfig.HasProjectWith(project.IDEQ(projID))).
			SetActive(false).
			Save(ctx)
		if err != nil {
			return kit.InternalError("deactivate configs failed", err.Error())
		}

		// Activate the specified config
		updated, err := client.ProjectConfig.UpdateOneID(projConfig.ID).
			SetActive(true).
			Save(ctx)
		if err != nil {
			return kit.InternalError("activate config failed", err.Error())
		}

		return kit.OK(c, updated)
	}
}

// ListProjectConfigsHandler lists all configs associated with a project.
//
//	@Summary      List project configs
//	@Description  List configs associated with a project (owner only)
//	@Tags         projects
//	@Accept       json
//	@Produce      json
//	@Param        id   path  string  true  "Project UUID"
//	@Success      200  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/projects/{id}/configs [get]
func ListProjectConfigsHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil || ac.Kind != "user" || !strings.HasPrefix(ac.Subject, "user:") {
			return fiber.ErrUnauthorized
		}
		ownerID, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:"))
		if err != nil {
			return fiber.ErrUnauthorized
		}

		projID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return kit.BadRequest("invalid project id", c.Params("id"))
		}

		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		// Verify project ownership
		proj, err := client.Project.Query().Where(project.IDEQ(projID)).WithOwner().Only(ctx)
		if err != nil {
			return kit.NotFound("project not found")
		}
		if proj.Edges.Owner == nil || proj.Edges.Owner.ID != ownerID {
			return fiber.ErrForbidden
		}

		// Get project configs with config item details
		projConfigs, err := client.ProjectConfig.Query().
			Where(projectconfig.HasProjectWith(project.IDEQ(projID))).
			WithConfigItem().
			Order(ent.Desc(projectconfig.FieldActive), ent.Desc(projectconfig.FieldUpdatedAt)).
			All(ctx)
		if err != nil {
			return kit.InternalError("query project configs failed", err.Error())
		}

		return kit.OK(c, projConfigs)
	}
}
