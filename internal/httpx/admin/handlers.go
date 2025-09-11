package admin

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/user"
	"fiber-ent-apollo-pg/internal/httpx/kit"
)

// PingHandler example protected route
//
//	@Summary      Admin Ping
//	@Description  Protected route requiring admin role
//	@Tags         admin
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {object}  map[string]string  "pong"
//	@Failure      401  {object}  map[string]interface{}  "unauthorized"
//	@Failure      403  {object}  map[string]interface{}  "forbidden"
//	@Router       /api/v1/admin/ping [get]
func PingHandler() fiber.Handler {
	return func(c *fiber.Ctx) error { return kit.OK(c, fiber.Map{"message": "pong"}) }
}

// PromoteUserHandler sets user.type=admin
//
//	@Summary      Promote user to admin
//	@Description  Set user.type = admin
//	@Tags         admin
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "User UUID"
//	@Success      200  {object}  map[string]string  "ok"
//	@Failure      400  {object}  map[string]interface{}
//	@Failure      401  {object}  map[string]interface{}
//	@Failure      403  {object}  map[string]interface{}
//	@Failure      404  {object}  map[string]interface{}
//	@Router       /api/v1/admin/users/{id}/promote [post]
func PromoteUserHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		uid, err := uuid.Parse(idStr)
		if err != nil {
			return kit.BadRequest("invalid user id", idStr)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		_, err = client.User.UpdateOneID(uid).SetType(user.TypeAdmin).Save(ctx)
		if err != nil {
			return kit.NotFound("user not found or update failed")
		}
		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}
