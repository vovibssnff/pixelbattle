package repository

import (
	"context"
	"fmt"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"

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

func userKey(usrID string) string {
	return fmt.Sprintf("usr:%s", usrID)
}

func (r UserRepository) RegisterUser(ctx context.Context, usr domain.User) error {
	key := userKey(usr.ID)
	serializedUser, err := utils.SerializeUser(&usr)
	if err != nil {
		return err
	}
	return r.userDb.Set(ctx, key, serializedUser, 0).Err()
}

func (r UserRepository) UserExists(ctx context.Context, usrID string) bool {
	key := userKey(usrID)
	res, err := r.userDb.Exists(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	return res == 1
}

func (r UserRepository) GetUsr(ctx context.Context, usrID string) domain.User {
	key := userKey(usrID)
	jsonUsr, err := r.userDb.Get(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	var usr domain.User
	utils.DeserializeUser([]byte(jsonUsr), &usr)
	return usr
}

func (r UserRepository) DelUsr(ctx context.Context, usrID string) {
	key := userKey(usrID)
	_, err := r.userDb.Del(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
}

func (r UserRepository) CheckBanned(ctx context.Context, userid string) bool {
	res, _ := r.bannedDb.Exists(ctx, userid).Result()
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
