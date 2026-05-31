package repository

import (
	"context"
	"pb_backend/internal/metrics"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const cooldownKey = "{timer}:cooldown_seconds"

type TimerRepository struct {
	rdb redis.Cmdable
}

func NewTimerRepo(rdb redis.Cmdable) *TimerRepository {
	return &TimerRepository{rdb: rdb}
}

func (r *TimerRepository) SetTimer(ctx context.Context, userid string, delay int) error {
	start := time.Now()
	if delay <= 0 {
		err := r.rdb.Del(ctx, userid).Err()
		metrics.ObserveDatabaseOperation("set_timer", "redis", time.Since(start), err)
		return err
	}
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

func (r *TimerRepository) SetCooldownSeconds(ctx context.Context, sec int) error {
	start := time.Now()
	err := r.rdb.Set(ctx, cooldownKey, strconv.Itoa(sec), 0).Err()
	metrics.ObserveDatabaseOperation("set_cooldown", "redis", time.Since(start), err)
	return err
}

func (r *TimerRepository) GetCooldownSeconds(ctx context.Context) (int, bool, error) {
	start := time.Now()
	raw, err := r.rdb.Get(ctx, cooldownKey).Result()
	if err == redis.Nil {
		metrics.ObserveDatabaseOperation("get_cooldown", "redis", time.Since(start), nil)
		return 0, false, nil
	}
	if err != nil {
		metrics.ObserveDatabaseOperation("get_cooldown", "redis", time.Since(start), err)
		return 0, false, err
	}
	sec, err := strconv.Atoi(raw)
	metrics.ObserveDatabaseOperation("get_cooldown", "redis", time.Since(start), err)
	if err != nil {
		return 0, false, err
	}
	return sec, true, nil
}
