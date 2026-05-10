package repository

import (
	"context"
	"database/sql"
	"fmt"
	"pb_backend/internal/core/domain"

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
	query := `INSERT INTO pixel_history (x, y, pixel_data, created_at) VALUES ($1, $2, $3, NOW())`
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
// Uses DISTINCT ON for optimal performance (PostgreSQL-specific optimization)
func (r *CanvasRepository) GetCanvas(ctx context.Context) (map[string][]string, error) {
	query := `
		SELECT DISTINCT ON (x, y) x, y, pixel_data, created_at
		FROM pixel_history
		ORDER BY x, y, created_at DESC
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
		var createdAt sql.NullTime

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

// Ensure CanvasRepository implements domain.CanvasRepository interface
var _ domain.CanvasRepository = (*CanvasRepository)(nil)
