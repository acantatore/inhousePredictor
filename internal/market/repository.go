package market

import (
	"context"
	"encoding/json"
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
		       m.resolved_at, m.dispute_deadline, m.is_shadow, m.forecast_question_id,
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
		&m.ResolvedAt, &m.DisputeDeadline, &m.IsShadow, &m.ForecastQuestionID,
		&p.YesReserve, &p.NoReserve, &p.TotalCollateral,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get market: %w", err)
	}
	p.MarketID = m.ID
	m.YesPrice = p.NoReserve / (p.YesReserve + p.NoReserve)
	m.NoPrice = p.YesReserve / (p.YesReserve + p.NoReserve)
	options, err := r.GetOptions(ctx, m.ID)
	if err == nil && len(options) > 0 {
		m.Options = options
		applyOptionSummary(m)
	}
	if snapshots, err := r.ListSnapshots(ctx, m.ID, 60); err == nil {
		m.Snapshots = snapshots
		applySentiment(m)
	}
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
		       m.resolved_at, m.dispute_deadline, m.is_shadow, m.forecast_question_id,
		       p.yes_reserve, p.no_reserve
		FROM markets m
		JOIN users creator ON creator.id = m.creator_id
		JOIN users resolver ON resolver.id = m.resolver_id
		JOIN pools p ON p.market_id = m.id
		WHERE m.is_shadow = false
		  AND ($1 = '' OR m.category = NULLIF($1, '')::market_category)
		  AND ($2 = '' OR m.status   = NULLIF($2, '')::market_status)
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
			&m.ResolvedAt, &m.DisputeDeadline, &m.IsShadow, &m.ForecastQuestionID,
			&yr, &nr,
		); err != nil {
			return nil, err
		}
		m.YesPrice = nr / (yr + nr)
		m.NoPrice = yr / (yr + nr)
		if options, err := r.GetOptions(ctx, m.ID); err == nil && len(options) > 0 {
			m.Options = options
			applyOptionSummary(m)
		}
		if snapshots, err := r.ListSnapshots(ctx, m.ID, 2); err == nil {
			m.Snapshots = snapshots
			applySentiment(m)
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

func (r *Repository) GetOptions(ctx context.Context, marketID uuid.UUID) ([]Option, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, market_id, label, sort_order, collateral, is_winner
		FROM market_options
		WHERE market_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, marketID)
	if err != nil {
		return nil, fmt.Errorf("get market options: %w", err)
	}
	defer rows.Close()
	var opts []Option
	var total int64
	for rows.Next() {
		var option Option
		if err := rows.Scan(&option.ID, &option.MarketID, &option.Label, &option.SortOrder, &option.Collateral, &option.IsWinner); err != nil {
			return nil, fmt.Errorf("scan market option: %w", err)
		}
		total += option.Collateral
		opts = append(opts, option)
	}
	for i := range opts {
		if total > 0 {
			opts[i].ProbabilityBps = int((float64(opts[i].Collateral) / float64(total)) * 10000)
		}
	}
	return opts, rows.Err()
}

func (r *Repository) ListSnapshots(ctx context.Context, marketID uuid.UUID, limit int) ([]Snapshot, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.db.Query(ctx, `SELECT id, market_id, points, captured_at FROM market_probability_snapshots WHERE market_id = $1 ORDER BY captured_at ASC LIMIT $2`, marketID, limit)
	if err != nil {
		return nil, fmt.Errorf("list probability snapshots: %w", err)
	}
	defer rows.Close()
	var snapshots []Snapshot
	for rows.Next() {
		var snapshot Snapshot
		var raw []byte
		if err := rows.Scan(&snapshot.ID, &snapshot.MarketID, &raw, &snapshot.CapturedAt); err != nil {
			return nil, fmt.Errorf("scan probability snapshot: %w", err)
		}
		if err := json.Unmarshal(raw, &snapshot.Points); err != nil {
			return nil, fmt.Errorf("decode probability snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func applyOptionSummary(m *Market) {
	if len(m.Options) == 0 {
		return
	}
	if len(m.Options) >= 1 {
		m.YesPrice = float64(m.Options[0].ProbabilityBps) / 10000
	}
	if len(m.Options) >= 2 {
		m.NoPrice = float64(m.Options[1].ProbabilityBps) / 10000
	}
}

func applySentiment(m *Market) {
	if len(m.Snapshots) < 2 {
		m.Sentiment = "flat"
		m.ChangeBps = 0
		return
	}
	last := maxSnapshotProbability(m.Snapshots[len(m.Snapshots)-1].Points)
	prev := maxSnapshotProbability(m.Snapshots[len(m.Snapshots)-2].Points)
	change := last - prev
	m.ChangeBps = change
	switch {
	case change > 25:
		m.Sentiment = "up"
	case change < -25:
		m.Sentiment = "down"
	default:
		m.Sentiment = "flat"
	}
}

func maxSnapshotProbability(points map[string]int) int {
	var total int
	for _, value := range points {
		total += value
	}
	if total == 0 {
		return 0
	}
	max := 0
	for _, value := range points {
		bps := int((float64(value) / float64(total)) * 10000)
		if bps > max {
			max = bps
		}
	}
	return max
}

func (r *Repository) CreateShadow(ctx context.Context, m *Market, p *Pool) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO markets
		  (question, description, category, creator_id, resolver_id, initial_liquidity, closes_at, resolves_at, is_shadow, forecast_question_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true,$9)
		RETURNING id, created_at
	`, m.Question, m.Description, string(m.Category), m.CreatorID, m.ResolverID, m.InitialLiquidity, m.ClosesAt, m.ResolvesAt, m.ForecastQuestionID).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert shadow market: %w", err)
	}
	if _, err := r.db.Exec(ctx, `
		INSERT INTO pools (market_id, yes_reserve, no_reserve, total_collateral)
		VALUES ($1,$2,$3,$4)
	`, m.ID, p.YesReserve, p.NoReserve, p.TotalCollateral); err != nil {
		return fmt.Errorf("insert shadow pool: %w", err)
	}
	return nil
}

func (r *Repository) UpsertOptions(ctx context.Context, marketID uuid.UUID, options []Option) error {
	for _, option := range options {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO market_options (market_id, label, sort_order, collateral, is_winner)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (market_id, label)
			DO UPDATE SET sort_order = EXCLUDED.sort_order, collateral = EXCLUDED.collateral, is_winner = EXCLUDED.is_winner
		`, marketID, option.Label, option.SortOrder, option.Collateral, option.IsWinner); err != nil {
			return fmt.Errorf("upsert market option: %w", err)
		}
	}
	return nil
}

func (r *Repository) RecordSnapshot(ctx context.Context, marketID uuid.UUID) error {
	options, err := r.GetOptions(ctx, marketID)
	if err != nil {
		return err
	}
	points := map[string]int{}
	for _, option := range options {
		points[option.Label] = int(option.Collateral)
	}
	raw, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("marshal snapshot points: %w", err)
	}
	if _, err := r.db.Exec(ctx, `INSERT INTO market_probability_snapshots (market_id, points) VALUES ($1, $2)`, marketID, raw); err != nil {
		return fmt.Errorf("insert probability snapshot: %w", err)
	}
	return nil
}

func (r *Repository) SyncShadowLifecycle(ctx context.Context, forecastQuestionID uuid.UUID, status Status, outcome *Outcome, evidenceURL *string, closesAt, resolvesAt time.Time, resolvedAt *time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE markets
		SET status = $1,
		    outcome = $2,
		    evidence_url = $3,
		    closes_at = $4,
		    resolves_at = $5,
		    resolved_at = $6
		WHERE forecast_question_id = $7 AND is_shadow = true
	`, string(status), outcome, evidenceURL, closesAt, resolvesAt, resolvedAt, forecastQuestionID)
	if err != nil {
		return fmt.Errorf("sync shadow lifecycle: %w", err)
	}
	return nil
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
	MarketID        uuid.UUID
	ResolverID      uuid.UUID
	Outcome         Outcome
	WinningOptionID *uuid.UUID
	EvidenceURL     string
}

func (r *Repository) Resolve(ctx context.Context, p ResolveParams) error {
	deadline := time.Now().Add(48 * time.Hour)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin resolve tx: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
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
	if _, err := tx.Exec(ctx, `UPDATE market_options SET is_winner = false WHERE market_id = $1`, p.MarketID); err != nil {
		return fmt.Errorf("clear winning options: %w", err)
	}
	if p.WinningOptionID != nil {
		if _, err := tx.Exec(ctx, `UPDATE market_options SET is_winner = true WHERE id = $1 AND market_id = $2`, *p.WinningOptionID, p.MarketID); err != nil {
			return fmt.Errorf("set winning option: %w", err)
		}
	}
	return tx.Commit(ctx)
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
