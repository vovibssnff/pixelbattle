package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// PixelKey returns the Redis list key for canvas cell (x,y).
// Legacy: pixel:y:x (Phase 1 monolith).
// HashTag: pixel:{y:x} so pixel and opstream for the same cell share a cluster slot (plan §11.4).
func PixelKey(hashTag bool, x, y uint) string {
	if hashTag {
		return fmt.Sprintf("pixel:{%d:%d}", y, x)
	}
	return fmt.Sprintf("pixel:%d:%d", y, x)
}

// OpStreamKey is the Redis Streams key colocated with PixelKey via the same {y:x} hashtag.
func OpStreamKey(x, y uint) string {
	return fmt.Sprintf("opstream:{%d:%d}", y, x)
}

const pixelKeyGlob = "pixel:*"

// collectKeysMatching returns all keys matching pattern. For *redis.ClusterClient it scans
// each master (KEYS is not cluster-safe). For standalone *redis.Client it uses KEYS.
func collectKeysMatching(ctx context.Context, rdb redis.Cmdable, pattern string) ([]string, error) {
	if cl, ok := rdb.(*redis.ClusterClient); ok {
		var out []string
		err := cl.ForEachMaster(ctx, func(ctx context.Context, c *redis.Client) error {
			var cursor uint64
			for {
				keys, next, err := c.Scan(ctx, cursor, pattern, 256).Result()
				if err != nil {
					return err
				}
				out = append(out, keys...)
				if next == 0 {
					break
				}
				cursor = next
			}
			return nil
		})
		return out, err
	}
	return rdb.Keys(ctx, pattern).Result()
}
