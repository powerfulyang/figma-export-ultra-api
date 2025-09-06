// Package httpx provides HTTP handling utilities and middleware
package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Minimal E2E covering the http error envelope and health route.
func TestE2E_Health(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler()})
	RegisterCommonMiddlewares(app)
	// Register routes with nil client/providers; only /health is invoked.
	Register(app, nil)

	// Health check
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}
	var body struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "OK" || body.Data["status"] != "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestE2E_NotFoundEnvelope(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler()})
	// no routes registered
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != "E_NOT_FOUND" {
		t.Fatalf("unexpected body: %v", body)
	}
}
