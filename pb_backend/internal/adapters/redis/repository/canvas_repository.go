package repository

import (
	"context"
	"pb_backend/internal/metrics"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type CanvasRepository struct {
	rdb         redis.Cmdable
	hashTagKeys bool // true: pixel:{y:x} / opstream:{y:x}; false: pixel:y:x (Phase 1)
}

// NewCanvasRepository stores canvas pixels in Redis lists. hashTagKeys must be true for
// Redis Cluster so each cell's keys share one hash slot (plan §11.4).
func NewCanvasRepository(rdb redis.Cmdable, hashTagKeys bool) *CanvasRepository {
	return &CanvasRepository{rdb: rdb, hashTagKeys: hashTagKeys}
}

func (r *CanvasRepository) pixelKey(x, y uint) string {
	return PixelKey(r.hashTagKeys, x, y)
}

func (r *CanvasRepository) WritePixel(ctx context.Context, x, y uint, pixelData []byte) error {
	start := time.Now()
	err := r.rdb.RPush(ctx, r.pixelKey(x, y), pixelData).Err()
	metrics.ObserveDatabaseOperation("write_pixel", "redis", time.Since(start), err)
	return err
}

func (r *CanvasRepository) CheckInitialized(ctx context.Context) bool {
	start := time.Now()
	keys, err := collectKeysMatching(ctx, r.rdb, pixelKeyGlob)
	metrics.ObserveDatabaseOperation("check_initialized", "redis", time.Since(start), err)
	if err != nil {
		logrus.Error(err)
		return false
	}
	return len(keys) > 0
}

func (r *CanvasRepository) GetCanvas(ctx context.Context) (map[string][]string, error) {
	start := time.Now()
	keys, err := collectKeysMatching(ctx, r.rdb, pixelKeyGlob)
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas", "redis", time.Since(start), err)
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	keyCmdMap := make(map[string]*redis.StringSliceCmd, len(keys))
	for _, key := range keys {
		cmd := pipe.LRange(ctx, key, -1, -1)
		keyCmdMap[key] = cmd
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas", "redis", time.Since(start), err)
		return nil, err
	}
	result := make(map[string][]string)
	for key, cmd := range keyCmdMap {
		values, err := cmd.Result()
		if err != nil {
			metrics.ObserveDatabaseOperation("get_canvas", "redis", time.Since(start), err)
			return nil, err
		}
		result[key] = values
	}
	metrics.ObserveDatabaseOperation("get_canvas", "redis", time.Since(start), nil)
	return result, nil
}

func (r *CanvasRepository) GetCanvasHistory(ctx context.Context) (map[string][]string, error) {
	start := time.Now()
	keys, err := collectKeysMatching(ctx, r.rdb, pixelKeyGlob)
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas_history", "redis", time.Since(start), err)
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	keyCmdMap := make(map[string]*redis.StringSliceCmd, len(keys))
	for _, key := range keys {
		cmd := pipe.LRange(ctx, key, 0, -1)
		keyCmdMap[key] = cmd
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas_history", "redis", time.Since(start), err)
		return nil, err
	}
	result := make(map[string][]string)
	for key, cmd := range keyCmdMap {
		values, err := cmd.Result()
		if err != nil {
			metrics.ObserveDatabaseOperation("get_canvas_history", "redis", time.Since(start), err)
			return nil, err
		}
		result[key] = values
	}
	metrics.ObserveDatabaseOperation("get_canvas_history", "redis", time.Since(start), nil)
	return result, nil
}

func (r *CanvasRepository) LoadHeatMap(ctx context.Context) (map[string]int64, error) {
	start := time.Now()
	keys, err := collectKeysMatching(ctx, r.rdb, pixelKeyGlob)
	if err != nil {
		metrics.ObserveDatabaseOperation("load_heatmap", "redis", time.Since(start), err)
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	lenCmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		lenCmds[i] = pipe.LLen(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		metrics.ObserveDatabaseOperation("load_heatmap", "redis", time.Since(start), err)
		return nil, err
	}
	result := make(map[string]int64)
	for i, key := range keys {
		length, err := lenCmds[i].Result()
		if err != nil {
			metrics.ObserveDatabaseOperation("load_heatmap", "redis", time.Since(start), err)
			return nil, err
		}
		result[key] = length
	}
	metrics.ObserveDatabaseOperation("load_heatmap", "redis", time.Since(start), nil)
	return result, nil
}
