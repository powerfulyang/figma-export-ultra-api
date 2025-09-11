package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/identity"
	"fiber-ent-apollo-pg/internal/config"
	// kit imported by testutil
	testutil "fiber-ent-apollo-pg/internal/httpx/kit/testutil"
)

func newTestApp(t *testing.T, client *ent.Client, cfg *config.Config) *fiber.App {
	t.Helper()
	return testutil.NewApp(
		func(app *fiber.App) { app.Post("/auth/anonymous/init", AnonymousInitHandler(cfg, client)) },
		func(app *fiber.App) { app.Post("/auth/login", LoginHandler(cfg, client)) },
		func(app *fiber.App) { app.Post("/auth/fp/sync", FpSyncHandler(client)) },
	)
}

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := "file:ent?mode=memory&cache=shared&_fk=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Ensure foreign keys enabled
	_, _ = db.Exec("PRAGMA foreign_keys = ON")
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	ctx, cancel := contextWithT(t)
	defer cancel()
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return client
}

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	// Default minimal JWT config (HS256 fallback)
	cfg.JWT.Algo = "HS256"
	cfg.JWT.HSSecret = "test-secret"
	cfg.JWT.Issuer = "test"
	cfg.JWT.Audience = "test"
	cfg.JWT.AccessMin = 15
	cfg.JWT.RefreshDays = 7
	return cfg
}

func TestAnonymousInit_CreatesVisitorAndDevice(t *testing.T) {
	client := newTestClient(t)
	cfg := newTestConfig()
	app := newTestApp(t, client, cfg)

	body := AnonymousInitRequest{DeviceID: "dev-1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/anonymous/init", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var env struct {
		Code string
		Data TokenResponse
	}
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := env.Data
	if out.AccessToken == "" || out.AnonID == "" {
		t.Fatalf("missing tokens/anon_id: %+v", out)
	}

	// Assert visitor/device exist
	ctx, cancel := contextWithT(t)
	defer cancel()
	if n, err := client.Visitor.Query().Count(ctx); err != nil || n != 1 {
		t.Fatalf("visitors=%d err=%v", n, err)
	}
	if n, err := client.Device.Query().Count(ctx); err != nil || n != 1 {
		t.Fatalf("devices=%d err=%v", n, err)
	}
}

func TestLogin_Password_IssuesToken(t *testing.T) {
	client := newTestClient(t)
	cfg := newTestConfig()
	app := newTestApp(t, client, cfg)

	// Arrange: create user + identity
	ctx, cancel := contextWithT(t)
	defer cancel()
	u, err := client.User.Create().SetDisplayName("Alice").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	hash, err := HashPassword("P@ssw0rd")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	_, err = client.Identity.Create().SetProvider(identity.ProviderPassword).SetIdentifier("alice@example.com").SetSecretHash(hash).SetUser(u).Save(ctx)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	// Act
	in := LoginRequest{Identifier: "alice@example.com", Password: "P@ssw0rd", DeviceID: "dev-2"}
	b, _ := json.Marshal(in)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var env struct{ Data TokenResponse }
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.AccessToken == "" {
		t.Fatalf("missing access_token")
	}
}

func TestFpSync_WithAnonHeader_BindsFingerprint(t *testing.T) {
	client := newTestClient(t)
	cfg := newTestConfig()
	app := newTestApp(t, client, cfg)

	// First, anon init to get anon_id
	initReq := AnonymousInitRequest{DeviceID: "dev-3"}
	b, _ := json.Marshal(initReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/anonymous/init", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("anon init request: %v", err)
	}
	var initEnv struct{ Data TokenResponse }
	if err := json.NewDecoder(res.Body).Decode(&initEnv); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Now fp sync with X-Anon-Id
	syncReq := FpSyncRequest{DeviceID: "dev-3", FPHash: strPtr("fphash-1"), UAHash: strPtr("uahash"), IPHash: strPtr("iph")}
	sb, _ := json.Marshal(syncReq)
	sreq := httptest.NewRequest(http.MethodPost, "/auth/fp/sync", bytes.NewReader(sb))
	sreq.Header.Set("Content-Type", "application/json")
	sreq.Header.Set("X-Anon-Id", initEnv.Data.AnonID)
	sres, err := app.Test(sreq)
	if err != nil {
		t.Fatalf("fp sync request: %v", err)
	}
	if sres.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", sres.StatusCode)
	}

	// Assert fingerprint bound to some visitor
	ctx, cancel := contextWithT(t)
	defer cancel()
	f, err := client.Fingerprint.Query().First(ctx)
	if err != nil {
		t.Fatalf("fingerprint not found: %v", err)
	}
	if _, err := f.QueryVisitor().Only(ctx); err != nil {
		t.Fatalf("fingerprint not bound to visitor: %v", err)
	}
}

// helpers
func strPtr(s string) *string { return &s }

func contextWithT(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 5*time.Second)
}
