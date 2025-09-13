package groups

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
	"fiber-ent-apollo-pg/ent/group"
	"fiber-ent-apollo-pg/ent/user"
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

func TestGroups_Create_List_Delete(t *testing.T) {
	client := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u1, err := client.User.Create().SetDisplayName("U1").Save(ctx)
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	u2, err := client.User.Create().SetDisplayName("U2").Save(ctx)
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	app := testutil.NewApp(
		func(app *fiber.App) {
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("auth", &mw.AuthContext{Subject: "user:" + u1.ID.String(), Kind: "user"})
				return c.Next()
			})
		},
		func(app *fiber.App) { app.Post("/groups", mw.RequireUser(), CreateGroupHandler(client)) },
		func(app *fiber.App) { app.Get("/groups", mw.RequireUser(), ListMyGroupsHandler(client)) },
		func(app *fiber.App) { app.Delete("/groups/:id", mw.RequireUser(), DeleteGroupHandler(client)) },
	)

	// create group with u2
	payload := map[string]any{"name": "G1", "member_ids": []string{u2.ID.String()}}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var env struct{ Data struct{ ID uuid.UUID } }
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// list groups: expect >=1
	lres, _ := app.Test(httptest.NewRequest(http.MethodGet, "/groups", nil))
	var list struct{ Data []map[string]any }
	if err := json.NewDecoder(lres.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list.Data) == 0 {
		t.Fatalf("expected groups")
	}

	// ensure membership
	ok, err := client.Group.Query().Where(group.IDEQ(env.Data.ID), group.HasMembersWith(user.IDEQ(u1.ID))).Exist(ctx)
	if err != nil || !ok {
		t.Fatalf("membership missing: %v", err)
	}

	// delete
	dres, _ := app.Test(httptest.NewRequest(http.MethodDelete, "/groups/"+env.Data.ID.String(), nil))
	if dres.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", dres.StatusCode)
	}
}
