package mw

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"

	"fiber-ent-apollo-pg/internal/redisx"
)

// RateLimitDefault builds a rate limit middleware with a composite key of ip+device+anon+subject.
func RateLimitDefault(rdb *redisx.Client, windowSec int, limit int) fiber.Handler {
	keyFn := func(c *fiber.Ctx) string {
		ip := c.IP()
		dev := c.Get("X-Device-Id")
		anon := c.Get("X-Anon-Id")
		sub := ""
		if ac, _ := c.Locals("auth").(*AuthContext); ac != nil {
			sub = ac.Subject
			if dev == "" {
				dev = ac.DeviceID
			}
		}
		return fmt.Sprintf("ip:%s|dev:%s|anon:%s|sub:%s", ip, dev, anon, sub)
	}
	if rdb == nil {
		return limiter.New(limiter.Config{
			Max:          limit,
			Expiration:   time.Duration(windowSec) * time.Second,
			KeyGenerator: func(c *fiber.Ctx) string { return keyFn(c) },
			LimitReached: func(_ *fiber.Ctx) error {
				return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
			},
		})
	}
	return func(c *fiber.Ctx) error {
		key := "rl:" + keyFn(c)
		ctx, cancel := context.WithTimeout(c.Context(), 200*time.Millisecond)
		defer cancel()
		script := redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return current`)
		ttlMs := int64(windowSec) * 1000
		res, err := script.Run(ctx, rdb, []string{key}, ttlMs).Result()
		if err != nil {
			return c.Next()
		}
		n, _ := res.(int64)
		if n > int64(limit) {
			c.Set("Retry-After", fmt.Sprint(windowSec))
			c.Set("X-RateLimit-Limit", fmt.Sprint(limit))
			c.Set("X-RateLimit-Remaining", fmt.Sprint(lo.Max([]int64{0, int64(limit) - n})))
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		c.Set("X-RateLimit-Limit", fmt.Sprint(limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprint(int64(limit)-n))
		return c.Next()
	}
}
