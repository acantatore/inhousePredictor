package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT to_regclass('public.users') IS NOT NULL`).Scan(&exists); err != nil {
		return fmt.Errorf("check schema presence: %w", err)
	}
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	if !exists {
		initSQLBytes, err := os.ReadFile(filepath.Join(root, "migrations", "001_init.sql"))
		if err != nil {
			return fmt.Errorf("read initial migration: %w", err)
		}
		if _, err := db.Exec(ctx, string(initSQLBytes)); err != nil {
			return fmt.Errorf("run initial migration: %w", err)
		}
	}
	if _, err := db.Exec(ctx, `ALTER TABLE markets ADD COLUMN IF NOT EXISTS payout_at TIMESTAMPTZ`); err != nil {
		return fmt.Errorf("ensure payout_at: %w", err)
	}
	if _, err := db.Exec(ctx, `ALTER TABLE pools DROP COLUMN IF EXISTS k`); err != nil {
		return fmt.Errorf("drop pools.k: %w", err)
	}
	return nil
}
