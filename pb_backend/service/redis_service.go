package service

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"pb_backend/models"
	"strconv"
	"time"
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
	redisPixel := models.NewRedisPixel(p.Userid, p.Faculty, p.Color, time.Now().Unix())
	serializedRedisPixel, err := redisPixel.ToRedisFormat()
	if err != nil {
		return err
	}
	return rdb.RPush(context.Background(), key, serializedRedisPixel).Err()
}

func CheckBanned(rdb *redis.Client, userid int) bool {
	res, _ := rdb.Exists(context.Background(), strconv.Itoa(userid)).Result()
	if res == 0 {
		return false
	}
	return true
}

func CheckTime(rdb *redis.Client, userid int) (int64, error) {
	return rdb.Exists(context.Background(), strconv.Itoa(userid)).Result()
}

func SetTimer(rdb *redis.Client, userid int, n int) error {
	return rdb.Set(context.Background(), strconv.Itoa(userid), "", time.Duration(n)*time.Second).Err()
}

func InitializeCanvas(rdb *redis.Client, height uint, width uint) error {
	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			key := fmt.Sprintf("pixel:%d:%d", i, j)
			redisPixel := models.NewRedisPixel(1, "", []uint{255, 255, 255}, time.Now().Unix())
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
	return nil
}

func CheckInitialized(rdb *redis.Client) bool {
	dbSize, err := rdb.DBSize(context.Background()).Result()
	if err != nil {
		logrus.Error(err)
	}
	return dbSize != 0
}

func loadHeatMap(rdb *redis.Client) ([]models.HeatMapUnit, error) {
	res := make([]models.HeatMapUnit, 0)
	var y, x int
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
		return nil, err
	}
	pipe := rdb.Pipeline()
	lenCmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		lenCmds[i] = pipe.LLen(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		length, err := lenCmds[i].Result()
		if err != nil {
			return nil, err
		}
		_, err = fmt.Sscanf(key, "pixel:%d:%d", &y, &x)
		if err != nil {
			return nil, err
		}
		res = append(res, models.HeatMapUnit{X: uint(x), Y: uint(y), Len: uint(length)})
	}
	return res, nil
}

func GetCanvas(rdb *redis.Client, img *Image) error {
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "pixel:*").Result()
	if err != nil {
		return err
	}
	pipe := rdb.Pipeline()
	keyCmdMap := make(map[string]*redis.StringSliceCmd, len(keys))
	for _, key := range keys {
		cmd := pipe.LRange(ctx, key, -1, -1)
		keyCmdMap[key] = cmd
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	for key, cmd := range keyCmdMap {
		jsonString, err := cmd.Result()
		if err != nil {
			return err
		}
		var deserialized models.RedisPixel
		if err := deserialized.FromRedisFormat([]byte(jsonString[0])); err != nil {
			return err
		}
		var y, x uint
		_, err = fmt.Sscanf(key, "pixel:%d:%d", &y, &x)
		if err != nil {
			return err
		}
		pixel := models.NewPixel(x, y, deserialized.Color)
		img.Data = append(img.Data, *pixel)
	}
	return nil
}

func RegisterUser(rdb *redis.Client, usr models.User) error {
	key := fmt.Sprintf("usr:%d", usr.ID)
	serializedUser, err := usr.SerializeUser()
	if err != nil {
		return err
	}
	return rdb.Set(context.Background(), key, serializedUser, 0).Err()
}

// true if exists
func UserExists(rdb *redis.Client, usrID int) bool {
	key := fmt.Sprintf("usr:%d", usrID)
	res, err := rdb.Exists(context.Background(), key).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res == 1 {
		return true
	}
	return false
}

func GetUsr(rdb *redis.Client, usrID int) models.User {
	key := fmt.Sprintf("usr:%d", usrID)
	jsonUsr, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		logrus.Error(err)
	}
	var usr models.User
	usr.DeserializeUser([]byte(jsonUsr))
	return usr
}

func DelUsr(rdb *redis.Client, usrID int) {
	key := fmt.Sprintf("usr:%d", usrID)
	_, err := rdb.Del(context.Background(), key).Result()
	if err != nil {
		logrus.Error(err)
	}
}
