package auth

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/device"
	"fiber-ent-apollo-pg/ent/fingerprint"
	"fiber-ent-apollo-pg/ent/identity"
	"fiber-ent-apollo-pg/ent/visitor"
	"fiber-ent-apollo-pg/internal/config"
	"fiber-ent-apollo-pg/internal/httpx/kit"
	"fiber-ent-apollo-pg/internal/httpx/mw"
)

// AnonymousInitHandler initializes an anonymous visitor and returns JWTs.
//
//	@Summary      Anonymous Init
//	@Description  Initialize anonymous visitor, upsert device, issue tokens
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body  body   auth.AnonymousInitRequest  true  "anonymous init"
//	@Success      200   {object}  auth.TokenResponse
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/anonymous/init [post]
func AnonymousInitHandler(cfg *config.Config, client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req AnonymousInitRequest
		if err := c.BodyParser(&req); err != nil || req.DeviceID == "" {
			return kit.BadRequest("device_id required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		anonID := uuid.NewString()
		v, err := client.Visitor.Create().
			SetAnonID(anonID).
			SetNillablePrimaryFpHash(req.FPHash).
			Save(ctx)
		if err != nil {
			if req.FPHash != nil {
				v, err = client.Visitor.Query().Where(visitor.PrimaryFpHashEQ(*req.FPHash)).First(ctx)
			}
			if err != nil {
				return kit.InternalError("init anonymous failed", err.Error())
			}
		}

		now := time.Now().UTC()
		if d, err := client.Device.Query().Where(device.DeviceIDEQ(req.DeviceID)).First(ctx); err == nil {
			upd := client.Device.UpdateOne(d).SetLastSeenAt(now).SetVisitor(v)
			if req.Meta != nil {
				upd = upd.SetMeta(req.Meta)
			}
			if err := upd.Exec(ctx); err != nil {
				return kit.InternalError("update device failed", err.Error())
			}
		} else if ent.IsNotFound(err) {
			cr := client.Device.Create().SetDeviceID(req.DeviceID).SetVisitor(v).SetLastSeenAt(now)
			if req.Meta != nil {
				cr = cr.SetMeta(req.Meta)
			}
			if _, err := cr.Save(ctx); err != nil {
				return kit.InternalError("create device failed", err.Error())
			}
		} else if err != nil {
			return kit.InternalError("query device failed", err.Error())
		}

		sub := "visitor:" + v.ID.String()
		access, _, err := SignAccess(cfg, sub, "anon", nil, req.DeviceID)
		if err != nil {
			return kit.InternalError("sign access failed", err.Error())
		}
		refresh, _, err := SignRefresh(cfg, sub, "anon", req.DeviceID)
		if err != nil {
			return kit.InternalError("sign refresh failed", err.Error())
		}
		SetRefreshCookie(c, refresh, cfg.JWT.RefreshDays)

		return kit.OK(c, TokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: cfg.JWT.AccessMin * 60, AnonID: v.AnonID, DeviceID: req.DeviceID})
	}
}

// RefreshHandler issues a new access token using refresh cookie.
//
//	@Summary      Refresh Access Token
//	@Description  Mint new access token from refresh cookie
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Success      200   {object}  auth.TokenResponse
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/refresh [post]
func RefreshHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rt := c.Cookies("refresh_token")
		if rt == "" {
			return fiber.ErrUnauthorized
		}
		claims, err := ParseAndValidate(cfg, rt)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		access, _, err := SignAccess(cfg, claims.Subject, claims.Kind, claims.Roles, claims.DeviceID)
		if err != nil {
			return kit.InternalError("sign access failed", err.Error())
		}
		return kit.OK(c, TokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: cfg.JWT.AccessMin * 60, DeviceID: claims.DeviceID})
	}
}

// LogoutHandler clears refresh cookie
//
//	@Summary      Logout (clear refresh)
//	@Description  Clear refresh cookie; access tokens expire naturally
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Success      204   {string}  string  "no content"
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/logout [post]
func LogoutHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ClearRefreshCookie(c)
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// MeHandler returns auth context if present.
//
//	@Summary      Who am I
//	@Description  Return current auth context
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/me [get]
func MeHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ac, _ := c.Locals("auth").(*mw.AuthContext)
		if ac == nil {
			return fiber.ErrUnauthorized
		}
		return kit.OK(c, fiber.Map{"subject": ac.Subject, "kind": ac.Kind, "roles": ac.Roles, "device_id": ac.DeviceID})
	}
}

// FpSyncHandler updates fingerprint and device meta; works for anon (visitor) or user contexts.
//
//	@Summary      Fingerprint/Device Sync
//	@Description  Upsert device and fingerprint metadata; bind to current user/visitor
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body  body   auth.FpSyncRequest  true  "fingerprint/device sync"
//	@Success      200   {object}  map[string]interface{}
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/fp/sync [post]
func FpSyncHandler(client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req FpSyncRequest
		if err := c.BodyParser(&req); err != nil || req.DeviceID == "" {
			return kit.BadRequest("device_id required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		userID, visitorID := resolveAuthIDsForSync(ctx, c, client)
		if userID == nil && visitorID == nil {
			return fiber.ErrUnauthorized
		}

		now := time.Now().UTC()
		if err := upsertDevice(ctx, client, userID, visitorID, &req, now); err != nil {
			return err
		}
		if err := upsertFingerprint(ctx, client, visitorID, &req, now); err != nil {
			return err
		}

		return kit.OK(c, fiber.Map{"status": "ok"})
	}
}

// resolveAuthIDsForSync extracts userID/visitorID from auth context or headers.
func resolveAuthIDsForSync(ctx context.Context, c *fiber.Ctx, client *ent.Client) (*uuid.UUID, *uuid.UUID) {
	var userID *uuid.UUID
	var visitorID *uuid.UUID
	if ac, _ := c.Locals("auth").(*mw.AuthContext); ac != nil {
		if ac.Kind == "user" && strings.HasPrefix(ac.Subject, "user:") {
			if uid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "user:")); err == nil {
				userID = &uid
			}
		}
		if ac.Kind == "anon" && strings.HasPrefix(ac.Subject, "visitor:") {
			if vid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "visitor:")); err == nil {
				visitorID = &vid
			}
		}
	}
	if visitorID == nil {
		if an := c.Get("X-Anon-Id"); an != "" {
			if v, err := client.Visitor.Query().Where(visitor.AnonIDEQ(an)).Only(ctx); err == nil {
				visitorID = &v.ID
			}
		}
	}
	return userID, visitorID
}

// upsertDevice updates or creates the device and binds to user/visitor as needed.
func upsertDevice(ctx context.Context, client *ent.Client, userID, visitorID *uuid.UUID, req *FpSyncRequest, now time.Time) error {
	if d, err := client.Device.Query().Where(device.DeviceIDEQ(req.DeviceID)).First(ctx); err == nil {
		upd := client.Device.UpdateOne(d).SetLastSeenAt(now)
		if userID != nil {
			upd = upd.ClearVisitor().SetUserID(*userID)
		} else if visitorID != nil {
			upd = upd.ClearUser().SetVisitorID(*visitorID)
		}
		if req.Meta != nil {
			upd = upd.SetMeta(req.Meta)
		}
		if err := upd.Exec(ctx); err != nil {
			return kit.InternalError("update device failed", err.Error())
		}
		return nil
	} else if ent.IsNotFound(err) {
		cr := client.Device.Create().SetDeviceID(req.DeviceID).SetLastSeenAt(now)
		if userID != nil {
			cr = cr.SetUserID(*userID)
		}
		if visitorID != nil {
			cr = cr.SetVisitorID(*visitorID)
		}
		if req.Meta != nil {
			cr = cr.SetMeta(req.Meta)
		}
		if _, err := cr.Save(ctx); err != nil {
			return kit.InternalError("create device failed", err.Error())
		}
		return nil
	} else if err != nil {
		return kit.InternalError("query device failed", err.Error())
	}
	return nil
}

// upsertFingerprint updates or creates fingerprint; binds to visitor if available.
func upsertFingerprint(ctx context.Context, client *ent.Client, visitorID *uuid.UUID, req *FpSyncRequest, now time.Time) error {
	if req.FPHash == nil || *req.FPHash == "" {
		return nil
	}
	if f, err := client.Fingerprint.Query().Where(fingerprint.FpHashEQ(*req.FPHash)).First(ctx); err == nil {
		upd := client.Fingerprint.UpdateOne(f).SetLastSeenAt(now)
		if req.UAHash != nil {
			upd = upd.SetUaHash(*req.UAHash)
		}
		if req.IPHash != nil {
			upd = upd.SetIPHash(*req.IPHash)
		}
		if visitorID != nil {
			upd = upd.SetVisitorID(*visitorID)
		}
		if err := upd.Exec(ctx); err != nil {
			return kit.InternalError("update fingerprint failed", err.Error())
		}
		return nil
	} else if ent.IsNotFound(err) {
		cr := client.Fingerprint.Create().SetFpHash(*req.FPHash).SetLastSeenAt(now)
		if req.UAHash != nil {
			cr = cr.SetUaHash(*req.UAHash)
		}
		if req.IPHash != nil {
			cr = cr.SetIPHash(*req.IPHash)
		}
		if visitorID != nil {
			cr = cr.SetVisitorID(*visitorID)
		}
		if _, err := cr.Save(ctx); err != nil {
			return kit.InternalError("create fingerprint failed", err.Error())
		}
		return nil
	} else if err != nil {
		return kit.InternalError("query fingerprint failed", err.Error())
	}
	return nil
}

// LoginHandler authenticates a user via password identity and returns JWTs.
//
//	@Summary      Login (password)
//	@Description  Authenticate by identifier/password and issue tokens
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body  body   auth.LoginRequest  true  "login"
//	@Success      200   {object}  auth.TokenResponse
//	@Failure      401   {object}  map[string]interface{}
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/login [post]
func LoginHandler(cfg *config.Config, client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil || req.Identifier == "" || req.Password == "" {
			return kit.BadRequest("identifier and password required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		idn, err := client.Identity.Query().Where(identity.ProviderEQ(identity.ProviderPassword), identity.IdentifierEQ(req.Identifier)).WithUser().Only(ctx)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		if idn.SecretHash == nil || !VerifyPassword(req.Password, *idn.SecretHash) {
			return fiber.ErrUnauthorized
		}
		if idn.Edges.User == nil {
			return kit.InternalError("identity has no user", nil)
		}

		// Optional: merge current anonymous Visitor devices into this User
		var curVisitorID *uuid.UUID
		if an := c.Get("X-Anon-Id"); an != "" {
			if v, err := client.Visitor.Query().Where(visitor.AnonIDEQ(an)).Only(ctx); err == nil {
				curVisitorID = &v.ID
			}
		} else if ac, _ := c.Locals("auth").(*mw.AuthContext); ac != nil && ac.Kind == "anon" && strings.HasPrefix(ac.Subject, "visitor:") {
			if vid, err := uuid.Parse(strings.TrimPrefix(ac.Subject, "visitor:")); err == nil {
				curVisitorID = &vid
			}
		}
		if curVisitorID != nil {
			if tx, err := client.Tx(ctx); err == nil {
				defer func() { _ = tx.Rollback() }()
				if err := tx.Device.Update().Where(device.HasVisitorWith(visitor.IDEQ(*curVisitorID))).ClearVisitor().SetUser(idn.Edges.User).Exec(ctx); err == nil {
					_ = tx.Commit()
				}
			}
		}

		sub := "user:" + idn.Edges.User.ID.String()
		access, _, err := SignAccess(cfg, sub, "user", nil, req.DeviceID)
		if err != nil {
			return kit.InternalError("sign access failed", err.Error())
		}
		refresh, _, err := SignRefresh(cfg, sub, "user", req.DeviceID)
		if err != nil {
			return kit.InternalError("sign refresh failed", err.Error())
		}
		SetRefreshCookie(c, refresh, cfg.JWT.RefreshDays)
		return kit.OK(c, TokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: cfg.JWT.AccessMin * 60, DeviceID: req.DeviceID})
	}
}

// RegisterHandler creates a new user and a password identity, then returns JWTs.
//
//	@Summary      Register (password)
//	@Description  Create user + password identity, then issue tokens
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body  body   auth.RegisterRequest  true  "register"
//	@Success      200   {object}  auth.TokenResponse
//	@Failure      400   {object}  map[string]interface{}
//	@Failure      429   {object}  map[string]interface{}
//	@Header       200   {string}  X-RateLimit-Limit      "Requests per window"
//	@Header       200   {string}  X-RateLimit-Remaining  "Remaining requests"
//	@Header       429   {string}  Retry-After            "Seconds to wait"
//	@Router       /api/v1/auth/register [post]
func RegisterHandler(cfg *config.Config, client *ent.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil || req.Identifier == "" || req.Password == "" {
			return kit.BadRequest("identifier and password required", nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		hash, err := HashPassword(req.Password)
		if err != nil {
			return kit.InternalError("hash password failed", err.Error())
		}

		tx, err := client.Tx(ctx)
		if err != nil {
			return kit.InternalError("begin tx failed", err.Error())
		}
		defer func() { _ = tx.Rollback() }()

		u, err := tx.User.Create().SetDisplayName(req.DisplayName).Save(ctx)
		if err != nil {
			return kit.InternalError("create user failed", err.Error())
		}

		_, err = tx.Identity.Create().SetProvider(identity.ProviderPassword).SetIdentifier(req.Identifier).SetSecretHash(hash).SetUser(u).Save(ctx)
		if err != nil {
			return kit.BadRequest("identifier already exists", nil)
		}
		if err := tx.Commit(); err != nil {
			return kit.InternalError("commit failed", err.Error())
		}

		sub := "user:" + u.ID.String()
		access, _, err := SignAccess(cfg, sub, "user", nil, req.DeviceID)
		if err != nil {
			return kit.InternalError("sign access failed", err.Error())
		}
		refresh, _, err := SignRefresh(cfg, sub, "user", req.DeviceID)
		if err != nil {
			return kit.InternalError("sign refresh failed", err.Error())
		}
		SetRefreshCookie(c, refresh, cfg.JWT.RefreshDays)
		return kit.OK(c, TokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: cfg.JWT.AccessMin * 60, DeviceID: req.DeviceID})
	}
}
