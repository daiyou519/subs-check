package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bestruirui/bestsub/internal/database"
	"github.com/bestruirui/bestsub/internal/model"
)

var (
	// ErrUserNotFound User not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists User already exists
	ErrUserExists = errors.New("user already exists")
)

// UserRepository User data access interface
type UserRepository interface {
	// GetByID Get user by ID
	GetByID(ctx context.Context, id int64) (*model.User, error)
	// GetByUsername Get user by username
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	// Create Create new user
	Create(ctx context.Context, user *model.User) error
	// Update Update user information
	Update(ctx context.Context, user *model.User) error
	// UpdatePassword Update user password
	UpdatePassword(ctx context.Context, userID int64, hashedPassword string) error
	// Delete Delete user
	Delete(ctx context.Context, id int64) error
}

// SQLUserRepository SQL-based user storage repository implementation
type SQLUserRepository struct {
	db *sql.DB
}

// NewUserRepository Create new user storage repository
func NewUserRepository(db *sql.DB) UserRepository {
	return &SQLUserRepository{db: db}
}

// GetByID Get user by ID
func (r *SQLUserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `SELECT id, username, password, created_at, updated_at
	          FROM users 
			  WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	user := &model.User{}
	var createdAt, updatedAt string

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Parse timestamp
	if user.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	if user.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return user, nil
}

// GetByUsername Get user by username
func (r *SQLUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, password, created_at, updated_at
	          FROM users 
			  WHERE username = ?`

	row := r.db.QueryRowContext(ctx, query, username)

	user := &model.User{}
	var createdAt, updatedAt string

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Parse timestamp
	if user.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	if user.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return user, nil
}

// Create Create new user
func (r *SQLUserRepository) Create(ctx context.Context, user *model.User) error {
	// Use transaction to ensure atomicity
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if user already exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)",
			user.Username,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if exists {
			return ErrUserExists
		}

		// Insert new user
		now := time.Now().UTC().Format(time.RFC3339)
		result, err := tx.ExecContext(ctx,
			`INSERT INTO users (username, password, created_at, updated_at) 
			 VALUES (?, ?, ?, ?)`,
			user.Username,
			user.Password,
			now,
			now,
		)

		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Get auto-increment ID
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		user.ID = id
		user.CreatedAt, _ = time.Parse(time.RFC3339, now)
		user.UpdatedAt = user.CreatedAt

		return nil
	})
}

// Update Update user information
func (r *SQLUserRepository) Update(ctx context.Context, user *model.User) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if user exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
			user.ID,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if !exists {
			return ErrUserNotFound
		}

		// Update user information
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE users 
			 SET username = ?, updated_at = ? 
			 WHERE id = ?`,
			user.Username,
			now,
			user.ID,
		)

		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		// Update in-memory object
		user.UpdatedAt, _ = time.Parse(time.RFC3339, now)

		return nil
	})
}

// UpdatePassword Update user password
func (r *SQLUserRepository) UpdatePassword(ctx context.Context, userID int64, hashedPassword string) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if user exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
			userID,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if !exists {
			return ErrUserNotFound
		}

		// Update password
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE users 
			 SET password = ?, updated_at = ? 
			 WHERE id = ?`,
			hashedPassword,
			now,
			userID,
		)

		if err != nil {
			return fmt.Errorf("failed to update user password: %w", err)
		}

		return nil
	})
}

// Delete Delete user
func (r *SQLUserRepository) Delete(ctx context.Context, id int64) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if user exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if !exists {
			return ErrUserNotFound
		}

		// Delete user
		_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})
}
