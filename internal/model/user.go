package model

import (
	"time"
)

// User User model
type User struct {
	ID        int64     `json:"id" example:"1"`
	Username  string    `json:"username" example:"admin"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}
