package trade

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naranjax/inhousepredictor/internal/cpmm"
	"github.com/naranjax/inhousepredictor/internal/httpx"
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
	Trade       *Trade
	NewPool     cpmm.Pool
	UserBalance int64
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
	var snapshot MarketSnapshot
	err = tx.QueryRow(ctx, `
		SELECT m.status, m.creator_id, m.closes_at, p.yes_reserve, p.no_reserve, p.total_collateral
		FROM markets m
		JOIN pools p ON p.market_id = m.id
		WHERE m.id = $1
		FOR UPDATE OF p, m
	`, p.MarketID).Scan(&snapshot.Status, &snapshot.CreatorID, &snapshot.ClosesAt, &snapshot.YesReserve, &snapshot.NoReserve, &snapshot.TotalCollateral)
	if err != nil {
		return nil, fmt.Errorf("get pool: %w", err)
	}
	pool := cpmm.Pool{YesReserve: snapshot.YesReserve, NoReserve: snapshot.NoReserve, K: snapshot.YesReserve * snapshot.NoReserve}

	if snapshot.Status != "open" {
		return nil, httpx.NewError(409, httpx.CodeMarketClosed, "Trading is closed for this market.", nil)
	}
	if !snapshot.ClosesAt.After(nowUTC()) {
		_, _ = tx.Exec(ctx, `UPDATE markets SET status = 'closed' WHERE id = $1 AND status = 'open'`, p.MarketID)
		return nil, httpx.NewError(409, httpx.CodeMarketClosed, "Trading is closed for this market.", nil)
	}
	if snapshot.CreatorID == p.UserID {
		return nil, httpx.NewError(409, httpx.CodeCreatorRestricted, "You cannot trade a market you created.", nil)
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
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}

	priceAfter := newPool.YesPrice()

	// Debit user balance atomically
	var userBalance int64
	tag, err := tx.Exec(ctx, `
		UPDATE users SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
	`, p.Cost, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("debit user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.NewError(409, httpx.CodeInsufficientBalance, "You do not have enough points for this trade.", nil)
	}
	if err := tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1`, p.UserID).Scan(&userBalance); err != nil {
		return nil, fmt.Errorf("get user balance: %w", err)
	}

	// Update pool state
	_, err = tx.Exec(ctx, `
		UPDATE pools
		SET yes_reserve = $1, no_reserve = $2,
		    total_collateral = total_collateral + $3, updated_at = NOW()
		WHERE market_id = $4
	`, newPool.YesReserve, newPool.NoReserve, p.Cost, p.MarketID)
	if err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}

	// Record immutable trade
	t := &Trade{
		UserID: p.UserID, MarketID: p.MarketID,
		Side: p.Side, Shares: shares, Cost: p.Cost,
		UserBalance:    userBalance,
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
		if httpx.IsSerialization(err) || errors.Is(err, pgx.ErrTxCommitRollback) {
			return nil, err
		}
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &executeResult{Trade: t, NewPool: newPool, UserBalance: userBalance}, nil
}

func (r *Repository) GetPositions(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.market_id, m.question, m.status, m.outcome::text, p.yes_shares, p.no_shares
		FROM positions p
		JOIN markets m ON m.id = p.market_id
		WHERE p.user_id = $1 AND (p.yes_shares > 0 OR p.no_shares > 0)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Position
	for rows.Next() {
		pos := &Position{}
		if err := rows.Scan(&pos.UserID, &pos.MarketID, &pos.MarketQuestion, &pos.MarketStatus, &pos.MarketOutcome, &pos.YesShares, &pos.NoShares); err != nil {
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

// Payout distributes total market collateral to winning share holders or refunds on cancellation.
func (r *Repository) Payout(ctx context.Context, marketID uuid.UUID, outcome string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var totalCollateral int64
	var payoutAt any
	if err := tx.QueryRow(ctx,
		`SELECT p.total_collateral, m.payout_at FROM pools p JOIN markets m ON m.id = p.market_id WHERE p.market_id = $1 FOR UPDATE`, marketID,
	).Scan(&totalCollateral, &payoutAt); err != nil {
		return err
	}
	if payoutAt != nil {
		return tx.Commit(ctx)
	}

	if outcome == "cancelled" {
		if _, err := tx.Exec(ctx, `
			UPDATE users u
			SET balance = balance + refunds.amount
			FROM (
				SELECT user_id, SUM(cost)::BIGINT AS amount
				FROM trades
				WHERE market_id = $1
				GROUP BY user_id
				UNION ALL
				SELECT creator_id AS user_id, initial_liquidity AS amount
				FROM markets WHERE id = $1
			) refunds
			WHERE u.id = refunds.user_id
		`, marketID); err != nil {
			return fmt.Errorf("refund cancelled market: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE markets SET payout_at = NOW() WHERE id = $1 AND payout_at IS NULL`, marketID); err != nil {
			return fmt.Errorf("mark cancelled payout complete: %w", err)
		}
		return tx.Commit(ctx)
	}

	col := "yes_shares"
	if outcome == string(SideNo) {
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
	if _, err := tx.Exec(ctx, `UPDATE markets SET payout_at = NOW() WHERE id = $1 AND payout_at IS NULL`, marketID); err != nil {
		return fmt.Errorf("mark payout complete: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) ReadyForPayout(ctx context.Context) ([]uuid.UUID, []string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, outcome::text
		FROM markets
		WHERE status = 'resolved'
		  AND payout_at IS NULL
		  AND dispute_deadline IS NOT NULL
		  AND dispute_deadline <= NOW()
		ORDER BY resolved_at ASC
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("list ready payouts: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	var outcomes []string
	for rows.Next() {
		var id uuid.UUID
		var outcome string
		if err := rows.Scan(&id, &outcome); err != nil {
			return nil, nil, fmt.Errorf("scan ready payout: %w", err)
		}
		ids = append(ids, id)
		outcomes = append(outcomes, outcome)
	}
	return ids, outcomes, rows.Err()
}

func nowUTC() time.Time { return time.Now().UTC() }
