CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE market_status   AS ENUM ('open', 'closed', 'resolved', 'disputed', 'cancelled');
CREATE TYPE market_outcome  AS ENUM ('yes', 'no', 'cancelled');
CREATE TYPE trade_side      AS ENUM ('yes', 'no');
CREATE TYPE market_category AS ENUM ('people', 'okrs', 'slas', 'financials', 'general');
CREATE TYPE forecast_question_status AS ENUM ('open', 'closed', 'resolved', 'cancelled');
CREATE TYPE forecast_question_outcome AS ENUM ('delivered', 'not_delivered', 'cancelled');

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    balance       BIGINT      NOT NULL DEFAULT 10000, -- play money points
    is_admin      BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE markets (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    question          TEXT            NOT NULL,
    description       TEXT            NOT NULL DEFAULT '',
    category          market_category NOT NULL,
    creator_id        UUID            NOT NULL REFERENCES users(id),
    resolver_id       UUID            NOT NULL REFERENCES users(id),
    status            market_status   NOT NULL DEFAULT 'open',
    outcome           market_outcome,
    evidence_url      TEXT,
    initial_liquidity BIGINT          NOT NULL,
    closes_at         TIMESTAMPTZ     NOT NULL,
    resolves_at       TIMESTAMPTZ     NOT NULL,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    resolved_at       TIMESTAMPTZ,
    dispute_deadline  TIMESTAMPTZ,
    payout_at         TIMESTAMPTZ,
    is_shadow         BOOLEAN         NOT NULL DEFAULT false,
    forecast_question_id UUID
);

-- AMM pool: constant-product invariant is computed from yes_reserve * no_reserve
CREATE TABLE pools (
    market_id        UUID             PRIMARY KEY REFERENCES markets(id) ON DELETE CASCADE,
    yes_reserve      DOUBLE PRECISION NOT NULL,
    no_reserve       DOUBLE PRECISION NOT NULL,
    total_collateral BIGINT           NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- User share holdings per market
CREATE TABLE positions (
    user_id    UUID             NOT NULL REFERENCES users(id),
    market_id  UUID             NOT NULL REFERENCES markets(id),
    yes_shares DOUBLE PRECISION NOT NULL DEFAULT 0,
    no_shares  DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, market_id)
);

-- Immutable trade ledger
CREATE TABLE trades (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID             NOT NULL REFERENCES users(id),
    market_id        UUID             NOT NULL REFERENCES markets(id),
    side             trade_side       NOT NULL,
    shares           DOUBLE PRECISION NOT NULL,
    cost             BIGINT           NOT NULL, -- play money points spent
    yes_price_before DOUBLE PRECISION NOT NULL,
    yes_price_after  DOUBLE PRECISION NOT NULL,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE disputes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id   UUID        NOT NULL REFERENCES markets(id),
    user_id     UUID        NOT NULL REFERENCES users(id),
    reason      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID        REFERENCES users(id),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_markets_status   ON markets(status);
CREATE INDEX idx_markets_category ON markets(category);
CREATE INDEX idx_trades_market    ON trades(market_id, created_at DESC);
CREATE INDEX idx_trades_user      ON trades(user_id);
CREATE INDEX idx_positions_user   ON positions(user_id);
CREATE INDEX idx_positions_market ON positions(market_id);

CREATE TABLE market_options (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id      UUID        NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    label          TEXT        NOT NULL,
    sort_order     INTEGER     NOT NULL DEFAULT 0,
    collateral     BIGINT      NOT NULL DEFAULT 0,
    is_winner      BOOLEAN     NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (market_id, label)
);

CREATE TABLE option_positions (
    user_id         UUID             NOT NULL REFERENCES users(id),
    market_option_id UUID            NOT NULL REFERENCES market_options(id) ON DELETE CASCADE,
    shares          DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, market_option_id)
);

CREATE TABLE option_trades (
    id                  UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID             NOT NULL REFERENCES users(id),
    market_id           UUID             NOT NULL REFERENCES markets(id),
    market_option_id    UUID             NOT NULL REFERENCES market_options(id) ON DELETE CASCADE,
    shares              DOUBLE PRECISION NOT NULL,
    cost                BIGINT           NOT NULL,
    probability_before  DOUBLE PRECISION NOT NULL,
    probability_after   DOUBLE PRECISION NOT NULL,
    created_at          TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE market_probability_snapshots (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id    UUID       NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    points      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_market_options_market ON market_options(market_id, sort_order);
CREATE INDEX idx_option_positions_user ON option_positions(user_id);
CREATE INDEX idx_option_positions_option ON option_positions(market_option_id);
CREATE INDEX idx_option_trades_market ON option_trades(market_id, created_at DESC);
CREATE INDEX idx_probability_snapshots_market ON market_probability_snapshots(market_id, captured_at DESC);

CREATE TABLE forecast_questions (
    id                UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    title             TEXT                     NOT NULL,
    description       TEXT                     NOT NULL DEFAULT '',
    program           TEXT                     NOT NULL,
    owner_id          UUID                     NOT NULL REFERENCES users(id),
    resolver_id       UUID                     NOT NULL REFERENCES users(id),
    status            forecast_question_status NOT NULL DEFAULT 'open',
    outcome           forecast_question_outcome,
    resolution_rule   TEXT                     NOT NULL,
    rationale_policy_threshold INTEGER         NOT NULL DEFAULT 10,
    linked_market_id  UUID                     REFERENCES markets(id),
    closes_at         TIMESTAMPTZ              NOT NULL,
    resolves_at       TIMESTAMPTZ              NOT NULL,
    resolved_at       TIMESTAMPTZ,
    evidence_url      TEXT,
    created_at        TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ              NOT NULL DEFAULT NOW()
);

CREATE TABLE forecast_question_contributors (
    forecast_question_id UUID NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    user_id              UUID NOT NULL REFERENCES users(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (forecast_question_id, user_id)
);

CREATE TABLE forecast_revisions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    forecast_question_id UUID       NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    user_id             UUID        NOT NULL REFERENCES users(id),
    probability_bps     INTEGER     NOT NULL,
    rationale           TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE forecast_latest_active (
    forecast_question_id UUID        NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    user_id              UUID        NOT NULL REFERENCES users(id),
    forecast_revision_id UUID        NOT NULL REFERENCES forecast_revisions(id) ON DELETE CASCADE,
    probability_bps      INTEGER     NOT NULL,
    rationale            TEXT        NOT NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (forecast_question_id, user_id)
);

CREATE TABLE forecast_projections (
    forecast_question_id UUID        PRIMARY KEY REFERENCES forecast_questions(id) ON DELETE CASCADE,
    official_probability_bps INTEGER NOT NULL DEFAULT 0,
    contributor_count    INTEGER     NOT NULL DEFAULT 0,
    last_change_bps      INTEGER     NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE forecast_snapshots (
    forecast_question_id UUID        NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    user_id              UUID        NOT NULL REFERENCES users(id),
    forecast_revision_id UUID        NOT NULL REFERENCES forecast_revisions(id) ON DELETE CASCADE,
    probability_bps      INTEGER     NOT NULL,
    rationale            TEXT        NOT NULL,
    snapshotted_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (forecast_question_id, user_id)
);

CREATE TABLE forecast_score_records (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    forecast_question_id UUID       NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    user_id             UUID        NOT NULL REFERENCES users(id),
    probability_bps     INTEGER     NOT NULL,
    outcome_value       INTEGER     NOT NULL,
    brier_score         DOUBLE PRECISION NOT NULL,
    coverage_score      DOUBLE PRECISION NOT NULL,
    revision_count      INTEGER     NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (forecast_question_id, user_id)
);

CREATE TABLE external_signal_snapshots (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    forecast_question_id UUID       NOT NULL REFERENCES forecast_questions(id) ON DELETE CASCADE,
    source              TEXT        NOT NULL,
    probability_bps     INTEGER     NOT NULL,
    note                TEXT        NOT NULL DEFAULT '',
    captured_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE background_jobs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        TEXT        NOT NULL,
    dedupe_key  TEXT        NOT NULL UNIQUE,
    payload     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    run_at      TIMESTAMPTZ NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    attempts    INTEGER     NOT NULL DEFAULT 0,
    max_attempts INTEGER    NOT NULL DEFAULT 5,
    last_error  TEXT,
    processed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_forecast_questions_program ON forecast_questions(program);
CREATE INDEX idx_forecast_questions_status ON forecast_questions(status);
CREATE INDEX idx_forecast_revisions_question_created ON forecast_revisions(forecast_question_id, created_at DESC);
CREATE INDEX idx_forecast_latest_active_question ON forecast_latest_active(forecast_question_id);
CREATE INDEX idx_background_jobs_run_at ON background_jobs(status, run_at);
