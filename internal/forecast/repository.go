package forecast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/user"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) CreateQuestion(ctx context.Context, q *Question, contributors []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create question tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO forecast_questions
		  (title, description, program, owner_id, resolver_id, resolution_rule, rationale_policy_threshold, closes_at, resolves_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at
	`, q.Title, q.Description, q.Program, q.OwnerID, q.ResolverID, q.ResolutionRule, q.RationalePolicyThreshold, q.ClosesAt, q.ResolvesAt).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert forecast question: %w", err)
	}

	for _, contributorID := range contributors {
		if _, err := tx.Exec(ctx, `
			INSERT INTO forecast_question_contributors (forecast_question_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, q.ID, contributorID); err != nil {
			return fmt.Errorf("insert contributor: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO forecast_projections (forecast_question_id)
		VALUES ($1)
		ON CONFLICT DO NOTHING
	`, q.ID); err != nil {
		return fmt.Errorf("seed projection: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) AttachMarket(ctx context.Context, questionID, marketID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE forecast_questions SET linked_market_id = $1, updated_at = NOW() WHERE id = $2`, marketID, questionID)
	if err != nil {
		return fmt.Errorf("attach market: %w", err)
	}
	return nil
}

func (r *Repository) ListQuestions(ctx context.Context, program string, status QuestionStatus) ([]*Question, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fq.id, fq.title, fq.description, fq.program,
		       fq.owner_id, owner.name, fq.resolver_id, resolver.name,
		       fq.status, fq.outcome, fq.resolution_rule, fq.rationale_policy_threshold,
		       fq.linked_market_id, fq.closes_at, fq.resolves_at, fq.resolved_at,
		       fq.evidence_url, fq.created_at, fq.updated_at,
		       COALESCE(mp.yes_reserve, 0), COALESCE(mp.no_reserve, 0),
		       COALESCE(fp.official_probability_bps, 0), COALESCE(fp.current_risk_bps, 0), COALESCE(fp.contributor_count, 0), COALESCE(fp.last_change_bps, 0), COALESCE(fp.updated_at, fq.updated_at)
		FROM forecast_questions fq
		JOIN users owner ON owner.id = fq.owner_id
		JOIN users resolver ON resolver.id = fq.resolver_id
		LEFT JOIN pools mp ON mp.market_id = fq.linked_market_id
		LEFT JOIN forecast_projections fp ON fp.forecast_question_id = fq.id
		WHERE ($1 = '' OR fq.program = $1)
		  AND ($2 = '' OR fq.status = NULLIF($2, '')::forecast_question_status)
		ORDER BY fq.created_at DESC
	`, program, string(status))
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	var questions []*Question
	for rows.Next() {
		q := &Question{Projection: &Projection{}}
		var marketYesReserve, marketNoReserve float64
		if err := rows.Scan(
			&q.ID, &q.Title, &q.Description, &q.Program,
			&q.OwnerID, &q.OwnerName, &q.ResolverID, &q.ResolverName,
			&q.Status, &q.Outcome, &q.ResolutionRule, &q.RationalePolicyThreshold,
			&q.LinkedMarketID, &q.ClosesAt, &q.ResolvesAt, &q.ResolvedAt,
			&q.EvidenceURL, &q.CreatedAt, &q.UpdatedAt,
			&marketYesReserve, &marketNoReserve,
			&q.Projection.OfficialProbabilityBps, &q.Projection.CurrentRiskBps, &q.Projection.ContributorCount, &q.Projection.LastChangeBps, &q.Projection.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		q.Projection.ForecastQuestionID = q.ID
		if total := marketYesReserve + marketNoReserve; total > 0 {
			q.LinkedMarketYesPrice = marketNoReserve / total
			q.LinkedMarketNoPrice = marketYesReserve / total
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (r *Repository) GetQuestion(ctx context.Context, questionID uuid.UUID) (*Question, error) {
	q := &Question{Projection: &Projection{}}
	var outcomeText *string
	var linkedMarketID *uuid.UUID
	var resolvedAt *time.Time
	var evidenceURL *string
	var marketYesReserve, marketNoReserve float64
	err := r.db.QueryRow(ctx, `
		SELECT fq.id, fq.title, fq.description, fq.program,
		       fq.owner_id, owner.name, fq.resolver_id, resolver.name,
		       fq.status, fq.outcome, fq.resolution_rule, fq.rationale_policy_threshold,
		       fq.linked_market_id, fq.closes_at, fq.resolves_at, fq.resolved_at,
		       fq.evidence_url, fq.created_at, fq.updated_at,
		       COALESCE(mp.yes_reserve, 0), COALESCE(mp.no_reserve, 0),
		       COALESCE(fp.official_probability_bps, 0), COALESCE(fp.current_risk_bps, 0), COALESCE(fp.contributor_count, 0), COALESCE(fp.last_change_bps, 0), COALESCE(fp.updated_at, fq.updated_at)
		FROM forecast_questions fq
		JOIN users owner ON owner.id = fq.owner_id
		JOIN users resolver ON resolver.id = fq.resolver_id
		LEFT JOIN pools mp ON mp.market_id = fq.linked_market_id
		LEFT JOIN forecast_projections fp ON fp.forecast_question_id = fq.id
		WHERE fq.id = $1
	`, questionID).Scan(
		&q.ID, &q.Title, &q.Description, &q.Program,
		&q.OwnerID, &q.OwnerName, &q.ResolverID, &q.ResolverName,
		&q.Status, &outcomeText, &q.ResolutionRule, &q.RationalePolicyThreshold,
		&linkedMarketID, &q.ClosesAt, &q.ResolvesAt, &resolvedAt,
		&evidenceURL, &q.CreatedAt, &q.UpdatedAt,
		&marketYesReserve, &marketNoReserve,
		&q.Projection.OfficialProbabilityBps, &q.Projection.CurrentRiskBps, &q.Projection.ContributorCount, &q.Projection.LastChangeBps, &q.Projection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewError(404, httpx.CodeNotFound, "Forecast question not found.", err)
		}
		return nil, httpx.WrapInternal("get forecast question", err)
	}
	if outcomeText != nil {
		outcome := QuestionOutcome(*outcomeText)
		q.Outcome = &outcome
	}
	q.LinkedMarketID = linkedMarketID
	q.ResolvedAt = resolvedAt
	q.EvidenceURL = evidenceURL
	q.Projection.ForecastQuestionID = q.ID
	if total := marketYesReserve + marketNoReserve; total > 0 {
		q.LinkedMarketYesPrice = marketNoReserve / total
		q.LinkedMarketNoPrice = marketYesReserve / total
	}

	contributors, err := r.getContributors(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("load contributors: %w", err)
	}
	revisions, err := r.ListRevisions(ctx, questionID, 25)
	if err != nil {
		return nil, fmt.Errorf("load revisions: %w", err)
	}
	scores, err := r.ListScores(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("load scores: %w", err)
	}
	signals, err := r.ListExternalSignals(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("load signals: %w", err)
	}
	q.Contributors = contributors
	q.LatestRationales = revisions
	q.Scores = scores
	q.ExternalSignals = signals
	return q, nil
}

func (r *Repository) getContributors(ctx context.Context, questionID uuid.UUID) ([]Contributor, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.name, u.email, u.is_admin
		FROM forecast_question_contributors fqc
		JOIN users u ON u.id = fqc.user_id
		WHERE fqc.forecast_question_id = $1
		ORDER BY u.name ASC
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("list contributors: %w", err)
	}
	defer rows.Close()
	var contributors []Contributor
	for rows.Next() {
		var c Contributor
		if err := rows.Scan(&c.UserID, &c.Name, &c.Email, &c.IsExecutive); err != nil {
			return nil, fmt.Errorf("scan contributor: %w", err)
		}
		contributors = append(contributors, c)
	}
	return contributors, rows.Err()
}

func (r *Repository) IsContributor(ctx context.Context, questionID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM forecast_question_contributors WHERE forecast_question_id = $1 AND user_id = $2
		)
	`, questionID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check contributor: %w", err)
	}
	return exists, nil
}

func (r *Repository) AddRevision(ctx context.Context, rev *Revision) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add revision tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO forecast_revisions (forecast_question_id, user_id, probability_bps, rationale)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, rev.ForecastQuestionID, rev.UserID, rev.ProbabilityBps, rev.Rationale).Scan(&rev.ID, &rev.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert revision: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO forecast_latest_active (forecast_question_id, user_id, forecast_revision_id, probability_bps, rationale)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (forecast_question_id, user_id)
		DO UPDATE SET forecast_revision_id = EXCLUDED.forecast_revision_id,
		              probability_bps = EXCLUDED.probability_bps,
		              rationale = EXCLUDED.rationale,
		              updated_at = NOW()
	`, rev.ForecastQuestionID, rev.UserID, rev.ID, rev.ProbabilityBps, rev.Rationale); err != nil {
		return fmt.Errorf("upsert latest active: %w", err)
	}

	projection, err := r.computeProjectionTx(ctx, tx, rev.ForecastQuestionID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO forecast_projections (forecast_question_id, official_probability_bps, current_risk_bps, contributor_count, last_change_bps)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (forecast_question_id)
		DO UPDATE SET official_probability_bps = EXCLUDED.official_probability_bps,
		              current_risk_bps = EXCLUDED.current_risk_bps,
		              contributor_count = EXCLUDED.contributor_count,
		              last_change_bps = EXCLUDED.last_change_bps,
		              updated_at = NOW()
	`, rev.ForecastQuestionID, projection.OfficialProbabilityBps, projection.CurrentRiskBps, projection.ContributorCount, projection.LastChangeBps); err != nil {
		return fmt.Errorf("upsert projection: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE forecast_questions SET updated_at = NOW() WHERE id = $1`, rev.ForecastQuestionID); err != nil {
		return fmt.Errorf("touch question: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) computeProjectionTx(ctx context.Context, tx pgx.Tx, questionID uuid.UUID) (*Projection, error) {
	rows, err := tx.Query(ctx, `SELECT probability_bps FROM forecast_latest_active WHERE forecast_question_id = $1 ORDER BY probability_bps ASC`, questionID)
	if err != nil {
		return nil, fmt.Errorf("load latest active for projection: %w", err)
	}
	defer rows.Close()
	var values []int
	for rows.Next() {
		var value int
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan projection probability: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projection probabilities: %w", err)
	}
	projection := &Projection{ForecastQuestionID: questionID, ContributorCount: len(values)}
	if len(values) == 0 {
		return projection, nil
	}
	median := medianBps(values)
	projection.OfficialProbabilityBps = median
	projection.CurrentRiskBps = 10000 - median
	var current int
	_ = tx.QueryRow(ctx, `SELECT official_probability_bps FROM forecast_projections WHERE forecast_question_id = $1`, questionID).Scan(&current)
	projection.LastChangeBps = median - current
	return projection, nil
}

func medianBps(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return int(math.Round(float64(values[mid-1]+values[mid]) / 2.0))
}

func (r *Repository) ListRevisions(ctx context.Context, questionID uuid.UUID, limit int) ([]Revision, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT fr.id, fr.forecast_question_id, fr.user_id, u.name, fr.probability_bps, fr.rationale, fr.created_at
		FROM forecast_revisions fr
		JOIN users u ON u.id = fr.user_id
		WHERE fr.forecast_question_id = $1
		ORDER BY fr.created_at DESC
		LIMIT $2
	`, questionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list revisions: %w", err)
	}
	defer rows.Close()
	var revisions []Revision
	for rows.Next() {
		var rev Revision
		if err := rows.Scan(&rev.ID, &rev.ForecastQuestionID, &rev.UserID, &rev.UserName, &rev.ProbabilityBps, &rev.Rationale, &rev.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan revision: %w", err)
		}
		revisions = append(revisions, rev)
	}
	return revisions, rows.Err()
}

func (r *Repository) ResolveQuestion(ctx context.Context, questionID, resolverID uuid.UUID, outcome QuestionOutcome, evidenceURL string) (*Question, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin resolve question tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var q Question
	if err := tx.QueryRow(ctx, `
		UPDATE forecast_questions
		SET status = 'resolved', outcome = $1, evidence_url = $2, resolved_at = NOW(), updated_at = NOW()
		WHERE id = $3 AND resolver_id = $4 AND status IN ('open', 'closed')
		RETURNING id, closes_at, resolves_at, resolved_at, linked_market_id
	`, string(outcome), evidenceURL, questionID, resolverID).Scan(&q.ID, &q.ClosesAt, &q.ResolvesAt, &q.ResolvedAt, &q.LinkedMarketID); err != nil {
		return nil, httpx.NewError(409, httpx.CodeInvalidState, "Forecast question could not be resolved.", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO forecast_snapshots (forecast_question_id, user_id, forecast_revision_id, probability_bps, rationale)
		SELECT forecast_question_id, user_id, forecast_revision_id, probability_bps, rationale
		FROM forecast_latest_active
		WHERE forecast_question_id = $1
		ON CONFLICT (forecast_question_id, user_id) DO UPDATE
		SET forecast_revision_id = EXCLUDED.forecast_revision_id,
		    probability_bps = EXCLUDED.probability_bps,
		    rationale = EXCLUDED.rationale,
		    snapshotted_at = NOW()
	`, questionID); err != nil {
		return nil, fmt.Errorf("snapshot forecasts at close: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit resolve question: %w", err)
	}

	return r.GetQuestion(ctx, questionID)
}

func (r *Repository) CreateScores(ctx context.Context, questionID uuid.UUID, outcome QuestionOutcome, closesAt, createdAt time.Time) error {
	rows, err := r.db.Query(ctx, `
		SELECT fs.user_id, u.name, fs.probability_bps, fs.forecast_revision_id,
		       (SELECT COUNT(*) FROM forecast_revisions fr WHERE fr.forecast_question_id = fs.forecast_question_id AND fr.user_id = fs.user_id) AS revision_count,
		       (SELECT MIN(fr.created_at) FROM forecast_revisions fr WHERE fr.forecast_question_id = fs.forecast_question_id AND fr.user_id = fs.user_id) AS first_revision_at
		FROM forecast_snapshots fs
		JOIN users u ON u.id = fs.user_id
		WHERE fs.forecast_question_id = $1
	`, questionID)
	if err != nil {
		return fmt.Errorf("load snapshots for scoring: %w", err)
	}
	defer rows.Close()
	outcomeValue := 0
	if outcome == OutcomeDelivered {
		outcomeValue = 1
	}
	for rows.Next() {
		var userID uuid.UUID
		var userName string
		var probabilityBps, revisionCount int
		var revisionID uuid.UUID
		var firstRevisionAt time.Time
		if err := rows.Scan(&userID, &userName, &probabilityBps, &revisionID, &revisionCount, &firstRevisionAt); err != nil {
			return fmt.Errorf("scan scoring snapshot: %w", err)
		}
		p := float64(probabilityBps) / 10000.0
		y := float64(outcomeValue)
		brier := math.Pow(p-y, 2)
		denominator := closesAt.Sub(createdAt).Hours()
		coverage := 1.0
		if denominator > 0 {
			coverage = math.Min(1, math.Max(0, closesAt.Sub(firstRevisionAt).Hours()/denominator))
		}
		if _, err := r.db.Exec(ctx, `
			INSERT INTO forecast_score_records (forecast_question_id, user_id, probability_bps, outcome_value, brier_score, coverage_score, revision_count)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (forecast_question_id, user_id)
			DO UPDATE SET probability_bps = EXCLUDED.probability_bps,
			              outcome_value = EXCLUDED.outcome_value,
			              brier_score = EXCLUDED.brier_score,
			              coverage_score = EXCLUDED.coverage_score,
			              revision_count = EXCLUDED.revision_count,
			              created_at = NOW()
		`, questionID, userID, probabilityBps, outcomeValue, brier, coverage, revisionCount); err != nil {
			return fmt.Errorf("upsert score record: %w", err)
		}
	}
	return rows.Err()
}

func (r *Repository) ListScores(ctx context.Context, questionID uuid.UUID) ([]ScoreRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fsr.user_id, u.name, fsr.probability_bps, fsr.outcome_value, fsr.brier_score, fsr.coverage_score, fsr.revision_count, fsr.created_at
		FROM forecast_score_records fsr
		JOIN users u ON u.id = fsr.user_id
		WHERE fsr.forecast_question_id = $1
		ORDER BY fsr.brier_score ASC
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("list scores: %w", err)
	}
	defer rows.Close()
	var scores []ScoreRecord
	for rows.Next() {
		var s ScoreRecord
		if err := rows.Scan(&s.UserID, &s.UserName, &s.ProbabilityBps, &s.OutcomeValue, &s.BrierScore, &s.CoverageScore, &s.RevisionCount, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

func (r *Repository) CreateExternalSignal(ctx context.Context, signal *ExternalSignal) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO external_signal_snapshots (forecast_question_id, source, probability_bps, note)
		VALUES ($1,$2,$3,$4)
		RETURNING id, captured_at
	`, signal.ForecastQuestionID, signal.Source, signal.ProbabilityBps, signal.Note).Scan(&signal.ID, &signal.CapturedAt)
	if err != nil {
		return fmt.Errorf("create external signal: %w", err)
	}
	return nil
}

func (r *Repository) ListExternalSignals(ctx context.Context, questionID uuid.UUID) ([]ExternalSignal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, forecast_question_id, source, probability_bps, note, captured_at
		FROM external_signal_snapshots
		WHERE forecast_question_id = $1
		ORDER BY captured_at DESC
		LIMIT 10
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("list external signals: %w", err)
	}
	defer rows.Close()
	var signals []ExternalSignal
	for rows.Next() {
		var s ExternalSignal
		if err := rows.Scan(&s.ID, &s.ForecastQuestionID, &s.Source, &s.ProbabilityBps, &s.Note, &s.CapturedAt); err != nil {
			return nil, fmt.Errorf("scan external signal: %w", err)
		}
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

func (r *Repository) ProgramRiskView(ctx context.Context, program string) (*ProgramRiskView, error) {
	questions, err := r.ListQuestions(ctx, program, "")
	if err != nil {
		return nil, err
	}
	view := &ProgramRiskView{Program: program}
	for _, q := range questions {
		row := ProgramRiskRow{
			QuestionID:             q.ID,
			Title:                  q.Title,
			OfficialProbabilityBps: q.Projection.OfficialProbabilityBps,
			CurrentRiskBps:         q.Projection.CurrentRiskBps,
			ResolutionOwner:        q.ResolverName,
			ClosesAt:               q.ClosesAt,
			ResolvesAt:             q.ResolvesAt,
			Status:                 q.Status,
		}
		changeBps, err := r.changeLast7Days(ctx, q.ID, q.Projection.OfficialProbabilityBps)
		if err == nil {
			row.ChangeLast7DaysBps = changeBps
		}
		if len(q.ExternalSignals) == 0 {
			signals, err := r.ListExternalSignals(ctx, q.ID)
			if err == nil && len(signals) > 0 {
				q.ExternalSignals = signals
			}
		}
		if len(q.ExternalSignals) > 0 {
			row.ExternalSignalProbability = &q.ExternalSignals[0].ProbabilityBps
		}
		if len(q.LatestRationales) == 0 {
			revisions, err := r.ListRevisions(ctx, q.ID, 3)
			if err == nil {
				q.LatestRationales = revisions
			}
		}
		for _, rev := range q.LatestRationales {
			row.LatestRationaleExcerpts = append(row.LatestRationaleExcerpts, rev.Rationale)
			if len(row.LatestRationaleExcerpts) >= 3 {
				break
			}
		}
		view.Questions = append(view.Questions, row)
	}
	return view, nil
}

func (r *Repository) changeLast7Days(ctx context.Context, questionID uuid.UUID, currentOfficial int) (int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT probability_bps
		FROM forecast_revisions
		WHERE forecast_question_id = $1 AND created_at >= NOW() - INTERVAL '7 days'
		ORDER BY created_at ASC
	`, questionID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var values []int
	for rows.Next() {
		var value int
		if err := rows.Scan(&value); err != nil {
			return 0, err
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return 0, nil
	}
	baseline := medianBps(values)
	return currentOfficial - baseline, nil
}

func (r *Repository) EnqueueJob(ctx context.Context, kind string, payload any, runAt time.Time, dedupeKey string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO background_jobs (kind, dedupe_key, payload, run_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (dedupe_key) DO NOTHING
	`, kind, dedupeKey, data, runAt)
	if err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}
	return nil
}

func (r *Repository) FetchDueJobs(ctx context.Context, limit int) ([]BackgroundJob, error) {
	if limit <= 0 {
		limit = 25
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin fetch due jobs tx: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id, kind, dedupe_key, payload::text, run_at, status, attempts, max_attempts, last_error, processed_at, created_at, updated_at
		FROM background_jobs
		WHERE status = 'pending' AND run_at <= NOW()
		ORDER BY run_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query due jobs: %w", err)
	}
	var jobs []BackgroundJob
	for rows.Next() {
		var j BackgroundJob
		var payloadText string
		if err := rows.Scan(&j.ID, &j.Kind, &j.DedupeKey, &payloadText, &j.RunAt, &j.Status, &j.Attempts, &j.MaxAttempts, &j.LastError, &j.ProcessedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan due job: %w", err)
		}
		j.Payload = []byte(payloadText)
		jobs = append(jobs, j)
	}
	rows.Close()
	for _, job := range jobs {
		if _, err := tx.Exec(ctx, `UPDATE background_jobs SET status = 'processing', updated_at = NOW() WHERE id = $1`, job.ID); err != nil {
			return nil, fmt.Errorf("mark job processing: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit due jobs tx: %w", err)
	}
	return jobs, nil
}

func (r *Repository) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE background_jobs SET status = 'completed', processed_at = NOW(), updated_at = NOW() WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

func (r *Repository) FailJob(ctx context.Context, jobID uuid.UUID, message string, retryAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE background_jobs
		SET attempts = attempts + 1,
		    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
		    run_at = CASE WHEN attempts + 1 >= max_attempts THEN run_at ELSE $2 END,
		    last_error = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, jobID, retryAt, message)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

func (r *Repository) ListUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]user.Summary, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `SELECT id, name, email, is_admin FROM users WHERE id = ANY($1) ORDER BY name ASC`, ids)
	if err != nil {
		return nil, fmt.Errorf("list users by ids: %w", err)
	}
	defer rows.Close()
	var summaries []user.Summary
	for rows.Next() {
		var summary user.Summary
		if err := rows.Scan(&summary.ID, &summary.Name, &summary.Email, &summary.IsAdmin); err != nil {
			return nil, fmt.Errorf("scan user summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}
