package httpx

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CursorPayload struct {
	ID int       `json:"id"`
	TS time.Time `json:"ts"`
}

type PagingParams struct {
	Limit  int
	Offset int
	// Cursor components (if any)
	CursorID *int
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

func parsePaging(c *fiber.Ctx) (PagingParams, error) {
	p := PagingParams{Limit: clamp(c.QueryInt("limit", 20), 1, 100)}
	p.Offset = c.QueryInt("offset", 0)
	rawCursor := c.Query("cursor", "")
	p.Sort = c.Query("sort", "")
	p.WithTotal = c.Query("with_total", "false") == "true"

	// snapshot fixed window
	snapshotStr := c.Query("snapshot", "")
	if snapshotStr == "" && c.Query("fixed", "") == "true" {
		now := time.Now().UTC()
		p.Snapshot = &now
	} else if snapshotStr != "" {
		ts, err := time.Parse(time.RFC3339Nano, snapshotStr)
		if err != nil {
			return p, BadRequest("invalid snapshot", snapshotStr)
		}
		p.Snapshot = &ts
	}

	// decode cursor: support plain int (legacy) or base64 JSON {id,ts}
	if rawCursor != "" {
		if id, err := strconv.Atoi(rawCursor); err == nil {
			p.Mode = "cursor"
			p.CursorID = &id
			if tsStr := c.Query("cursor_ts", ""); tsStr != "" {
				if ts, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
					p.CursorTS = &ts
				}
			}
		} else {
			// try base64
			payload, err := decodeCursor(rawCursor)
			if err != nil {
				return p, BadRequest("invalid cursor", rawCursor)
			}
			p.Mode = "cursor"
			p.CursorID = &payload.ID
			t := payload.TS.UTC()
			p.CursorTS = &t
		}
	}

	if p.Snapshot != nil {
		p.Mode = "snapshot"
	} else if p.Mode == "" {
		p.Mode = "offset"
	}

	return p, nil
}

func encodeCursor(id int, ts time.Time) string {
	// Compact binary encoding: [version=1][uvarint id][uvarint unixNano]
	buf := make([]byte, 1+binary.MaxVarintLen64*2)
	buf[0] = 1
	n := 1
	n += binary.PutUvarint(buf[n:], uint64(id))
	n += binary.PutUvarint(buf[n:], uint64(ts.UTC().UnixNano()))
	return base64.RawURLEncoding.EncodeToString(buf[:n])
}

func decodeCursor(s string) (CursorPayload, error) {
	var out CursorPayload
	// Try base64 binary format
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		if b[0] == 1 {
			var off int = 1
			id, n := binary.Uvarint(b[off:])
			if n <= 0 {
				return out, fiber.NewError(fiber.StatusBadRequest, "invalid cursor id")
			}
			off += n
			ts, n2 := binary.Uvarint(b[off:])
			if n2 <= 0 {
				return out, fiber.NewError(fiber.StatusBadRequest, "invalid cursor ts")
			}
			out.ID = int(id)
			out.TS = time.Unix(0, int64(ts)).UTC()
			return out, nil
		}
		// else try JSON payload
		if err := json.Unmarshal(b, &out); err == nil {
			out.TS = out.TS.UTC()
			return out, nil
		}
	}
	// Fallback: numeric id only already handled in parsePaging
	return out, fiber.NewError(fiber.StatusBadRequest, "invalid cursor")
}
