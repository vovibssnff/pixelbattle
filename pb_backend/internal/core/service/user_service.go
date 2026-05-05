package service

import (
	"context"
	"errors"
	"fmt"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

type UserService struct {
	repo         domain.UserRepository
	admIds       []int
	admUsernames []string
}

func NewUserService(userRepo domain.UserRepository, admIds []int, admUsernames []string) *UserService {
	return &UserService{
		repo:         userRepo,
		admIds:       admIds,
		admUsernames: admUsernames,
	}
}

// CreateUser builds a VK-backed user document (ID must be vk_*).
func (s *UserService) CreateUser(id string, firstName, lastName, accessToken string) *domain.User {
	return &domain.User{
		ID:          id,
		FirstName:   firstName,
		LastName:    lastName,
		AccessToken: accessToken,
	}
}

// RegisterUser registers a new user (Mongo).
func (s *UserService) RegisterUser(ctx context.Context, usr domain.User) error {
	return s.repo.RegisterUser(ctx, usr)
}

// RegisterWithPassword creates a password-based user with normalized username as _id.
func (s *UserService) RegisterWithPassword(ctx context.Context, username, password, faculty string) (*domain.User, error) {
	u := utils.NormalizeUsername(username)
	if !utils.IsValidLocalUsername(u) {
		return nil, errors.New("invalid username")
	}
	if len(password) < 8 {
		return nil, errors.New("password too short")
	}
	if !utils.IsValidFaculty(faculty) {
		return nil, errors.New("invalid faculty")
	}
	if s.repo.UserExists(ctx, u) {
		return nil, errors.New("username taken")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	usr := &domain.User{
		ID:           u,
		FirstName:    u,
		LastName:     "",
		PasswordHash: string(hash),
		Faculty:      faculty,
	}
	if err := s.repo.RegisterUser(ctx, *usr); err != nil {
		return nil, err
	}
	usr.PasswordHash = ""
	return usr, nil
}

// LoginWithPassword verifies credentials for a local user.
func (s *UserService) LoginWithPassword(ctx context.Context, username, password string) (*domain.User, error) {
	u := utils.NormalizeUsername(username)
	usr, err := s.repo.GetUserWithHash(ctx, u)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if usr.PasswordHash == "" {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	usr.PasswordHash = ""
	return &usr, nil
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, usr domain.User) error {
	return s.repo.UpdateUser(ctx, usr)
}

// UserExists checks if a user exists.
func (s *UserService) UserExists(ctx context.Context, usrID string) bool {
	return s.repo.UserExists(ctx, usrID)
}

// GetUser retrieves a user.
func (s *UserService) GetUser(ctx context.Context, usrID string) domain.User {
	return s.repo.GetUsr(ctx, usrID)
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, usrID string) {
	s.repo.DelUsr(ctx, usrID)
}

// IsUserBanned checks if a user is banned.
func (s *UserService) IsUserBanned(ctx context.Context, userid string) bool {
	return s.repo.CheckBanned(ctx, userid)
}

// IsAdmin returns true if id is a configured admin (VK id as vk_* or local username).
func (s *UserService) IsAdmin(id string) bool {
	idl := strings.ToLower(strings.TrimSpace(id))
	for _, name := range s.admUsernames {
		if idl == name {
			return true
		}
	}
	for _, vid := range s.admIds {
		if id == domain.VKUserID(vid) {
			return true
		}
	}
	return false
}

// IsEffectiveAdmin includes static admins and Mongo `admin_grants`.
func (s *UserService) IsEffectiveAdmin(ctx context.Context, id string) bool {
	if s.IsAdmin(id) {
		return true
	}
	return s.repo.IsDynamicAdmin(ctx, strings.TrimSpace(id))
}

// GrantAdminRole persists a dynamic admin grant (Mongo).
func (s *UserService) GrantAdminRole(ctx context.Context, userid string) error {
	userid = strings.TrimSpace(userid)
	if userid == "" {
		return fmt.Errorf("empty user id")
	}
	return s.repo.GrantAdminRole(ctx, userid)
}

// RevokeAdminRole removes a dynamic admin grant.
func (s *UserService) RevokeAdminRole(ctx context.Context, userid string) error {
	userid = strings.TrimSpace(userid)
	if userid == "" {
		return fmt.Errorf("empty user id")
	}
	return s.repo.RevokeAdminRole(ctx, userid)
}

// ListUserIDs returns up to `limit` user _id values for admin UI dropdowns.
func (s *UserService) ListUserIDs(ctx context.Context, limit int) ([]string, error) {
	return s.repo.ListUserIDs(ctx, limit)
}

// BanUser bans a user by canonical id (e.g. vk_123 or local username).
func (s *UserService) BanUser(ctx context.Context, userid string) error {
	userid = strings.TrimSpace(userid)
	if userid == "" {
		return fmt.Errorf("empty user id")
	}
	return s.repo.BanUser(ctx, userid)
}

// UnbanUser removes a ban.
func (s *UserService) UnbanUser(ctx context.Context, userid string) error {
	userid = strings.TrimSpace(userid)
	if userid == "" {
		return fmt.Errorf("empty user id")
	}
	return s.repo.UnbanUser(ctx, userid)
}
