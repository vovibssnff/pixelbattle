package repository

import (
	"context"
	"fmt"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type UserRepository struct {
	userDb   *redis.Client
	bannedDb *redis.Client
}

func NewUserRepository(userDb, bannedDb *redis.Client) *UserRepository {
	return &UserRepository{userDb: userDb, bannedDb: bannedDb}
}

func (r UserRepository) RegisterUser(ctx context.Context, usr domain.User) error {
	key := fmt.Sprintf("usr:%d", usr.ID)
	serializedUser, err := utils.SerializeUser(&usr)
	if err != nil {
		return err
	}
	return r.userDb.Set(ctx, key, serializedUser, 0).Err()
}

func (r UserRepository) UserExists(ctx context.Context, usrID int) bool {
	key := fmt.Sprintf("usr:%d", usrID)
	res, err := r.userDb.Exists(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	return res == 1
}

func (r UserRepository) GetUsr(ctx context.Context, usrID int) domain.User {
	key := fmt.Sprintf("usr:%d", usrID)
	jsonUsr, err := r.userDb.Get(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	var usr domain.User
	utils.DeserializeUser([]byte(jsonUsr), &usr)
	return usr
}

func (r UserRepository) DelUsr(ctx context.Context, usrID int) {
	key := fmt.Sprintf("usr:%d", usrID)
	_, err := r.userDb.Del(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
}

func (r UserRepository) CheckBanned(ctx context.Context, userid int) bool {
	res, _ := r.bannedDb.Exists(ctx, strconv.Itoa(userid)).Result()
	return res != 0
}

func (r UserRepository) GetAllUserKeys(ctx context.Context) ([]string, error) {
	pattern := "usr:*"
	var cursor uint64
	var keys []string

	for {
		scanKeys, nextCursor, err := r.userDb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan Redis keys: %w", err)
		}
		keys = append(keys, scanKeys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}
