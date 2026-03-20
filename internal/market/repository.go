package market

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
		INSERT INTO pools (market_id, yes_reserve, no_reserve, k, total_collateral)
		VALUES ($1,$2,$3,$4,$5)
	`, m.ID, p.YesReserve, p.NoReserve, p.K, p.TotalCollateral)
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
		return fmt.Errorf("insufficient balance")
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Market, *Pool, error) {
	m := &Market{}
	p := &Pool{}
	err := r.db.QueryRow(ctx, `
		SELECT m.id, m.question, m.description, m.category,
		       m.creator_id, m.resolver_id, m.status, m.outcome, m.evidence_url,
		       m.initial_liquidity, m.closes_at, m.resolves_at, m.created_at,
		       m.resolved_at, m.dispute_deadline,
		       p.yes_reserve, p.no_reserve, p.k, p.total_collateral
		FROM markets m
		JOIN pools p ON p.market_id = m.id
		WHERE m.id = $1
	`, id).Scan(
		&m.ID, &m.Question, &m.Description, &m.Category,
		&m.CreatorID, &m.ResolverID, &m.Status, &m.Outcome, &m.EvidenceURL,
		&m.InitialLiquidity, &m.ClosesAt, &m.ResolvesAt, &m.CreatedAt,
		&m.ResolvedAt, &m.DisputeDeadline,
		&p.YesReserve, &p.NoReserve, &p.K, &p.TotalCollateral,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get market: %w", err)
	}
	p.MarketID = m.ID
	m.YesPrice = p.NoReserve / (p.YesReserve + p.NoReserve)
	m.NoPrice = p.YesReserve / (p.YesReserve + p.NoReserve)
	return m, p, nil
}

func (r *Repository) List(ctx context.Context, category Category, status Status) ([]*Market, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.question, m.description, m.category,
		       m.creator_id, m.resolver_id, m.status, m.outcome, m.evidence_url,
		       m.initial_liquidity, m.closes_at, m.resolves_at, m.created_at,
		       m.resolved_at, m.dispute_deadline,
		       p.yes_reserve, p.no_reserve
		FROM markets m
		JOIN pools p ON p.market_id = m.id
		WHERE ($1 = '' OR m.category = $1::market_category)
		  AND ($2 = '' OR m.status   = $2::market_status)
		ORDER BY m.created_at DESC
	`, string(category), string(status))
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
			&m.CreatorID, &m.ResolverID, &m.Status, &m.Outcome, &m.EvidenceURL,
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
		return fmt.Errorf("market not found, wrong resolver, or not open")
	}
	return nil
}

func (r *Repository) Dispute(ctx context.Context, marketID, userID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO disputes (market_id, user_id, reason)
		VALUES ($1, $2, $3)
	`, marketID, userID, reason)
	if err != nil {
		return fmt.Errorf("file dispute: %w", err)
	}
	// Mark market as disputed
	_, err = r.db.Exec(ctx, `
		UPDATE markets SET status = 'disputed'
		WHERE id = $1 AND status = 'resolved'
		  AND dispute_deadline > NOW()
	`, marketID)
	return err
}
