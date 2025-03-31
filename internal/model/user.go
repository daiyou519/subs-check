package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User User model
// Represents user account in the system
type User struct {
	ID        int64     `json:"id" example:"1"`
	Username  string    `json:"username" example:"admin"`
	Password  string    `json:"-"` // Password should not be returned in JSON response
	CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// NewUser Create new user instance
func NewUser(username, password string) (*User, error) {
	user := &User{
		Username: username,
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	return user, nil
}

// SetPassword Set password (encrypted storage)
func (u *User) SetPassword(password string) error {
	// Use bcrypt algorithm to encrypt password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword Check if password matches
func (u *User) CheckPassword(password string) bool {
	// Compare provided password and stored hash
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsAdmin Check if user is admin
func (u *User) IsAdmin() bool {
	// Simple implementation: User with ID 1 is considered an admin
	return u.ID == 1
}

// Clone Create a deep copy of the user object
func (u *User) Clone() *User {
	clone := *u
	return &clone
}

// Sanitize Remove sensitive information, used for API response
func (u *User) Sanitize() *User {
	sanitized := u.Clone()
	sanitized.Password = ""
	return sanitized
}
