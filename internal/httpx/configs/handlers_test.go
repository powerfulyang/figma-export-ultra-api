package configs

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
	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/internal/httpx/kit/testutil"
	"fiber-ent-apollo-pg/internal/httpx/mw"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := "file:ent?mode=memory&cache=shared&_fk=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, _ = db.Exec("PRAGMA foreign_keys = ON")
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return client
}

func TestConfig_Create_List_Share_Unshare(t *testing.T) {
	client := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	owner, err := client.User.Create().SetDisplayName("Owner").Save(ctx)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	target, err := client.User.Create().SetDisplayName("Target").Save(ctx)
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	// App for owner
	appOwner := testutil.NewApp(
		func(app *fiber.App) {
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("auth", &mw.AuthContext{Subject: "user:" + owner.ID.String(), Kind: "user"})
				return c.Next()
			})
		},
		func(app *fiber.App) { app.Post("/configs", mw.RequireUser(), CreateConfigHandler(client)) },
		func(app *fiber.App) { app.Get("/configs", mw.RequireUser(), ListConfigsHandler(client)) },
		func(app *fiber.App) { app.Get("/configs/visible", mw.RequireUser(), VisibleConfigsHandler(client)) },
		func(app *fiber.App) {
			app.Post("/configs/:id/share/user/:user_id", mw.RequireUser(), ShareToUserHandler(client))
		},
		func(app *fiber.App) {
			app.Post("/configs/:id/unshare/user/:user_id", mw.RequireUser(), UnshareFromUserHandler(client))
		},
	)

	// Create config
	body := map[string]any{"name": "Cfg1", "data": map[string]any{"k": "v"}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/configs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := appOwner.Test(req)
	if err != nil {
		t.Fatalf("create cfg req: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var envCreated struct{ Data struct{ ID uuid.UUID } }
	if err := json.NewDecoder(res.Body).Decode(&envCreated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	cfgID := envCreated.Data.ID

	// List my configs
	lreq := httptest.NewRequest(http.MethodGet, "/configs", nil)
	lres, err := appOwner.Test(lreq)
	if err != nil {
		t.Fatalf("list req: %v", err)
	}
	if lres.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", lres.StatusCode)
	}
	var envList struct{ Data []map[string]any }
	if err := json.NewDecoder(lres.Body).Decode(&envList); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(envList.Data) != 1 {
		t.Fatalf("want 1, got %d", len(envList.Data))
	}

	// Target visible before share: 0
	appTarget := testutil.NewApp(
		func(app *fiber.App) {
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("auth", &mw.AuthContext{Subject: "user:" + target.ID.String(), Kind: "user"})
				return c.Next()
			})
		},
		func(app *fiber.App) { app.Get("/configs/visible", mw.RequireUser(), VisibleConfigsHandler(client)) },
	)
	vreq := httptest.NewRequest(http.MethodGet, "/configs/visible", nil)
	vres, err := appTarget.Test(vreq)
	if err != nil {
		t.Fatalf("visible req: %v", err)
	}
	var envVis struct{ Data []map[string]any }
	if err := json.NewDecoder(vres.Body).Decode(&envVis); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(envVis.Data) != 0 {
		t.Fatalf("target visible pre-share: %d", len(envVis.Data))
	}

	// Share to target
	sreq := httptest.NewRequest(http.MethodPost, "/configs/"+cfgID.String()+"/share/user/"+target.ID.String(), nil)
	sres, err := appOwner.Test(sreq)
	if err != nil {
		t.Fatalf("share req: %v", err)
	}
	if sres.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", sres.StatusCode)
	}

	// Target visible after share: 1
	vres2, _ := appTarget.Test(httptest.NewRequest(http.MethodGet, "/configs/visible", nil))
	if err := json.NewDecoder(vres2.Body).Decode(&envVis); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(envVis.Data) != 1 {
		t.Fatalf("target visible post-share: %d", len(envVis.Data))
	}

	// Unshare
	ureq := httptest.NewRequest(http.MethodPost, "/configs/"+cfgID.String()+"/unshare/user/"+target.ID.String(), nil)
	ures, err := appOwner.Test(ureq)
	if err != nil {
		t.Fatalf("unshare req: %v", err)
	}
	if ures.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", ures.StatusCode)
	}

	// Target visible after unshare: 0
	vres3, _ := appTarget.Test(httptest.NewRequest(http.MethodGet, "/configs/visible", nil))
	if err := json.NewDecoder(vres3.Body).Decode(&envVis); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(envVis.Data) != 0 {
		t.Fatalf("target visible post-unshare: %d", len(envVis.Data))
	}
}
