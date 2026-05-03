package config

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func OpenDB(cfg Config) (*bun.DB, error) {
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresDSN())))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := ensureSchema(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return bun.NewDB(sqlDB, pgdialect.New()), nil
}

func ensureSchema(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stmts := []string{
		"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS is_closed BOOLEAN NOT NULL DEFAULT FALSE",
		"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ",
		"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS final_slot VARCHAR(16)",
		"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS finalized_by VARCHAR(64)",
		"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS finalized_at TIMESTAMPTZ",
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}
