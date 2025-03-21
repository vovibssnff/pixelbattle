package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type CanvasRepository struct {
	rdb *redis.Client
}

func NewCanvasRepository(rdb *redis.Client) *CanvasRepository {
	return &CanvasRepository{rdb: rdb}
}

func (r *CanvasRepository) WritePixel(ctx context.Context, x, y uint, pixelData []byte) error {
	return r.rdb.RPush(ctx, fmt.Sprintf("pixel:%d:%d", y, x), pixelData).Err()
}

func (r *CanvasRepository) CheckInitialized(ctx context.Context) bool {
	keys, err := r.rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
		logrus.Error(err)
		return false
	}
	return len(keys) > 0
}

func (r *CanvasRepository) GetCanvas(ctx context.Context) (map[string][]string, error) {
	keys, err := r.rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
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
		return nil, err
	}
	result := make(map[string][]string)
	for key, cmd := range keyCmdMap {
		values, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		result[key] = values
	}
	return result, nil
}

func (r *CanvasRepository) GetCanvasHistory(ctx context.Context) (map[string][]string, error) {
	keys, err := r.rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
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
		return nil, err
	}
	result := make(map[string][]string)
	for key, cmd := range keyCmdMap {
		values, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		result[key] = values
	}
	return result, nil
}

func (r *CanvasRepository) LoadHeatMap(ctx context.Context) (map[string]int64, error) {
	keys, err := r.rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	lenCmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		lenCmds[i] = pipe.LLen(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for i, key := range keys {
		length, err := lenCmds[i].Result()
		if err != nil {
			return nil, err
		}
		result[key] = length
	}
	return result, nil
}
