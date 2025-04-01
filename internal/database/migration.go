package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
)

type MigrationFunc func(tx *sql.Tx) error

type Migration struct {
	Version     int
	Description string
	Execute     MigrationFunc
}

var migrations = []Migration{
	{
		Version:     1,
		Description: "添加迁移版本表",
		Execute:     createMigrationTable,
	},
	{
		Version:     2,
		Description: "添加节点统计字段到subs表",
		Execute:     addNodesStatsColumns,
	},
}

func RunMigrations(db *sql.DB) error {
	logger.Info("Running database migrations...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := ensureMigrationTableExists(tx); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	currentVersion, err := getCurrentVersion(tx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	logger.Info("Current database version: %d", currentVersion)

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		logger.Info("Applying migration %d: %s", migration.Version, migration.Description)

		if err := migration.Execute(tx); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if err := updateVersionRecord(tx, migration.Version, migration.Description); err != nil {
			return fmt.Errorf("failed to update version record: %w", err)
		}

		logger.Info("Successfully applied migration %d", migration.Version)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}

// ensureMigrationTableExists 确保迁移表存在
func ensureMigrationTableExists(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER NOT NULL,
			description TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// getCurrentVersion 获取当前数据库版本
func getCurrentVersion(tx *sql.Tx) (int, error) {
	var exists int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' AND name='migrations'
	`).Scan(&exists)

	if err != nil {
		return 0, err
	}

	if exists == 0 {
		return 0, nil
	}

	var version int
	err = tx.QueryRow(`
		SELECT COALESCE(MAX(version), 0) FROM migrations
	`).Scan(&version)

	if err != nil {
		return 0, err
	}

	return version, nil
}

// updateVersionRecord 更新版本记录
func updateVersionRecord(tx *sql.Tx, version int, description string) error {
	_, err := tx.Exec(`
		INSERT INTO migrations (version, description)
		VALUES (?, ?)
	`, version, description)
	return err
}

// createMigrationTable 初始迁移：创建迁移版本表
func createMigrationTable(tx *sql.Tx) error {
	return nil
}

// addNodesStatsColumns 迁移：添加节点统计相关字段到subs表
func addNodesStatsColumns(tx *sql.Tx) error {
	var countTotal, countAlive int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('subs') 
		WHERE name = 'total_nodes'
	`).Scan(&countTotal)
	if err != nil {
		return fmt.Errorf("failed to check if total_nodes column exists: %w", err)
	}

	err = tx.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('subs') 
		WHERE name = 'alive_nodes'
	`).Scan(&countAlive)
	if err != nil {
		return fmt.Errorf("failed to check if alive_nodes column exists: %w", err)
	}

	if countTotal == 0 {
		_, err = tx.Exec("ALTER TABLE subs ADD COLUMN total_nodes INTEGER DEFAULT 0")
		if err != nil {
			return fmt.Errorf("failed to add total_nodes column: %w", err)
		}
	}

	if countAlive == 0 {
		_, err = tx.Exec("ALTER TABLE subs ADD COLUMN alive_nodes INTEGER DEFAULT 0")
		if err != nil {
			return fmt.Errorf("failed to add alive_nodes column: %w", err)
		}
	}

	return nil
}

func addNewColumnMigration(tx *sql.Tx) error {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('table_name') 
		WHERE name = 'column_name'
	`).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to check if column exists: %w", err)
	}

	if count == 0 {
		logger.Info("Adding column to table...")

		_, err = tx.Exec("ALTER TABLE table_name ADD COLUMN column_name COLUMN_TYPE DEFAULT default_value")
		if err != nil {
			return fmt.Errorf("failed to add column: %w", err)
		}

		logger.Info("Successfully added column to table")
	} else {
		logger.Info("Column already exists in table")
	}

	return nil
}
