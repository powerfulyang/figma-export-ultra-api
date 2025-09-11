package kit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

// PageMeta contains pagination metadata for API responses
type PageMeta struct {
	Limit         int     `json:"limit"`
	Offset        int     `json:"offset,omitempty"`
	Count         int     `json:"count"`
	NextOffset    *int    `json:"next_offset,omitempty"`
	Cursor        *string `json:"cursor,omitempty"`
	NextCursor    *string `json:"next_cursor,omitempty"`
	CursorEnc     string  `json:"cursor_enc,omitempty"`
	NextCursorEnc string  `json:"next_cursor_enc,omitempty"`
	HasMore       bool    `json:"has_more,omitempty"`
	Mode          string  `json:"mode,omitempty"` // offset | cursor | snapshot
	Snapshot      string  `json:"snapshot,omitempty"`
	CursorTS      string  `json:"cursor_ts,omitempty"`
	NextCursorTS  string  `json:"next_cursor_ts,omitempty"`
	Total         *int    `json:"total,omitempty"`
}

// RequestID extracts request id from headers
func RequestID(c *fiber.Ctx) string {
	rid := c.GetRespHeader("X-Request-ID")
	return lo.Ternary(rid != "", rid, c.Get("X-Request-ID"))
}

func envelope(status int, code, msg string, data any, meta any, c *fiber.Ctx) error {
	body := fiber.Map{
		"code":       code,
		"message":    msg,
		"data":       data,
		"request_id": RequestID(c),
	}
	if meta != nil {
		body["meta"] = meta
	}
	return c.Status(status).JSON(body)
}

// OK sends a 200 OK response with data
func OK(c *fiber.Ctx, data any) error {
	return envelope(fiber.StatusOK, "OK", "success", data, nil, c)
}

// Created sends a 201 Created response with data
func Created(c *fiber.Ctx, data any) error {
	return envelope(fiber.StatusCreated, "OK", "success", data, nil, c)
}

// List sends a 200 OK response with paginated data and metadata
func List(c *fiber.Ctx, items any, meta PageMeta) error {
	return envelope(fiber.StatusOK, "OK", "success", items, meta, c)
}
