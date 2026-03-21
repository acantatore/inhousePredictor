package forecast

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/market"
	"github.com/naranjax/inhousepredictor/internal/validate"
)

const defaultRationaleThreshold = 10

type marketCreator interface {
	CreateShadow(ctx context.Context, m *market.Market, p *market.Pool) error
	SyncShadowLifecycle(ctx context.Context, forecastQuestionID uuid.UUID, status market.Status, outcome *market.Outcome, evidenceURL *string, closesAt, resolvesAt time.Time, resolvedAt *time.Time) error
}

type Service struct {
	repo       *Repository
	marketRepo marketCreator
	logger     *slog.Logger
}

func NewService(repo *Repository, marketRepo marketCreator, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, marketRepo: marketRepo, logger: logger}
}

type CreateQuestionParams struct {
	Title          string
	Description    string
	Program        string
	OwnerID        uuid.UUID
	ResolverID     uuid.UUID
	ResolutionRule string
	ClosesAt       time.Time
	ResolvesAt     time.Time
	ContributorIDs []uuid.UUID
}

func (s *Service) CreateQuestion(ctx context.Context, params CreateQuestionParams) (*Question, error) {
	if err := validate.Question(params.Title); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	if err := validate.Description(params.Description); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	if params.Program == "" {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Program is required.", nil)
	}
	if params.ResolutionRule == "" {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Resolution rule is required.", nil)
	}
	if params.OwnerID == params.ResolverID {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Owner and resolver must be different people.", nil)
	}
	if err := validate.MarketTiming(time.Now(), params.ClosesAt, params.ResolvesAt); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	contributors := dedupeContributors(append(params.ContributorIDs, params.OwnerID))
	if len(contributors) == 0 {
		return nil, httpx.NewError(400, httpx.CodeValidation, "At least one contributor is required.", nil)
	}
	q := &Question{
		Title:                    params.Title,
		Description:              params.Description,
		Program:                  params.Program,
		OwnerID:                  params.OwnerID,
		ResolverID:               params.ResolverID,
		ResolutionRule:           params.ResolutionRule,
		RationalePolicyThreshold: defaultRationaleThreshold,
		ClosesAt:                 params.ClosesAt,
		ResolvesAt:               params.ResolvesAt,
		Status:                   QuestionStatusOpen,
	}
	if err := s.repo.CreateQuestion(ctx, q, contributors); err != nil {
		return nil, err
	}
	linkedQuestionID := q.ID
	shadowMarket := &market.Market{
		Question:           q.Title,
		Description:        q.Description,
		Category:           market.CategoryGeneral,
		CreatorID:          q.OwnerID,
		ResolverID:         q.ResolverID,
		InitialLiquidity:   100,
		ClosesAt:           q.ClosesAt,
		ResolvesAt:         q.ResolvesAt,
		Status:             market.StatusOpen,
		IsShadow:           true,
		ForecastQuestionID: &linkedQuestionID,
	}
	shadowPool := &market.Pool{YesReserve: 100, NoReserve: 100, TotalCollateral: 100}
	if err := s.marketRepo.CreateShadow(ctx, shadowMarket, shadowPool); err != nil {
		return nil, err
	}
	if err := s.repo.AttachMarket(ctx, q.ID, shadowMarket.ID); err != nil {
		return nil, err
	}
	q.LinkedMarketID = &shadowMarket.ID
	if err := s.enqueueReminderJobs(ctx, q.ID, q.ClosesAt, contributors); err != nil {
		return nil, err
	}
	summaries, err := s.repo.ListUsersByIDs(ctx, contributors)
	if err == nil {
		for _, summary := range summaries {
			q.Contributors = append(q.Contributors, Contributor{UserID: summary.ID, Name: summary.Name, Email: summary.Email, IsExecutive: summary.IsAdmin})
		}
	}
	q.Projection = &Projection{ForecastQuestionID: q.ID, CurrentRiskBps: 0, OfficialProbabilityBps: 0, ContributorCount: 0}
	return q, nil
}

func (s *Service) ListQuestions(ctx context.Context, program string, status QuestionStatus) ([]*Question, error) {
	return s.repo.ListQuestions(ctx, program, status)
}

func (s *Service) GetQuestion(ctx context.Context, questionID uuid.UUID) (*Question, error) {
	return s.repo.GetQuestion(ctx, questionID)
}

type SubmitForecastParams struct {
	QuestionID     uuid.UUID
	UserID         uuid.UUID
	ProbabilityBps int
	Rationale      string
}

func (s *Service) SubmitForecast(ctx context.Context, params SubmitForecastParams) (*Question, error) {
	question, err := s.repo.GetQuestion(ctx, params.QuestionID)
	if err != nil {
		return nil, err
	}
	if question.Status != QuestionStatusOpen {
		return nil, httpx.NewError(409, httpx.CodeMarketClosed, "This forecast question is closed.", nil)
	}
	if !question.ClosesAt.After(time.Now()) {
		return nil, httpx.NewError(409, httpx.CodeMarketClosed, "This forecast question is closed.", nil)
	}
	if params.ProbabilityBps < 0 || params.ProbabilityBps > 10000 {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Probability must be between 0 and 100%.", nil)
	}
	allowed, err := s.repo.IsContributor(ctx, params.QuestionID, params.UserID)
	if err != nil {
		return nil, httpx.WrapInternal("check contributor", err)
	}
	if !allowed {
		return nil, httpx.NewError(403, httpx.CodeForbidden, "You are not allowed to forecast on this commitment.", nil)
	}
	if stringsTrim(params.Rationale) == "" {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Rationale is required.", nil)
	}
	latest, err := s.repo.ListRevisions(ctx, params.QuestionID, 1)
	if err == nil && len(latest) > 0 && latest[0].UserID == params.UserID {
		if math.Abs(float64(latest[0].ProbabilityBps-params.ProbabilityBps)) >= float64(question.RationalePolicyThreshold*100) && stringsTrim(params.Rationale) == "" {
			return nil, httpx.NewError(400, httpx.CodeValidation, "A rationale is required for significant confidence changes.", nil)
		}
	}
	revision := &Revision{ForecastQuestionID: params.QuestionID, UserID: params.UserID, ProbabilityBps: params.ProbabilityBps, Rationale: params.Rationale}
	if err := s.repo.AddRevision(ctx, revision); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetQuestion(ctx, params.QuestionID)
	if err != nil {
		return nil, err
	}
	if math.Abs(float64(updated.Projection.LastChangeBps)) >= 1000 {
		_ = s.repo.EnqueueJob(ctx, "consensus_change_notification", map[string]any{"forecast_question_id": params.QuestionID, "change_bps": updated.Projection.LastChangeBps}, time.Now(), fmt.Sprintf("consensus-change:%s:%d", params.QuestionID, updated.Projection.OfficialProbabilityBps))
	}
	return updated, nil
}

func (s *Service) ResolveQuestion(ctx context.Context, questionID, resolverID uuid.UUID, outcome QuestionOutcome, evidenceURL string) (*Question, error) {
	if outcome != OutcomeDelivered && outcome != OutcomeNotDelivered && outcome != OutcomeCancelled {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Outcome must be delivered, not_delivered, or cancelled.", nil)
	}
	if err := validate.URL(evidenceURL); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	q, err := s.repo.ResolveQuestion(ctx, questionID, resolverID, outcome, evidenceURL)
	if err != nil {
		return nil, err
	}
	var marketOutcome *market.Outcome
	if outcome == OutcomeDelivered {
		o := market.OutcomeYes
		marketOutcome = &o
	} else if outcome == OutcomeNotDelivered {
		o := market.OutcomeNo
		marketOutcome = &o
	} else {
		o := market.OutcomeCancelled
		marketOutcome = &o
	}
	if err := s.marketRepo.SyncShadowLifecycle(ctx, questionID, mapQuestionStatus(q.Status), marketOutcome, q.EvidenceURL, q.ClosesAt, q.ResolvesAt, q.ResolvedAt); err != nil {
		return nil, err
	}
	if err := s.repo.CreateScores(ctx, questionID, outcome, q.ClosesAt, q.CreatedAt); err != nil {
		return nil, err
	}
	contributors, _ := s.repo.getContributors(ctx, questionID)
	for _, c := range contributors {
		_ = s.repo.EnqueueJob(ctx, "resolution_notification", map[string]any{"forecast_question_id": questionID, "user_id": c.UserID, "outcome": outcome}, time.Now(), fmt.Sprintf("resolution:%s:%s", questionID, c.UserID))
	}
	return s.repo.GetQuestion(ctx, questionID)
}

func (s *Service) ProgramRiskView(ctx context.Context, program string) (*ProgramRiskView, error) {
	if program == "" {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Program is required.", nil)
	}
	return s.repo.ProgramRiskView(ctx, program)
}

func (s *Service) CreateExternalSignal(ctx context.Context, questionID uuid.UUID, source string, probabilityBps int, note string) (*ExternalSignal, error) {
	if source == "" {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Source is required.", nil)
	}
	if probabilityBps < 0 || probabilityBps > 10000 {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Probability must be between 0 and 100%.", nil)
	}
	signal := &ExternalSignal{ForecastQuestionID: questionID, Source: source, ProbabilityBps: probabilityBps, Note: note}
	if err := s.repo.CreateExternalSignal(ctx, signal); err != nil {
		return nil, err
	}
	return signal, nil
}

func (s *Service) enqueueReminderJobs(ctx context.Context, questionID uuid.UUID, closesAt time.Time, contributors []uuid.UUID) error {
	reminderAt := closesAt.Add(-24 * time.Hour)
	if reminderAt.Before(time.Now()) {
		return nil
	}
	for _, contributorID := range contributors {
		if err := s.repo.EnqueueJob(ctx, "forecast_reminder", map[string]any{"forecast_question_id": questionID, "user_id": contributorID}, reminderAt, fmt.Sprintf("forecast-reminder:%s:%s", questionID, contributorID)); err != nil {
			return err
		}
	}
	return nil
}

type Worker struct {
	repo   *Repository
	logger *slog.Logger
}

func NewWorker(repo *Repository, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{repo: repo, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jobs, err := w.repo.FetchDueJobs(ctx, 20)
			if err != nil {
				w.logger.Error("fetch due jobs", "error", err)
				continue
			}
			for _, job := range jobs {
				if err := w.handleJob(ctx, job); err != nil {
					w.logger.Error("job failed", "kind", job.Kind, "error", err)
					_ = w.repo.FailJob(ctx, job.ID, err.Error(), time.Now().Add(30*time.Second))
					continue
				}
				_ = w.repo.CompleteJob(ctx, job.ID)
			}
		}
	}
}

func (w *Worker) handleJob(ctx context.Context, job BackgroundJob) error {
	var payload map[string]any
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode job payload: %w", err)
		}
	}
	w.logger.Info("processed background job", "kind", job.Kind, "payload", payload)
	return nil
}

func dedupeContributors(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	var out []uuid.UUID
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}

func mapQuestionStatus(status QuestionStatus) market.Status {
	switch status {
	case QuestionStatusClosed:
		return market.StatusClosed
	case QuestionStatusResolved:
		return market.StatusResolved
	case QuestionStatusCancelled:
		return market.StatusCancelled
	default:
		return market.StatusOpen
	}
}
