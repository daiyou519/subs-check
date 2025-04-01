package model

import (
	"errors"
	"time"
)

var (
	ErrSubNotFound   = errors.New("sub not found")
	ErrSubExists     = errors.New("sub already exists")
	ErrFetchFailed   = errors.New("failed to fetch subscription data")
	ErrInvalidSubURL = errors.New("invalid subscription URL")
	ErrParsingFailed = errors.New("failed to parse subscription content")
)

// Sub represents a subscription entry
type Sub struct {
	ID         int64      `json:"id"`
	URL        string     `json:"url"`
	LastCheck  *time.Time `json:"last_check,omitempty"`
	LastFetch  *time.Time `json:"last_fetch,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	TotalNodes int        `json:"total_nodes"`
	AliveNodes int        `json:"alive_nodes"`
	Cron       string     `json:"cron,omitempty"`
	AutoUpdate bool       `json:"auto_update"`
}
