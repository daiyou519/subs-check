package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var (
	// DB Global database connection instance
	DB      *sql.DB
	once    sync.Once
	ErrNoTx = errors.New("no transaction provided")
)

// Config Database configuration
type Config struct {
	// Database path
	Path string
	// Maximum number of idle connections
	MaxIdleConns int
	// Maximum number of open connections
	MaxOpenConns int
	// Maximum lifetime of connections
	ConnMaxLifetime time.Duration
}

// DefaultConfig Returns default configuration
func DefaultConfig(path string) Config {
	return Config{
		Path:            path,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}
}

// InitDatabase Initializes database connection and creates table structure
func InitDatabase(dbPath string) error {
	return InitDatabaseWithConfig(DefaultConfig(dbPath))
}

// InitDatabaseWithConfig Initializes database with custom configuration
func InitDatabaseWithConfig(config Config) error {
	var err error

	once.Do(func() {
		DB, err = setupDatabase(config)
	})

	return err
}

// setupDatabase Sets up database connection and structure
func setupDatabase(config Config) (*sql.DB, error) {
	logger.Info("Opening database connection to %s", config.Path)

	db, err := sql.Open("sqlite3", config.Path+"?_loc=auto&_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	if err := createInitialAdminUser(db); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database initialized successfully")
	return db, nil
}

// createSchema Creates database table structure
func createSchema(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS subs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL,
			last_check DATETIME,
			last_fetch DATETIME,
			cron TEXT DEFAULT '0 */1 * * *',
			auto_update INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			total_nodes INTEGER DEFAULT 0,
			alive_nodes INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// createInitialAdminUser Creates initial admin account
func createInitialAdminUser(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = 1").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO users (id, username, password) VALUES (1, ?, ?)",
			"admin",
			string(hashedPassword),
		)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		logger.Info("Initial admin user (ID: 1) created")
	}

	return nil
}

// WithTransaction Executes a function within a transaction
func WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func Close() error {
	if DB != nil {
		logger.Info("Closing database connection")
		return DB.Close()
	}
	return nil
}
