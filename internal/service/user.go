package service

import (
	"context"
	"errors"

	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// UserService User related business logic service
type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService Create a new user service instance
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// CreateUser Create a new user
func (s *UserService) CreateUser(ctx context.Context, username, password string) (*model.User, error) {
	// Create user object
	user := &model.User{
		Username: username,
	}

	// Hash password
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user.Password = hashedPassword

	// Save to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Authenticate User authentication
func (s *UserService) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	// Get user
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if !s.VerifyPassword(user.Password, password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// ChangePassword Change user password
func (s *UserService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !s.VerifyPassword(user.Password, oldPassword) {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	return s.userRepo.UpdatePassword(ctx, userID, hashedPassword)
}

// UpdateUserInfo Update user information
func (s *UserService) UpdateUserInfo(ctx context.Context, user *model.User) error {
	return s.userRepo.Update(ctx, user)
}

// HashPassword Hash password
func (s *UserService) HashPassword(password string) (string, error) {
	// Use bcrypt algorithm to hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// VerifyPassword Verify if password matches
func (s *UserService) VerifyPassword(hashedPassword, password string) bool {
	// Compare provided password and stored hash
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// IsAdmin Check if user is an admin
func (s *UserService) IsAdmin(user *model.User) bool {
	// Simple implementation: User with ID 1 is considered an admin
	return user.ID == 1
}

// SanitizeUser Remove sensitive information, used for API response
func (s *UserService) SanitizeUser(user *model.User) *model.User {
	// Create a deep copy of the user
	sanitized := *user
	// Clear password
	sanitized.Password = ""
	return &sanitized
}
