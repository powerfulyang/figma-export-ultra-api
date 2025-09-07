package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestCursorEncodeDecode_Binary(t *testing.T) {
	id := uuid.New().String()
	ts := time.Unix(1700000000, 123456789).UTC()
	enc := encodeCursor(id, ts)
	got, err := decodeCursor(enc)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.ID != id {
		t.Fatalf("id mismatch: %s != %s", got.ID, id)
	}
	if !got.TS.Equal(ts) {
		t.Fatalf("ts mismatch: %v != %v", got.TS, ts)
	}
}

func TestParsePaging_FixedSnapshot(t *testing.T) {
	app := fiber.New()
	app.Get("/t", func(c *fiber.Ctx) error {
		p, err := parsePaging(c)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if p.Mode != "snapshot" {
			t.Fatalf("expected snapshot mode, got %s", p.Mode)
		}
		if p.Snapshot == nil {
			t.Fatal("expected snapshot not nil")
		}
		return c.SendStatus(200)
	})
	req := httptestNewRequest("GET", "/t?fixed=true&limit=10", nil)
	_, _ = app.Test(req)
}

// minimal wrapper: Fiber's test uses net/http/httptest's NewRequest
func httptestNewRequest(method, target string, body any) *http.Request {
	var r io.Reader
	if body != nil {
		if b, ok := body.([]byte); ok {
			r = bytes.NewReader(b)
		}
	}
	req := httptest.NewRequest(method, target, r)
	return req
}
