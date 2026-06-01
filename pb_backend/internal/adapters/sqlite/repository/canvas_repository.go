package repository

import (
	"context"
	"database/sql"
	"fmt"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"

	"github.com/sirupsen/logrus"
)

type CanvasRepository struct {
	db *sql.DB
}

func NewCanvasRepository(db *sql.DB) *CanvasRepository {
	return &CanvasRepository{db: db}
}

// WritePixel appends a pixel change to history (append-only, no conflicts)
func (r *CanvasRepository) WritePixel(ctx context.Context, x, y uint, pixelData []byte) error {
	query := `INSERT INTO pixel_history (x, y, pixel_data, created_at) VALUES (?, ?, ?, strftime('%s', 'now'))`
	_, err := r.db.ExecContext(ctx, query, x, y, pixelData)
	if err != nil {
		return fmt.Errorf("failed to write pixel at (%d, %d): %w", x, y, err)
	}
	return nil
}

// CheckInitialized checks if canvas has any pixels
func (r *CanvasRepository) CheckInitialized(ctx context.Context) bool {
	query := `SELECT COUNT(DISTINCT (x, y)) > 0 FROM pixel_history`
	var initialized bool
	err := r.db.QueryRowContext(ctx, query).Scan(&initialized)
	if err != nil {
		logrus.Error("Failed to check initialization: ", err)
		return false
	}
	return initialized
}

// GetCanvas retrieves the latest pixel state for all coordinates
// Uses window function ROW_NUMBER() for SQLite compatibility
func (r *CanvasRepository) GetCanvas(ctx context.Context) (map[string][]string, error) {
	query := `
		SELECT x, y, pixel_data, created_at
		FROM (
			SELECT x, y, pixel_data, created_at,
				   ROW_NUMBER() OVER (PARTITION BY x, y ORDER BY created_at DESC) as rn
			FROM pixel_history
		) WHERE rn = 1
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get canvas: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var x, y uint
		var pixelData []byte
		var createdAt sql.NullInt64

		if err := rows.Scan(&x, &y, &pixelData, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		key := fmt.Sprintf("pixel:%d:%d", y, x)
		result[key] = []string{string(pixelData)}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

func (r *CanvasRepository) GetLatestPixel(ctx context.Context, x, y uint) (domain.RedisPixel, error) {
	query := `
		SELECT pixel_data
		FROM pixel_history
		WHERE x = ? AND y = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var pixelData []byte
	if err := r.db.QueryRowContext(ctx, query, x, y).Scan(&pixelData); err != nil {
		return domain.RedisPixel{}, err
	}
	var pixel domain.RedisPixel
	if err := utils.DeserializeRedisPixel(pixelData, &pixel); err != nil {
		return domain.RedisPixel{}, err
	}
	return pixel, nil
}

// LoadHeatMap counts history length per coordinate
func (r *CanvasRepository) LoadHeatMap(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT x, y, COUNT(*) as history_length
		FROM pixel_history
		GROUP BY x, y
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load heatmap: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var x, y uint
		var length int64

		if err := rows.Scan(&x, &y, &length); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		key := fmt.Sprintf("pixel:%d:%d", y, x)
		result[key] = length
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetCanvasDimensions returns (0,0) so service infers from GetCanvas (no meta table).
func (r *CanvasRepository) GetCanvasDimensions(ctx context.Context) (uint, uint, error) {
	_, _ = ctx, r
	return 0, 0, nil
}

// SetCanvasDimensions is a no-op for SQLite (extent is implicit in data).
func (r *CanvasRepository) SetCanvasDimensions(ctx context.Context, width, height uint) error {
	_, _, _, _ = ctx, r, width, height
	return nil
}

// Ensure CanvasRepository implements domain.CanvasRepository interface
var _ domain.CanvasRepository = (*CanvasRepository)(nil)
