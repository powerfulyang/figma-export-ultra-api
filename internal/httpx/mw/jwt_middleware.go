package mw

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AuthContext struct {
	Subject  string // user:<uuid> or visitor:<uuid>
	Kind     string // user | anon
	Roles    []string
	DeviceID string
}

// JWTMiddleware expects a validator function to parse token and return claims-like struct.
// To avoid tight coupling, pass a function that returns (subject, kind, roles, deviceID, error).
type TokenParser func(token string) (string, string, []string, string, error)

func JWTMiddlewareDynamic(parse TokenParser) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authz := c.Get("Authorization")
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			return c.Next()
		}
		token := strings.TrimSpace(authz[len("Bearer "):])
		sub, kind, roles, deviceID, err := parse(token)
		if err == nil && sub != "" {
			c.Locals("auth", &AuthContext{Subject: sub, Kind: kind, Roles: roles, DeviceID: deviceID})
		}
		return c.Next()
	}
}

// RequireUser enforces authenticated user (kind=user)
func RequireUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*AuthContext)
		if ac == nil || ac.Kind != "user" || ac.Subject == "" {
			return fiber.ErrUnauthorized
		}
		return c.Next()
	}
}

// RequireRoles enforces that the authenticated context has at least one of the roles.
func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*AuthContext)
		if ac == nil || ac.Kind == "" {
			return fiber.ErrUnauthorized
		}
		if len(roles) == 0 {
			return c.Next()
		}
		for _, need := range roles {
			for _, have := range ac.Roles {
				if have == need {
					return c.Next()
				}
			}
		}
		return fiber.ErrForbidden
	}
}
