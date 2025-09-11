package kit

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// CursorPayload represents cursor data for pagination
type CursorPayload struct {
	ID string    `json:"id"`
	TS time.Time `json:"ts"`
}

// PagingParams contains pagination parameters from HTTP request
type PagingParams struct {
	Limit  int
	Offset int
	// Cursor components (if any)
	CursorID *uuid.UUID
	CursorTS *time.Time
	// Snapshot time (if any)
	Snapshot *time.Time
	// Sort key string
	Sort string
	// Mode: offset | cursor | snapshot
	Mode string
	// Whether to compute total count (offset mode only)
	WithTotal bool
}

func ParsePaging(c *fiber.Ctx) (PagingParams, error) {
	p := PagingParams{Limit: lo.Clamp(c.QueryInt("limit", 20), 1, 100)}
	p.Offset = c.QueryInt("offset", 0)
	rawCursor := c.Query("cursor", "")
	p.Sort = c.Query("sort", "")
	p.WithTotal = c.Query("with_total", "false") == "true"

	// snapshot fixed window
	snapshotStr := c.Query("snapshot", "")
	if snapshotStr == "" && c.Query("fixed", "") == "true" {
		p.Snapshot = lo.ToPtr(time.Now().UTC())
	} else if snapshotStr != "" {
		ts, err := time.Parse(time.RFC3339Nano, snapshotStr)
		if err != nil {
			return p, BadRequest("invalid snapshot", snapshotStr)
		}
		p.Snapshot = &ts
	}

	// decode cursor: support UUID string or base64 JSON {id,ts}
	if rawCursor != "" {
		if id, err := uuid.Parse(rawCursor); err == nil {
			p.Mode = "cursor"
			p.CursorID = lo.ToPtr(id)
			if tsStr := c.Query("cursor_ts", ""); tsStr != "" {
				if ts, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
					p.CursorTS = lo.ToPtr(ts)
				}
			}
		} else {
			// try base64
			payload, err := DecodeCursor(rawCursor)
			if err != nil {
				return p, BadRequest("invalid cursor", rawCursor)
			}
			p.Mode = "cursor"
			if id, err := uuid.Parse(payload.ID); err == nil {
				p.CursorID = lo.ToPtr(id)
				t := payload.TS.UTC()
				p.CursorTS = lo.ToPtr(t)
			} else {
				return p, BadRequest("invalid cursor id format", payload.ID)
			}
		}
	}

	if p.Snapshot != nil {
		p.Mode = "snapshot"
	} else if p.Mode == "" {
		p.Mode = "offset"
	}

	return p, nil
}

func EncodeCursor(id string, ts time.Time) string {
	payload := CursorPayload{ID: id, TS: ts.UTC()}
	b, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (CursorPayload, error) {
	var out CursorPayload
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &out); err == nil {
			out.TS = out.TS.UTC()
			return out, nil
		}
	}
	return out, fiber.NewError(fiber.StatusBadRequest, "invalid cursor")
}
