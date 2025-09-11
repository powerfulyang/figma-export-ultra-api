// Package mw contains HTTP middleware including authentication and rate limiting.
package mw

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthContext holds authentication details extracted from JWT.
type AuthContext struct {
	Subject  string // user:<uuid> or visitor:<uuid>
	Kind     string // user | anon
	Roles    []string
	DeviceID string
}

// TokenParser parses a token string and returns subject, kind, roles, deviceID.
type TokenParser func(token string) (string, string, []string, string, error)

// JWTMiddlewareDynamic attaches auth context parsed by the given token parser.
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
