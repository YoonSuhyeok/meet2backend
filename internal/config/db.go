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
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
				  AND table_name = 'notification_subscriptions'
			) THEN
				IF EXISTS (
					SELECT 1
					FROM information_schema.columns
					WHERE table_schema = 'public'
					  AND table_name = 'notification_subscriptions'
					  AND column_name = 'meeting_id'
				) THEN
					ALTER TABLE notification_subscriptions ALTER COLUMN meeting_id DROP NOT NULL;
				END IF;

				ALTER TABLE notification_subscriptions DROP CONSTRAINT IF EXISTS notification_subscriptions_uniq;
				DROP INDEX IF EXISTS notification_subscriptions_meeting_active_idx;

				WITH ranked AS (
					SELECT
						id,
						ROW_NUMBER() OVER (
							PARTITION BY user_id, device_id
							ORDER BY is_active DESC, updated_at DESC, created_at DESC, id DESC
						) AS row_num
					FROM notification_subscriptions
				)
				DELETE FROM notification_subscriptions ns
				USING ranked
				WHERE ns.id = ranked.id
				  AND ranked.row_num > 1;

				CREATE UNIQUE INDEX IF NOT EXISTS notification_subscriptions_user_device_unique_idx
					ON notification_subscriptions (user_id, device_id);
				CREATE INDEX IF NOT EXISTS notification_subscriptions_active_idx
					ON notification_subscriptions (user_id, is_active, endpoint_status);
			END IF;
		END $$`,
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}
