package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRedisPixelKey parses canvas list keys written by the Redis adapter:
//   - Legacy: pixel:y:x
//   - Cluster / hashtag: pixel:{y:x}
//
// Returns (x, y) matching domain.Pixel coordinates.
func ParseRedisPixelKey(key string) (x, y uint, err error) {
	const prefix = "pixel:"
	if !strings.HasPrefix(key, prefix) {
		return 0, 0, fmt.Errorf("redis pixel key: want prefix %q, got %q", prefix, key)
	}
	rest := key[len(prefix):]
	if strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") {
		inner := rest[1 : len(rest)-1]
		parts := strings.SplitN(inner, ":", 2)
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("redis pixel key: invalid hashtag body in %q", key)
		}
		yu, err1 := strconv.ParseUint(parts[0], 10, 32)
		xu, err2 := strconv.ParseUint(parts[1], 10, 32)
		if err1 != nil {
			return 0, 0, fmt.Errorf("redis pixel key y: %w", err1)
		}
		if err2 != nil {
			return 0, 0, fmt.Errorf("redis pixel key x: %w", err2)
		}
		return uint(xu), uint(yu), nil
	}
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("redis pixel key: invalid legacy %q", key)
	}
	yu, err1 := strconv.ParseUint(parts[0], 10, 32)
	xu, err2 := strconv.ParseUint(parts[1], 10, 32)
	if err1 != nil {
		return 0, 0, fmt.Errorf("redis pixel key y: %w", err1)
	}
	if err2 != nil {
		return 0, 0, fmt.Errorf("redis pixel key x: %w", err2)
	}
	return uint(xu), uint(yu), nil
}
