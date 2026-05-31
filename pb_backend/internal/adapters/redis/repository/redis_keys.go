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

// PixelMetaKey is a HASH holding latest_stream_id and HLC fields for LWW (same slot as pixel / opstream).
func PixelMetaKey(x, y uint) string {
	return fmt.Sprintf("pixmeta:{%d:%d}", y, x)
}

// ProbeKey is outside the pixel:* namespace so availability checks never make an
// empty canvas look initialized.
func ProbeKey(hashTag bool) string {
	if hashTag {
		return "{probe}:availability"
	}
	return "probe:availability"
}

const pixelKeyGlob = "pixel:*"

// CollectPixelKeysByDimension returns every canvas list key for width×height (row-major y then x).
// Used instead of SCAN when logical size is known so snapshots read exactly w×h cells.
func CollectPixelKeysByDimension(width, height uint, hashTagKeys bool) []string {
	if width == 0 || height == 0 {
		return nil
	}
	n := int(width) * int(height)
	keys := make([]string, 0, n)
	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			keys = append(keys, PixelKey(hashTagKeys, x, y))
		}
	}
	return keys
}

// CanvasSizeKey stores logical width/height in a small Redis HASH (ADR-003).
// Hash-tagged for cluster so meta lives in one slot.
func CanvasSizeKey(hashTagKeys bool) string {
	if hashTagKeys {
		return "{canvas}:size"
	}
	return "canvas:size"
}

// collectPixelKeys returns all canvas pixel keys (pixel:* / hashtag form). For *redis.ClusterClient it scans
// each master (KEYS is not cluster-safe). For standalone *redis.Client it uses KEYS.
func collectPixelKeys(ctx context.Context, rdb redis.Cmdable) ([]string, error) {
	pattern := pixelKeyGlob
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
