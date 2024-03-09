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

func InitializeCanvas(rdb *redis.Client, height uint, width uint) error {
	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			key := fmt.Sprintf("pixel:%d:%d", i, j)
			redisPixel := models.NewRedisPixel("Vovi", []uint{255, 255, 255}, time.Now().Unix())
			serializedRedisPixel, err := redisPixel.ToRedisFormat()
			if err != nil {
				return err
			}
			err = rdb.RPush(context.Background(), key, serializedRedisPixel).Err()
			if err != nil {
				return err
			}
		}
	}
	// rdb.Conn().Select(context.Background(), 3)
	// rdb.Set(context.Background(), "initialized", "true", 0).Err()
	// rdb.Conn().Select(context.Background(), 2)
	return nil
}

// func CheckInitialized(rdb *redis.Client) (bool) {
// 	rdb.Conn().Select(context.Background(), 3)
// 	initialized, _ := rdb.Exists(context.Background(), "initialized").Result()
// 	rdb.Conn().Select(context.Background(), 2)
// 	return initialized == 1
// }

func GetCanvas(rdb *redis.Client, img *Image) error {
	keys, err := rdb.Keys(context.Background(), "*").Result()
	if err != nil {
		return err
	}
	var y, x uint
	for _, key := range keys {
		_, err := fmt.Sscanf(key, "pixel:%d:%d", &y, &x)
		if err != nil {
			return err
		}
		jsonString, err := rdb.LRange(context.Background(), key, -1, -1).Result()
		if err != nil {
			return err
		}
		var deserialized models.RedisPixel
		if err := deserialized.FromRedisFormat([]byte(jsonString[0])); err != nil {
			return err
		}
		pixel := models.NewPixel(x, y, deserialized.Color)
		img.Data = append(img.Data, *pixel)
	}
	return nil
}
