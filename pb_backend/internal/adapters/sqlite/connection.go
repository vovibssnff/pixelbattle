package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func NewSQLiteConnection(dbPath string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite has limited concurrency - use small connection pool
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Info("SQLite connection established successfully")
	return db, nil
}

// InitializeSchema creates the pixel_history table with optimized indexes
func InitializeSchema(db *sql.DB) error {
	schema := `
	-- Append-only table: EVERY pixel change is stored, no data loss
	CREATE TABLE IF NOT EXISTS pixel_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		pixel_data BLOB NOT NULL,
		created_at INTEGER DEFAULT (strftime('%s', 'now')) NOT NULL
	);

	-- Optimized indexes for specific operations:
	-- 1. Fast latest pixel lookup per coordinate (GetCanvas, GetPixel)
	CREATE INDEX IF NOT EXISTS idx_pixel_latest_lookup ON pixel_history(x, y, created_at DESC);

	-- 2. Fast coordinate-based queries
	CREATE INDEX IF NOT EXISTS idx_pixel_coords ON pixel_history(x, y);

	-- 3. Fast history counting per coordinate (LoadHeatMap)
	CREATE INDEX IF NOT EXISTS idx_pixel_count ON pixel_history(x, y);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	logrus.Info("SQLite schema initialized successfully")
	return nil
}
