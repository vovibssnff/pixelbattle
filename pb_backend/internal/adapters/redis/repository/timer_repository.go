package repository

import (
	"context"
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
	return r.rdb.Set(ctx, userid, "", time.Duration(delay)*time.Second).Err()
}

func (r *TimerRepository) CheckTime(ctx context.Context, userid string) (int64, error) {
	return r.rdb.Exists(ctx, userid).Result()
}
