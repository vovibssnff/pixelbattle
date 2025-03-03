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

// RegisterUser registers a new user in Redis.
func (r UserRepository) RegisterUser(ctx context.Context, usr domain.User) error {
	key := fmt.Sprintf("usr:%d", usr.ID)
	serializedUser, err := utils.SerializeUser(&usr)
	if err != nil {
		return err
	}
	return r.userDb.Set(ctx, key, serializedUser, 0).Err()
}

// UserExists checks if a user exists in Redis.
func (r UserRepository) UserExists(ctx context.Context, usrID int) bool {
	key := fmt.Sprintf("usr:%d", usrID)
	res, err := r.userDb.Exists(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	return res == 1
}

// GetUsr retrieves a user from Redis.
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

// DelUsr deletes a user from Redis.
func (r UserRepository) DelUsr(ctx context.Context, usrID int) {
	key := fmt.Sprintf("usr:%d", usrID)
	_, err := r.userDb.Del(ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
}

// CheckBanned checks if a user is banned.
func (r UserRepository) CheckBanned(ctx context.Context, userid int) bool {
	res, _ := r.bannedDb.Exists(ctx, strconv.Itoa(userid)).Result()
	return res != 0
}
