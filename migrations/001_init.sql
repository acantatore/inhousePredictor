CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE market_status   AS ENUM ('open', 'closed', 'resolved', 'disputed', 'cancelled');
CREATE TYPE market_outcome  AS ENUM ('yes', 'no', 'cancelled');
CREATE TYPE trade_side      AS ENUM ('yes', 'no');
CREATE TYPE market_category AS ENUM ('people', 'okrs', 'slas', 'financials', 'general');

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
    dispute_deadline  TIMESTAMPTZ
);

-- AMM pool: yes_reserve * no_reserve = k (constant product invariant)
CREATE TABLE pools (
    market_id        UUID             PRIMARY KEY REFERENCES markets(id) ON DELETE CASCADE,
    yes_reserve      DOUBLE PRECISION NOT NULL,
    no_reserve       DOUBLE PRECISION NOT NULL,
    k                DOUBLE PRECISION NOT NULL,
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
