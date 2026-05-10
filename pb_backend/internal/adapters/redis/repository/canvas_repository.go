package repository

import (
	"context"
	"pb_backend/internal/metrics"
	"strconv"
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

func (r *CanvasRepository) sizeKey() string {
	return CanvasSizeKey(r.hashTagKeys)
}

// GetCanvasDimensions reads persisted logical size (0,0 if unset).
func (r *CanvasRepository) GetCanvasDimensions(ctx context.Context) (uint, uint, error) {
	start := time.Now()
	ws, err := r.rdb.HGet(ctx, r.sizeKey(), "width").Result()
	if err == redis.Nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), nil)
		return 0, 0, nil
	}
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), err)
		return 0, 0, err
	}
	hs, err := r.rdb.HGet(ctx, r.sizeKey(), "height").Result()
	if err == redis.Nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), nil)
		return 0, 0, nil
	}
	if err != nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), err)
		return 0, 0, err
	}
	w64, err1 := strconv.ParseUint(ws, 10, 32)
	h64, err2 := strconv.ParseUint(hs, 10, 32)
	if err1 != nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), err1)
		return 0, 0, err1
	}
	if err2 != nil {
		metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), err2)
		return 0, 0, err2
	}
	metrics.ObserveDatabaseOperation("get_canvas_size", "redis", time.Since(start), nil)
	return uint(w64), uint(h64), nil
}

// SetCanvasDimensions persists logical canvas size (used after init / resize).
func (r *CanvasRepository) SetCanvasDimensions(ctx context.Context, width, height uint) error {
	start := time.Now()
	err := r.rdb.HSet(ctx, r.sizeKey(), "width", strconv.FormatUint(uint64(width), 10), "height", strconv.FormatUint(uint64(height), 10)).Err()
	metrics.ObserveDatabaseOperation("set_canvas_size", "redis", time.Since(start), err)
	return err
}

func (r *CanvasRepository) WritePixel(ctx context.Context, x, y uint, pixelData []byte) error {
	if r.hashTagKeys {
		return r.writePixelWithOpLog(ctx, x, y, pixelData)
	}
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
