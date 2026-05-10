package repository

import (
	"context"
	"errors"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/metrics"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	users       *mongo.Collection
	banned      *mongo.Collection
	adminGrants *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		users:       db.Collection("users"),
		banned:      db.Collection("banned_users"),
		adminGrants: db.Collection("admin_grants"),
	}
}

func (r *UserRepository) RegisterUser(ctx context.Context, usr domain.User) error {
	start := time.Now()
	mongoUser := domain.User{
		ID:           usr.ID,
		FirstName:    usr.FirstName,
		LastName:     usr.LastName,
		PasswordHash: usr.PasswordHash,
		AccessToken:  usr.AccessToken,
		Faculty:      usr.Faculty,
		Stats:        domain.UserStats{TotalPixelsPlaced: 0, ActivePixels: 0},
	}
	_, err := r.users.InsertOne(ctx, mongoUser)
	metrics.ObserveDatabaseOperation("register_user", "mongo", time.Since(start), err)
	return err
}

func (r *UserRepository) UpdateUser(ctx context.Context, usr domain.User) error {
	start := time.Now()
	set := bson.M{
		"first_name":   usr.FirstName,
		"last_name":    usr.LastName,
		"access_token": usr.AccessToken,
		"faculty":      usr.Faculty,
	}
	if usr.PasswordHash != "" {
		set["password_hash"] = usr.PasswordHash
	}
	update := bson.M{"$set": set}
	_, err := r.users.UpdateOne(ctx, bson.M{"_id": usr.ID}, update)
	metrics.ObserveDatabaseOperation("update_user", "mongo", time.Since(start), err)
	return err
}

func (r *UserRepository) UserExists(ctx context.Context, usrID string) bool {
	start := time.Now()
	count, err := r.users.CountDocuments(ctx, bson.M{"_id": usrID})
	metrics.ObserveDatabaseOperation("user_exists", "mongo", time.Since(start), err)
	if err != nil {
		logrus.Error(err)
		return false
	}
	return count > 0
}

func (r *UserRepository) GetUsr(ctx context.Context, usrID string) domain.User {
	u, err := r.GetUserWithHash(ctx, usrID)
	if err != nil {
		logrus.Error(err)
		return domain.User{}
	}
	u.PasswordHash = ""
	return u
}

func (r *UserRepository) GetUserWithHash(ctx context.Context, usrID string) (domain.User, error) {
	start := time.Now()
	var mongoUser domain.User
	err := r.users.FindOne(ctx, bson.M{"_id": usrID}).Decode(&mongoUser)
	metrics.ObserveDatabaseOperation("get_user", "mongo", time.Since(start), err)
	if err != nil {
		return domain.User{}, err
	}
	return mongoUser, nil
}

func (r *UserRepository) DelUsr(ctx context.Context, usrID string) {
	start := time.Now()
	_, err := r.users.DeleteOne(ctx, bson.M{"_id": usrID})
	metrics.ObserveDatabaseOperation("delete_user", "mongo", time.Since(start), err)
	if err != nil {
		logrus.Error(err)
	}
}

func (r *UserRepository) CheckBanned(ctx context.Context, userid string) bool {
	start := time.Now()
	count, err := r.banned.CountDocuments(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("check_banned", "mongo", time.Since(start), err)
	return count > 0
}

func (r *UserRepository) BanUser(ctx context.Context, userid string) error {
	start := time.Now()
	_, err := r.banned.InsertOne(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("ban_user", "mongo", time.Since(start), err)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *UserRepository) UnbanUser(ctx context.Context, userid string) error {
	start := time.Now()
	res, err := r.banned.DeleteOne(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("unban_user", "mongo", time.Since(start), err)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("not banned")
	}
	return nil
}

func (r *UserRepository) GrantAdminRole(ctx context.Context, userid string) error {
	start := time.Now()
	_, err := r.adminGrants.InsertOne(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("grant_admin", "mongo", time.Since(start), err)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *UserRepository) RevokeAdminRole(ctx context.Context, userid string) error {
	start := time.Now()
	res, err := r.adminGrants.DeleteOne(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("revoke_admin", "mongo", time.Since(start), err)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("not granted")
	}
	return nil
}

func (r *UserRepository) IsDynamicAdmin(ctx context.Context, userid string) bool {
	start := time.Now()
	count, err := r.adminGrants.CountDocuments(ctx, bson.M{"_id": userid})
	metrics.ObserveDatabaseOperation("is_dynamic_admin", "mongo", time.Since(start), err)
	if err != nil {
		logrus.Error(err)
		return false
	}
	return count > 0
}

func (r *UserRepository) ListUserIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	start := time.Now()
	opts := options.Find().
		SetLimit(int64(limit)).
		SetProjection(bson.M{"_id": 1})
	cur, err := r.users.Find(ctx, bson.M{}, opts)
	metrics.ObserveDatabaseOperation("list_user_ids", "mongo", time.Since(start), err)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []string
	for cur.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.ID)
	}
	return out, cur.Err()
}

func (r *UserRepository) UpdateUserStats(ctx context.Context, usr domain.User, activeDiff, totalDiff int) error {
	start := time.Now()
	update := bson.M{
		"$set": bson.M{
			"stats.total_pixels_placed": totalDiff,
			"stats.active_pixels":       activeDiff,
		},
	}
	_, err := r.users.UpdateOne(ctx, bson.M{"_id": usr.ID}, update)
	metrics.ObserveDatabaseOperation("update_user_stats", "mongo", time.Since(start), err)
	return err
}

func (r *UserRepository) GetTopUsers(ctx context.Context, limit int) ([]domain.BroadcastStats, error) {
	start := time.Now()
	var topUsers []domain.BroadcastStats

	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$project", Value: bson.M{
				"_id":                 "$_id",
				"first_name":          "$first_name",
				"surname":             "$last_name",
				"total_pixels_placed": "$stats.total_pixels_placed",
				"active_pixels":       "$stats.active_pixels",
			}},
		},
		bson.D{
			{Key: "$sort", Value: bson.M{"total_pixels_placed": -1}},
		},
		bson.D{
			{Key: "$limit", Value: limit},
		},
	}

	cursor, err := r.users.Aggregate(ctx, pipeline)
	if err != nil {
		metrics.ObserveDatabaseOperation("get_top_users", "mongo", time.Since(start), err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user domain.BroadcastStats
		if err := cursor.Decode(&user); err != nil {
			metrics.ObserveDatabaseOperation("get_top_users", "mongo", time.Since(start), err)
			return nil, err
		}
		topUsers = append(topUsers, user)
	}

	if err := cursor.Err(); err != nil {
		metrics.ObserveDatabaseOperation("get_top_users", "mongo", time.Since(start), err)
		return nil, err
	}

	metrics.ObserveDatabaseOperation("get_top_users", "mongo", time.Since(start), nil)
	return topUsers, nil
}

func (r *UserRepository) IsEmpty(ctx context.Context) (bool, error) {
	start := time.Now()
	count, err := r.users.CountDocuments(ctx, bson.M{})
	metrics.ObserveDatabaseOperation("is_empty", "mongo", time.Since(start), err)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}
