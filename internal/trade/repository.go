package trade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naranjax/inhousepredictor/internal/cpmm"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type executeParams struct {
	UserID   uuid.UUID
	MarketID uuid.UUID
	Side     Side
	Cost     int64
}

type executeResult struct {
	Trade   *Trade
	NewPool cpmm.Pool
}

// execute runs a full trade atomically at serializable isolation:
// lock pool → validate market open → compute CPMM → debit user → update pool → record trade → upsert position.
func (r *Repository) execute(ctx context.Context, p executeParams) (*executeResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Read + lock pool to prevent concurrent trades on same market
	var pool cpmm.Pool
	err = tx.QueryRow(ctx, `
		SELECT yes_reserve, no_reserve, k
		FROM pools WHERE market_id = $1 FOR UPDATE
	`, p.MarketID).Scan(&pool.YesReserve, &pool.NoReserve, &pool.K)
	if err != nil {
		return nil, fmt.Errorf("get pool: %w", err)
	}

	var status string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM markets WHERE id = $1`, p.MarketID,
	).Scan(&status); err != nil || status != "open" {
		return nil, fmt.Errorf("market is not open")
	}

	priceBefore := pool.YesPrice()

	var shares float64
	var newPool cpmm.Pool
	if p.Side == SideYes {
		shares, newPool, err = pool.BuyYes(p.Cost)
	} else {
		shares, newPool, err = pool.BuyNo(p.Cost)
	}
	if err != nil {
		return nil, err
	}

	priceAfter := newPool.YesPrice()

	// Debit user balance atomically
	tag, err := tx.Exec(ctx, `
		UPDATE users SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
	`, p.Cost, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("debit user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Update pool state
	_, err = tx.Exec(ctx, `
		UPDATE pools
		SET yes_reserve = $1, no_reserve = $2, k = $3,
		    total_collateral = total_collateral + $4, updated_at = NOW()
		WHERE market_id = $5
	`, newPool.YesReserve, newPool.NoReserve, newPool.K, p.Cost, p.MarketID)
	if err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}

	// Record immutable trade
	t := &Trade{
		UserID: p.UserID, MarketID: p.MarketID,
		Side: p.Side, Shares: shares, Cost: p.Cost,
		YesPriceBefore: priceBefore, YesPriceAfter: priceAfter,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO trades
		  (user_id, market_id, side, shares, cost, yes_price_before, yes_price_after)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at
	`, t.UserID, t.MarketID, string(t.Side), t.Shares, t.Cost,
		t.YesPriceBefore, t.YesPriceAfter,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert trade: %w", err)
	}

	// Upsert position
	if p.Side == SideYes {
		_, err = tx.Exec(ctx, `
			INSERT INTO positions (user_id, market_id, yes_shares, no_shares)
			VALUES ($1,$2,$3,0)
			ON CONFLICT (user_id, market_id)
			DO UPDATE SET yes_shares = positions.yes_shares + $3
		`, p.UserID, p.MarketID, shares)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO positions (user_id, market_id, yes_shares, no_shares)
			VALUES ($1,$2,0,$3)
			ON CONFLICT (user_id, market_id)
			DO UPDATE SET no_shares = positions.no_shares + $3
		`, p.UserID, p.MarketID, shares)
	}
	if err != nil {
		return nil, fmt.Errorf("upsert position: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &executeResult{Trade: t, NewPool: newPool}, nil
}

func (r *Repository) GetPositions(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, market_id, yes_shares, no_shares
		FROM positions
		WHERE user_id = $1 AND (yes_shares > 0 OR no_shares > 0)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Position
	for rows.Next() {
		pos := &Position{}
		if err := rows.Scan(&pos.UserID, &pos.MarketID, &pos.YesShares, &pos.NoShares); err != nil {
			return nil, err
		}
		out = append(out, pos)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketTrades(ctx context.Context, marketID uuid.UUID, limit int) ([]*Trade, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, market_id, side, shares, cost,
		       yes_price_before, yes_price_after, created_at
		FROM trades WHERE market_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, marketID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Trade
	for rows.Next() {
		t := &Trade{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.MarketID, &t.Side,
			&t.Shares, &t.Cost, &t.YesPriceBefore, &t.YesPriceAfter, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Payout distributes total market collateral to winning share holders.
// Payout per share = total_collateral / total_winning_shares
func (r *Repository) Payout(ctx context.Context, marketID uuid.UUID, outcome Side) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var totalCollateral int64
	if err := tx.QueryRow(ctx,
		`SELECT total_collateral FROM pools WHERE market_id = $1`, marketID,
	).Scan(&totalCollateral); err != nil {
		return err
	}

	col := "yes_shares"
	if outcome == SideNo {
		col = "no_shares"
	}

	var totalWinning float64
	if err := tx.QueryRow(ctx, fmt.Sprintf(
		`SELECT COALESCE(SUM(%s), 0) FROM positions WHERE market_id = $1`, col,
	), marketID).Scan(&totalWinning); err != nil || totalWinning == 0 {
		return tx.Commit(ctx) // no winners
	}

	payoutPerShare := float64(totalCollateral) / totalWinning

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE users u
		SET balance = balance + FLOOR(p.%s * $1)::BIGINT
		FROM positions p
		WHERE p.market_id = $2 AND p.user_id = u.id AND p.%s > 0
	`, col, col), payoutPerShare, marketID)
	if err != nil {
		return fmt.Errorf("payout: %w", err)
	}

	return tx.Commit(ctx)
}
