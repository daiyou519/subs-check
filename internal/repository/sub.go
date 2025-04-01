package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bestruirui/bestsub/internal/database"
	"github.com/bestruirui/bestsub/internal/model"
)

// SubRepository Sub data access interface
type SubRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Sub, error)
	GetAll(ctx context.Context) ([]*model.Sub, error)
	GetAllAutoUpdateSubs(ctx context.Context) ([]*model.Sub, error)
	Create(ctx context.Context, sub *model.Sub) error
	Update(ctx context.Context, sub *model.Sub) error
	Delete(ctx context.Context, id int64) error
	UpdateStats(ctx context.Context, id int64, totalNodes, aliveNodes int) error
	UpdateLastCheck(ctx context.Context, id int64) error
	UpdateLastFetch(ctx context.Context, id int64) error
	UpdateCronSettings(ctx context.Context, id int64, cron string, autoUpdate bool) error
}

// SQLSubRepository SQL-based sub storage repository implementation
type SQLSubRepository struct {
	db *sql.DB
}

// NewSubRepository Create new sub storage repository
func NewSubRepository(db *sql.DB) SubRepository {
	return &SQLSubRepository{db: db}
}

// GetByID Get sub by ID
func (r *SQLSubRepository) GetByID(ctx context.Context, id int64) (*model.Sub, error) {
	query := `SELECT id, url, last_check, last_fetch, created_at, updated_at, total_nodes, alive_nodes, cron, auto_update
	          FROM subs 
			  WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	sub := &model.Sub{}
	var lastCheck, lastFetch sql.NullTime
	var createdAt, updatedAt string
	var autoUpdate int

	err := row.Scan(
		&sub.ID,
		&sub.URL,
		&lastCheck,
		&lastFetch,
		&createdAt,
		&updatedAt,
		&sub.TotalNodes,
		&sub.AliveNodes,
		&sub.Cron,
		&autoUpdate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrSubNotFound
		}
		return nil, fmt.Errorf("failed to get sub by ID: %w", err)
	}

	if lastCheck.Valid {
		sub.LastCheck = &lastCheck.Time
	}

	if lastFetch.Valid {
		sub.LastFetch = &lastFetch.Time
	}

	sub.AutoUpdate = autoUpdate == 1

	if sub.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	if sub.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return sub, nil
}

// GetAll Get all subs
func (r *SQLSubRepository) GetAll(ctx context.Context) ([]*model.Sub, error) {
	query := `SELECT id, url, last_check, last_fetch, created_at, updated_at, total_nodes, alive_nodes, cron, auto_update
	          FROM subs 
			  ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all subs: %w", err)
	}
	defer rows.Close()

	var subs []*model.Sub
	for rows.Next() {
		sub := &model.Sub{}
		var lastCheck, lastFetch sql.NullTime
		var createdAt, updatedAt string
		var autoUpdate int

		err := rows.Scan(
			&sub.ID,
			&sub.URL,
			&lastCheck,
			&lastFetch,
			&createdAt,
			&updatedAt,
			&sub.TotalNodes,
			&sub.AliveNodes,
			&sub.Cron,
			&autoUpdate,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan sub row: %w", err)
		}

		if lastCheck.Valid {
			sub.LastCheck = &lastCheck.Time
		}

		if lastFetch.Valid {
			sub.LastFetch = &lastFetch.Time
		}

		// 将SQLite的整数布尔值转换为Go布尔值
		sub.AutoUpdate = autoUpdate == 1

		// Parse timestamps
		if sub.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		if sub.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}

		subs = append(subs, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sub rows: %w", err)
	}

	return subs, nil
}

// GetAllAutoUpdateSubs 获取所有启用了自动更新的订阅
func (r *SQLSubRepository) GetAllAutoUpdateSubs(ctx context.Context) ([]*model.Sub, error) {
	query := `SELECT id, url, last_check, last_fetch, created_at, updated_at, total_nodes, alive_nodes, cron, auto_update
	          FROM subs 
			  WHERE auto_update = 1
			  ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get auto-update subs: %w", err)
	}
	defer rows.Close()

	var subs []*model.Sub
	for rows.Next() {
		sub := &model.Sub{}
		var lastCheck, lastFetch sql.NullTime
		var createdAt, updatedAt string
		var autoUpdate int

		err := rows.Scan(
			&sub.ID,
			&sub.URL,
			&lastCheck,
			&lastFetch,
			&createdAt,
			&updatedAt,
			&sub.TotalNodes,
			&sub.AliveNodes,
			&sub.Cron,
			&autoUpdate,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan sub row: %w", err)
		}

		if lastCheck.Valid {
			sub.LastCheck = &lastCheck.Time
		}

		if lastFetch.Valid {
			sub.LastFetch = &lastFetch.Time
		}

		// 将SQLite的整数布尔值转换为Go布尔值
		sub.AutoUpdate = autoUpdate == 1

		// Parse timestamps
		if sub.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		if sub.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}

		subs = append(subs, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating auto-update sub rows: %w", err)
	}

	return subs, nil
}

// Create Create new sub
func (r *SQLSubRepository) Create(ctx context.Context, sub *model.Sub) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub already exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE url = ?)",
			sub.URL,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if exists {
			return model.ErrSubExists
		}

		// 将Go布尔值转换为SQLite整数值
		autoUpdateInt := 0
		if sub.AutoUpdate {
			autoUpdateInt = 1
		}

		// Insert new sub
		now := time.Now().Local().Format(time.RFC3339)
		result, err := tx.ExecContext(ctx,
			`INSERT INTO subs (url, last_check, last_fetch, created_at, updated_at, total_nodes, alive_nodes, cron, auto_update) 
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sub.URL,
			sub.LastCheck,
			sub.LastFetch,
			now,
			now,
			sub.TotalNodes,
			sub.AliveNodes,
			sub.Cron,
			autoUpdateInt,
		)

		if err != nil {
			return fmt.Errorf("failed to create sub: %w", err)
		}

		// Get auto-increment ID
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		sub.ID = id
		sub.CreatedAt, _ = time.Parse(time.RFC3339, now)
		sub.UpdatedAt = sub.CreatedAt

		return nil
	})
}

// Update Update sub information
func (r *SQLSubRepository) Update(ctx context.Context, sub *model.Sub) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			sub.ID,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// 将Go布尔值转换为SQLite整数值
		autoUpdateInt := 0
		if sub.AutoUpdate {
			autoUpdateInt = 1
		}

		// Update sub information
		now := time.Now().Local().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE subs 
			 SET url = ?, last_check = ?, last_fetch = ?, updated_at = ?, total_nodes = ?, alive_nodes = ?, cron = ?, auto_update = ?
			 WHERE id = ?`,
			sub.URL,
			sub.LastCheck,
			sub.LastFetch,
			now,
			sub.TotalNodes,
			sub.AliveNodes,
			sub.Cron,
			autoUpdateInt,
			sub.ID,
		)

		if err != nil {
			return fmt.Errorf("failed to update sub: %w", err)
		}

		// Update in-memory object
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, now)

		return nil
	})
}

// Delete Delete sub
func (r *SQLSubRepository) Delete(ctx context.Context, id int64) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// Delete sub
		_, err = tx.ExecContext(ctx, "DELETE FROM subs WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete sub: %w", err)
		}

		return nil
	})
}

// UpdateStats Update sub statistics
func (r *SQLSubRepository) UpdateStats(ctx context.Context, id int64, totalNodes, aliveNodes int) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// Update statistics
		now := time.Now().Local().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE subs 
			 SET total_nodes = ?, alive_nodes = ?, updated_at = ?
			 WHERE id = ?`,
			totalNodes,
			aliveNodes,
			now,
			id,
		)

		if err != nil {
			return fmt.Errorf("failed to update sub statistics: %w", err)
		}

		return nil
	})
}

// UpdateLastCheck Update last check time
func (r *SQLSubRepository) UpdateLastCheck(ctx context.Context, id int64) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// Update last check time
		now := time.Now().Local().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE subs 
			 SET last_check = ?, updated_at = ?
			 WHERE id = ?`,
			now,
			now,
			id,
		)

		if err != nil {
			return fmt.Errorf("failed to update last check time: %w", err)
		}

		return nil
	})
}

// UpdateLastFetch Update last fetch time
func (r *SQLSubRepository) UpdateLastFetch(ctx context.Context, id int64) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Check if sub exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// Update last fetch time
		now := time.Now().Local().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE subs 
			 SET last_fetch = ?, updated_at = ?
			 WHERE id = ?`,
			now,
			now,
			id,
		)

		if err != nil {
			return fmt.Errorf("failed to update last fetch time: %w", err)
		}

		return nil
	})
}

// UpdateCronSettings 更新订阅的定时设置
func (r *SQLSubRepository) UpdateCronSettings(ctx context.Context, id int64, cron string, autoUpdate bool) error {
	return database.WithTransaction(ctx, func(tx *sql.Tx) error {
		// 检查sub是否存在
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM subs WHERE id = ?)",
			id,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if sub exists: %w", err)
		}

		if !exists {
			return model.ErrSubNotFound
		}

		// 将Go布尔值转换为SQLite整数值
		autoUpdateInt := 0
		if autoUpdate {
			autoUpdateInt = 1
		}

		// 更新定时设置
		now := time.Now().Local().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`UPDATE subs 
			 SET cron = ?, auto_update = ?, updated_at = ?
			 WHERE id = ?`,
			cron,
			autoUpdateInt,
			now,
			id,
		)

		if err != nil {
			return fmt.Errorf("failed to update cron settings: %w", err)
		}

		return nil
	})
}
