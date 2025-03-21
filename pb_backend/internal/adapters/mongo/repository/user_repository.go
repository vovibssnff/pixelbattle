package repository

import (
	"context"
	"pb_backend/internal/core/domain"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	users  *mongo.Collection
	banned *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		users:  db.Collection("users"),
		banned: db.Collection("banned_users"),
	}
}

// RegisterUser registers a new user in MongoDB
func (r *UserRepository) RegisterUser(ctx context.Context, usr domain.User) error {
	mongoUser := domain.User{
		ID:          usr.ID,
		FirstName:   usr.FirstName,
		LastName:    usr.LastName,
		AccessToken: usr.AccessToken,
		Faculty:     usr.Faculty,
		Stats:       domain.UserStats{TotalPixelsPlaced: 0, ActivePixels: 0},
	}

	_, err := r.users.InsertOne(ctx, mongoUser)
	return err
}

// UserExists checks if a user exists in MongoDB
func (r *UserRepository) UserExists(ctx context.Context, usrID int) bool {
	count, err := r.users.CountDocuments(ctx, bson.M{"_id": usrID})
	if err != nil {
		logrus.Error(err)
		return false
	}
	return count > 0
}

// GetUsr retrieves a user from MongoDB
func (r *UserRepository) GetUsr(ctx context.Context, usrID int) domain.User {
	var mongoUser domain.User
	err := r.users.FindOne(ctx, bson.M{"_id": usrID}).Decode(&mongoUser)
	if err != nil {
		logrus.Error(err)
		return domain.User{}
	}

	return domain.User{
		ID:          mongoUser.ID,
		FirstName:   mongoUser.FirstName,
		LastName:    mongoUser.LastName,
		AccessToken: mongoUser.AccessToken,
		Faculty:     mongoUser.Faculty,
	}
}

// DelUsr deletes a user from MongoDB
func (r *UserRepository) DelUsr(ctx context.Context, usrID int) {
	_, err := r.users.DeleteOne(ctx, bson.M{"_id": usrID})
	if err != nil {
		logrus.Error(err)
	}
}

// CheckBanned checks if a user is banned
func (r *UserRepository) CheckBanned(ctx context.Context, userid int) bool {
	count, _ := r.banned.CountDocuments(ctx, bson.M{"_id": userid})
	return count > 0
}

// UpdateUserStats updates the user's pixel statistics
func (r *UserRepository) UpdateUserStats(ctx context.Context, usr domain.User, activeDiff, totalDiff int) error {
	update := bson.M{
		"$set": bson.M{
			"stats.total_pixels_placed": totalDiff,
			"stats.active_pixels":       activeDiff,
		},
	}
	_, err := r.users.UpdateOne(ctx, bson.M{"_id": usr}, update)
	return err
}

// GetTopUsers retrieves the top users based on total pixels placed.
func (r *UserRepository) GetTopUsers(ctx context.Context, limit int) ([]domain.BroadcastStats, error) {
	var topUsers []domain.BroadcastStats

	// Aggregate to get the top users based on total pixels placed
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
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user domain.BroadcastStats
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		topUsers = append(topUsers, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return topUsers, nil
}
	

// IsEmpty checks if the users collection is empty
func (r *UserRepository) IsEmpty(ctx context.Context) (bool, error) {
	count, err := r.users.CountDocuments(ctx, bson.M{})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}
