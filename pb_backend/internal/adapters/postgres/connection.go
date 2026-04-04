package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func NewPostgresConnection(host, port, user, password, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for optimal performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Info("PostgreSQL connection established successfully")
	return db, nil
}

// InitializeSchema creates the pixel_history table with optimized indexes
func InitializeSchema(db *sql.DB) error {
	schema := `
	-- Append-only table: EVERY pixel change is stored, no data loss
	CREATE TABLE IF NOT EXISTS pixel_history (
		id BIGSERIAL PRIMARY KEY,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		pixel_data BYTEA NOT NULL,
		created_at TIMESTAMP DEFAULT NOW() NOT NULL
	);

	-- Optimized indexes for specific operations:
	-- 1. Fast latest pixel lookup per coordinate (GetCanvas, GetPixel)
	CREATE INDEX IF NOT EXISTS idx_pixel_latest_lookup ON pixel_history(x, y, created_at DESC);

	-- 2. Fast coordinate-based queries
	CREATE INDEX IF NOT EXISTS idx_pixel_coords ON pixel_history(x, y);

	-- 3. Fast history counting per coordinate (LoadHeatMap)
	CREATE INDEX IF NOT EXISTS idx_pixel_count ON pixel_history(x, y);

	-- 4. Fast timestamp-based queries (if needed for analytics)
	CREATE INDEX IF NOT EXISTS idx_pixel_created_at ON pixel_history(created_at DESC);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	logrus.Info("PostgreSQL schema initialized successfully")
	return nil
}
