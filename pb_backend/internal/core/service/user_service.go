package service

import (
	"context"
	"pb_backend/internal/core/domain"
)

type UserService struct {
	repo   domain.UserRepository
	admIds []int
}

func NewUserService(userRepo domain.UserRepository, admIds []int) *UserService {
	return &UserService{
		repo:   userRepo,
		admIds: admIds,
	}
}

func (s *UserService) CreateUser(id int, firstName, lastName, accessToken string) *domain.User {
	return &domain.User{
		ID:          id, // style
		FirstName:   firstName,
		LastName:    lastName,
		AccessToken: accessToken,
	}
}

// RegisterUser registers a new user.
func (s *UserService) RegisterUser(ctx context.Context, usr domain.User) error {
	return s.repo.RegisterUser(ctx, usr)
}

// UserExists checks if a user exists.
func (s *UserService) UserExists(ctx context.Context, usrID int) bool {
	return s.repo.UserExists(ctx, usrID)
}

// GetUser retrieves a user.
func (s *UserService) GetUser(ctx context.Context, usrID int) domain.User {
	return s.repo.GetUsr(ctx, usrID)
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, usrID int) {
	s.repo.DelUsr(ctx, usrID)
}

// IsUserBanned checks if a user is banned.
func (s *UserService) IsUserBanned(ctx context.Context, userid int) bool {
	return s.repo.CheckBanned(ctx, userid)
}

func (s *UserService) IsAdmin(id int) bool {
	for _, admId := range s.admIds {
		if id == admId {
			return true 
		}
	}
	return false
}
