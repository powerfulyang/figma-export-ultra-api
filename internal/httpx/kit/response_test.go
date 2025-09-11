package kit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestOKEnvelope(t *testing.T) {
	app := fiber.New()
	app.Get("/t", func(c *fiber.Ctx) error {
		return OK(c, fiber.Map{"x": 1})
	})
	req := httptest.NewRequest("GET", "/t", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request err: %v", err)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != "OK" || body["message"] != "success" {
		t.Fatalf("unexpected envelope: %v", body)
	}
	data := body["data"].(map[string]any)
	if int(data["x"].(float64)) != 1 {
		t.Fatalf("unexpected data: %v", data)
	}
}
