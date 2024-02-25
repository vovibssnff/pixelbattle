package service

import (
	"context"
	"fmt"
	"pb_backend/models"
	"time"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func WritePixel(rdb *redis.Client, p *models.Pixel) error {
	key := fmt.Sprintf("pixel:%d:%d", p.Y, p.X)

	redisPixel := models.NewRedisPixel(p.Userid, p.Color, time.Now().Unix())

	serializedRedisPixel, err := redisPixel.ToRedisFormat()
	if err != nil {
		return err
	}

	return rdb.RPush(context.Background(), key, serializedRedisPixel).Err()
}

func CheckTime(rdb *redis.Client, userid string) (int64, error) {
	return rdb.Exists(context.Background(), userid).Result()
}

func SetTimer(rdb *redis.Client, userid string, n int) error {
	return rdb.Set(context.Background(), userid, "", time.Duration(n)*time.Second).Err()
}
