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
	if _, err := db.Exec(ctx, `ALTER TABLE markets ADD COLUMN IF NOT EXISTS is_shadow BOOLEAN NOT NULL DEFAULT false`); err != nil {
		return fmt.Errorf("ensure is_shadow: %w", err)
	}
	if _, err := db.Exec(ctx, `ALTER TABLE markets ADD COLUMN IF NOT EXISTS forecast_question_id UUID`); err != nil {
		return fmt.Errorf("ensure forecast_question_id: %w", err)
	}
	if _, err := db.Exec(ctx, `ALTER TABLE pools DROP COLUMN IF EXISTS k`); err != nil {
		return fmt.Errorf("drop pools.k: %w", err)
	}
	if _, err := db.Exec(ctx, `DO $$ BEGIN CREATE TYPE forecast_question_status AS ENUM ('open', 'closed', 'resolved', 'cancelled'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;`); err != nil {
		return fmt.Errorf("ensure forecast_question_status: %w", err)
	}
	if _, err := db.Exec(ctx, `DO $$ BEGIN CREATE TYPE forecast_question_outcome AS ENUM ('delivered', 'not_delivered', 'cancelled'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;`); err != nil {
		return fmt.Errorf("ensure forecast_question_outcome: %w", err)
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS forecast_questions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			program TEXT NOT NULL,
			owner_id UUID NOT NULL REFERENCES users(id),
			resolver_id UUID NOT NULL REFERENCES users(id),
			status forecast_question_status NOT NULL DEFAULT 'open',
			outcome forecast_question_outcome,
			resolution_rule TEXT NOT NULL,
			rationale_policy_threshold INTEGER NOT NULL DEFAULT 10,
			linked_market_id UUID REFERENCES markets(id),
			closes_at TIMESTAMPTZ NOT NULL,
			resolves_at TIMESTAMPTZ NOT NULL,
			resolved_at TIMESTAMPTZ,
			evidence_url TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_question_contributors (
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (forecast_question_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_revisions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			probability_bps INTEGER NOT NULL,
			rationale TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_latest_active (
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			forecast_revision_id UUID NOT NULL REFERENCES forecast_revisions(id) ON DELETE CASCADE,
			probability_bps INTEGER NOT NULL,
			rationale TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (forecast_question_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_projections (
			forecast_question_id UUID PRIMARY KEY REFERENCES forecast_questions(id) ON DELETE CASCADE,
			official_probability_bps INTEGER NOT NULL DEFAULT 0,
			current_risk_bps INTEGER NOT NULL DEFAULT 0,
			contributor_count INTEGER NOT NULL DEFAULT 0,
			last_change_bps INTEGER NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_snapshots (
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			forecast_revision_id UUID NOT NULL REFERENCES forecast_revisions(id) ON DELETE CASCADE,
			probability_bps INTEGER NOT NULL,
			rationale TEXT NOT NULL,
			snapshotted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (forecast_question_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_score_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			probability_bps INTEGER NOT NULL,
			outcome_value INTEGER NOT NULL,
			brier_score DOUBLE PRECISION NOT NULL,
			coverage_score DOUBLE PRECISION NOT NULL,
			revision_count INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (forecast_question_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS external_signal_snapshots (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
			source TEXT NOT NULL,
			probability_bps INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS background_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			kind TEXT NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			run_at TIMESTAMPTZ NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 5,
			last_error TEXT,
			processed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_forecast_questions_program ON forecast_questions(program)`,
		`CREATE INDEX IF NOT EXISTS idx_forecast_questions_status ON forecast_questions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_forecast_revisions_question_created ON forecast_revisions(forecast_question_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_forecast_latest_active_question ON forecast_latest_active(forecast_question_id)`,
		`CREATE INDEX IF NOT EXISTS idx_background_jobs_run_at ON background_jobs(status, run_at)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("apply forecasting schema: %w", err)
		}
	}
	alterStmts := []string{
		`ALTER TABLE forecast_questions ADD COLUMN IF NOT EXISTS rationale_policy_threshold INTEGER NOT NULL DEFAULT 10`,
		`ALTER TABLE forecast_questions ADD COLUMN IF NOT EXISTS linked_market_id UUID REFERENCES markets(id)`,
		`ALTER TABLE forecast_questions ADD COLUMN IF NOT EXISTS evidence_url TEXT`,
		`ALTER TABLE forecast_projections ADD COLUMN IF NOT EXISTS current_risk_bps INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_projections ADD COLUMN IF NOT EXISTS contributor_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_projections ADD COLUMN IF NOT EXISTS last_change_bps INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_latest_active ADD COLUMN IF NOT EXISTS rationale TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE forecast_score_records ADD COLUMN IF NOT EXISTS probability_bps INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_score_records ADD COLUMN IF NOT EXISTS outcome_value INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_score_records ADD COLUMN IF NOT EXISTS coverage_score DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE forecast_score_records ADD COLUMN IF NOT EXISTS revision_count INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE external_signal_snapshots ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS dedupe_key TEXT`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS max_attempts INTEGER NOT NULL DEFAULT 5`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
	}
	for _, stmt := range alterStmts {
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("alter forecasting schema: %w", err)
		}
	}
	if _, err := db.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_background_jobs_dedupe_key ON background_jobs(dedupe_key)`); err != nil {
		return fmt.Errorf("ensure background_jobs dedupe index: %w", err)
	}
	return nil
}
