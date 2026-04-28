package repository

import (
	"context"
	"pb_backend/internal/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

type TimerRepository struct {
	rdb *redis.Client
}

func NewTimerRepo(rdb *redis.Client) *TimerRepository {
	return &TimerRepository{rdb: rdb}
}

func (r *TimerRepository) SetTimer(ctx context.Context, userid string, delay int) error {
	start := time.Now()
	err := r.rdb.Set(ctx, userid, "", time.Duration(delay)*time.Second).Err()
	metrics.ObserveDatabaseOperation("set_timer", "redis", time.Since(start), err)
	return err
}

func (r *TimerRepository) CheckTime(ctx context.Context, userid string) (int64, error) {
	start := time.Now()
	res, err := r.rdb.Exists(ctx, userid).Result()
	metrics.ObserveDatabaseOperation("check_time", "redis", time.Since(start), err)
	return res, err
}
