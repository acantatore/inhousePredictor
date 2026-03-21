package market

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naranjax/inhousepredictor/internal/httpx"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts the market and its pool in a single transaction,
// debiting the creator's balance for the initial liquidity.
func (r *Repository) Create(ctx context.Context, m *Market, p *Pool) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO markets
		  (question, description, category, creator_id, resolver_id,
		   initial_liquidity, closes_at, resolves_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at
	`, m.Question, m.Description, string(m.Category),
		m.CreatorID, m.ResolverID,
		m.InitialLiquidity, m.ClosesAt, m.ResolvesAt,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert market: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO pools (market_id, yes_reserve, no_reserve, total_collateral)
		VALUES ($1,$2,$3,$4)
	`, m.ID, p.YesReserve, p.NoReserve, p.TotalCollateral)
	if err != nil {
		return fmt.Errorf("insert pool: %w", err)
	}

	// Debit creator — fails atomically if balance is insufficient
	tag, err := tx.Exec(ctx, `
		UPDATE users SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
	`, m.InitialLiquidity, m.CreatorID)
	if err != nil {
		return fmt.Errorf("debit creator: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NewError(409, httpx.CodeInsufficientBalance, "You do not have enough points to create this market.", nil)
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Market, *Pool, error) {
	m := &Market{}
	p := &Pool{}
	err := r.db.QueryRow(ctx, `
		SELECT m.id, m.question, m.description, m.category,
		       m.creator_id, creator.name, m.resolver_id, resolver.name, m.status, m.outcome, m.evidence_url,
		       m.initial_liquidity, m.closes_at, m.resolves_at, m.created_at,
		       m.resolved_at, m.dispute_deadline,
		       p.yes_reserve, p.no_reserve, p.total_collateral
		FROM markets m
		JOIN users creator ON creator.id = m.creator_id
		JOIN users resolver ON resolver.id = m.resolver_id
		JOIN pools p ON p.market_id = m.id
		WHERE m.id = $1
	`, id).Scan(
		&m.ID, &m.Question, &m.Description, &m.Category,
		&m.CreatorID, &m.CreatorName, &m.ResolverID, &m.ResolverName, &m.Status, &m.Outcome, &m.EvidenceURL,
		&m.InitialLiquidity, &m.ClosesAt, &m.ResolvesAt, &m.CreatedAt,
		&m.ResolvedAt, &m.DisputeDeadline,
		&p.YesReserve, &p.NoReserve, &p.TotalCollateral,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get market: %w", err)
	}
	p.MarketID = m.ID
	m.YesPrice = p.NoReserve / (p.YesReserve + p.NoReserve)
	m.NoPrice = p.YesReserve / (p.YesReserve + p.NoReserve)
	return m, p, nil
}

func (r *Repository) List(ctx context.Context, category Category, status Status, limit int) ([]*Market, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.question, m.description, m.category,
		       m.creator_id, creator.name, m.resolver_id, resolver.name, m.status, m.outcome, m.evidence_url,
		       m.initial_liquidity, m.closes_at, m.resolves_at, m.created_at,
		       m.resolved_at, m.dispute_deadline,
		       p.yes_reserve, p.no_reserve
		FROM markets m
		JOIN users creator ON creator.id = m.creator_id
		JOIN users resolver ON resolver.id = m.resolver_id
		JOIN pools p ON p.market_id = m.id
		WHERE ($1 = '' OR m.category = $1::market_category)
		  AND ($2 = '' OR m.status   = $2::market_status)
		ORDER BY m.created_at DESC
		LIMIT $3
	`, string(category), string(status), limit)
	if err != nil {
		return nil, fmt.Errorf("list markets: %w", err)
	}
	defer rows.Close()

	var markets []*Market
	for rows.Next() {
		m := &Market{}
		var yr, nr float64
		if err := rows.Scan(
			&m.ID, &m.Question, &m.Description, &m.Category,
			&m.CreatorID, &m.CreatorName, &m.ResolverID, &m.ResolverName, &m.Status, &m.Outcome, &m.EvidenceURL,
			&m.InitialLiquidity, &m.ClosesAt, &m.ResolvesAt, &m.CreatedAt,
			&m.ResolvedAt, &m.DisputeDeadline,
			&yr, &nr,
		); err != nil {
			return nil, err
		}
		m.YesPrice = nr / (yr + nr)
		m.NoPrice = yr / (yr + nr)
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

func (r *Repository) ListDisputes(ctx context.Context) ([]*DisputeRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.market_id, m.question, d.reason, d.created_at,
		       m.creator_id, creator.name, m.resolver_id, resolver.name,
		       m.status, m.outcome, m.evidence_url, d.resolved_by, d.resolved_at,
		       m.dispute_deadline, m.initial_liquidity
		FROM disputes d
		JOIN markets m ON m.id = d.market_id
		JOIN users creator ON creator.id = m.creator_id
		JOIN users resolver ON resolver.id = m.resolver_id
		WHERE d.resolved_at IS NULL
		ORDER BY d.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list disputes: %w", err)
	}
	defer rows.Close()

	var disputes []*DisputeRecord
	for rows.Next() {
		record := &DisputeRecord{}
		if err := rows.Scan(
			&record.DisputeID, &record.MarketID, &record.MarketQuestion, &record.Reason, &record.CreatedAt,
			&record.CreatorID, &record.CreatorName, &record.ResolverID, &record.ResolverName,
			&record.Status, &record.Outcome, &record.EvidenceURL, &record.ResolvedBy, &record.ResolvedAt,
			&record.DisputeDeadline, &record.InitialLiquidity,
		); err != nil {
			return nil, fmt.Errorf("scan dispute: %w", err)
		}
		disputes = append(disputes, record)
	}
	return disputes, rows.Err()
}

type ResolveParams struct {
	MarketID    uuid.UUID
	ResolverID  uuid.UUID
	Outcome     Outcome
	EvidenceURL string
}

func (r *Repository) Resolve(ctx context.Context, p ResolveParams) error {
	deadline := time.Now().Add(48 * time.Hour)
	tag, err := r.db.Exec(ctx, `
		UPDATE markets
		SET status = 'resolved', outcome = $1, evidence_url = $2,
		    resolved_at = NOW(), dispute_deadline = $3
		WHERE id = $4 AND resolver_id = $5 AND status = 'open'
	`, string(p.Outcome), p.EvidenceURL, deadline, p.MarketID, p.ResolverID)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NewError(409, httpx.CodeInvalidState, "Market not found, wrong resolver, or not open.", nil)
	}
	return nil
}

func (r *Repository) Dispute(ctx context.Context, marketID, userID uuid.UUID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dispute tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status Status
	var deadline *time.Time
	if err := tx.QueryRow(ctx, `SELECT status, dispute_deadline FROM markets WHERE id = $1 FOR UPDATE`, marketID).Scan(&status, &deadline); err != nil {
		return httpx.NewError(404, httpx.CodeNotFound, "Market not found.", err)
	}
	if status != StatusResolved {
		return httpx.NewError(409, httpx.CodeInvalidState, "This market cannot be disputed right now.", nil)
	}
	if deadline == nil || !deadline.After(time.Now()) {
		return httpx.NewError(409, httpx.CodeDisputeClosed, "This dispute window has closed.", nil)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO disputes (market_id, user_id, reason)
		VALUES ($1, $2, $3)
	`, marketID, userID, reason)
	if err != nil {
		return fmt.Errorf("file dispute: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE markets SET status = 'disputed'
		WHERE id = $1
	`, marketID)
	if err != nil {
		return fmt.Errorf("mark disputed: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *Repository) ReviewDispute(ctx context.Context, marketID, adminID uuid.UUID, action DisputeAction, outcome *Outcome, evidenceURL *string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dispute review tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status Status
	if err := tx.QueryRow(ctx, `SELECT status FROM markets WHERE id = $1 FOR UPDATE`, marketID).Scan(&status); err != nil {
		return httpx.NewError(404, httpx.CodeNotFound, "Market not found.", err)
	}
	if status != StatusDisputed {
		return httpx.NewError(409, httpx.CodeInvalidState, "This market is not awaiting dispute review.", nil)
	}

	switch action {
	case DisputeActionConfirmOriginal:
		_, err = tx.Exec(ctx, `UPDATE markets SET status = 'resolved' WHERE id = $1`, marketID)
	case DisputeActionOverrideOutcome:
		if outcome == nil || evidenceURL == nil {
			return httpx.NewError(400, httpx.CodeValidation, "Override outcome requires a new outcome and evidence link.", nil)
		}
		_, err = tx.Exec(ctx, `UPDATE markets SET status = 'resolved', outcome = $1, evidence_url = $2, payout_at = NULL WHERE id = $3`, string(*outcome), *evidenceURL, marketID)
	case DisputeActionCancelMarket:
		cancelled := OutcomeCancelled
		_, err = tx.Exec(ctx, `UPDATE markets SET status = 'cancelled', outcome = $1, payout_at = NULL WHERE id = $2`, string(cancelled), marketID)
	default:
		return httpx.NewError(400, httpx.CodeValidation, "Invalid dispute action.", nil)
	}
	if err != nil {
		return fmt.Errorf("apply dispute action: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE disputes SET resolved_by = $1, resolved_at = NOW() WHERE market_id = $2 AND resolved_at IS NULL`, adminID, marketID); err != nil {
		return fmt.Errorf("mark disputes reviewed: %w", err)
	}
	return tx.Commit(ctx)
}
