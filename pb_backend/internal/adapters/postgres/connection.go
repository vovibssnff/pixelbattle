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

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

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
	CREATE TABLE IF NOT EXISTS pixel_history (
		id BIGSERIAL PRIMARY KEY,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		pixel_data BYTEA NOT NULL,
		created_at TIMESTAMP DEFAULT NOW() NOT NULL
	);

	-- GetCanvas: DISTINCT ON (x, y) ORDER BY x, y, created_at DESC
	CREATE INDEX IF NOT EXISTS idx_pixel_latest_lookup ON pixel_history(x, y, created_at DESC);

	-- LoadHeatMap: GROUP BY x, y COUNT(*)
	CREATE INDEX IF NOT EXISTS idx_pixel_count ON pixel_history(x, y);

	-- Drop redundant indexes that were never used (0 scans in pg_stat_user_indexes)
	DROP INDEX IF EXISTS idx_pixel_coords;
	DROP INDEX IF EXISTS idx_pixel_created_at;
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	logrus.Info("PostgreSQL schema initialized successfully")
	return nil
}
