package market

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/cpmm"
	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/validate"
)

type Service struct {
	repo      *Repository
	payoutSvc payoutService
}

type payoutService interface {
	Payout(ctx context.Context, marketID uuid.UUID, outcome string) error
	FinalizeEligiblePayouts(ctx context.Context) error
}

func NewService(repo *Repository, payoutSvc payoutService) *Service {
	return &Service{repo: repo, payoutSvc: payoutSvc}
}

type CreateParams struct {
	Question         string
	Description      string
	Category         Category
	Options          []string
	CreatorID        uuid.UUID
	ResolverID       uuid.UUID
	InitialLiquidity int64
	ClosesAt         time.Time
	ResolvesAt       time.Time
}

type ResolveRequest struct {
	MarketID        uuid.UUID
	ResolverID      uuid.UUID
	Outcome         Outcome
	WinningOptionID *uuid.UUID
	EvidenceURL     string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Market, error) {
	if err := validate.Question(p.Question); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	if err := validate.Description(p.Description); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	if !ValidCategories[p.Category] {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Invalid category.", nil)
	}
	if p.InitialLiquidity < 100 {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Minimum initial liquidity is 100 points.", nil)
	}
	if p.CreatorID == p.ResolverID {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Creator and resolver must be different people.", nil)
	}
	options := normalizeOptions(p.Options)
	if len(options) < 2 {
		options = []string{"YES", "NO"}
	}
	if err := validate.MarketTiming(time.Now(), p.ClosesAt, p.ResolvesAt); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}

	pool := cpmm.New(p.InitialLiquidity)
	m := &Market{
		Question:         p.Question,
		Description:      p.Description,
		Category:         p.Category,
		CreatorID:        p.CreatorID,
		ResolverID:       p.ResolverID,
		InitialLiquidity: p.InitialLiquidity,
		ClosesAt:         p.ClosesAt,
		ResolvesAt:       p.ResolvesAt,
		Status:           StatusOpen,
	}
	dbPool := &Pool{
		YesReserve:      pool.YesReserve,
		NoReserve:       pool.NoReserve,
		TotalCollateral: p.InitialLiquidity,
	}
	if err := s.repo.Create(ctx, m, dbPool); err != nil {
		return nil, err
	}
	perOption := p.InitialLiquidity / int64(len(options))
	if perOption < 1 {
		perOption = 1
	}
	for i, label := range options {
		option := Option{MarketID: m.ID, Label: label, SortOrder: i, Collateral: perOption, ProbabilityBps: 10000 / len(options)}
		m.Options = append(m.Options, option)
	}
	if err := s.repo.UpsertOptions(ctx, m.ID, m.Options); err != nil {
		return nil, err
	}
	if storedOptions, err := s.repo.GetOptions(ctx, m.ID); err == nil {
		m.Options = storedOptions
	}
	if err := s.repo.RecordSnapshot(ctx, m.ID); err != nil {
		return nil, err
	}
	applyOptionSummary(m)
	applySentiment(m)
	return m, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Market, error) {
	if s.payoutSvc != nil {
		if err := s.payoutSvc.FinalizeEligiblePayouts(ctx); err != nil {
			return nil, err
		}
	}
	m, _, err := s.repo.GetByID(ctx, id)
	return m, err
}

func normalizeOptions(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if _, ok := seen[upper]; ok {
			continue
		}
		seen[upper] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (s *Service) List(ctx context.Context, category Category, status Status) ([]*Market, error) {
	if s.payoutSvc != nil {
		if err := s.payoutSvc.FinalizeEligiblePayouts(ctx); err != nil {
			return nil, err
		}
	}
	return s.repo.List(ctx, category, status, 50)
}

func (s *Service) Resolve(ctx context.Context, req ResolveRequest) error {
	if req.Outcome != OutcomeYes && req.Outcome != OutcomeNo && req.Outcome != OutcomeCancelled {
		return httpx.NewError(400, httpx.CodeValidation, "Outcome must be yes, no, or cancelled.", nil)
	}
	if err := validate.URL(req.EvidenceURL); err != nil {
		return httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	if err := s.repo.Resolve(ctx, ResolveParams{
		MarketID:        req.MarketID,
		ResolverID:      req.ResolverID,
		Outcome:         req.Outcome,
		WinningOptionID: req.WinningOptionID,
		EvidenceURL:     req.EvidenceURL,
	}); err != nil {
		return err
	}
	if req.Outcome == OutcomeCancelled || s.payoutSvc == nil {
		return nil
	}
	return s.payoutSvc.Payout(ctx, req.MarketID, string(req.Outcome))
}

func (s *Service) Dispute(ctx context.Context, marketID, userID uuid.UUID, reason string) error {
	if reason == "" {
		return httpx.NewError(400, httpx.CodeValidation, "A dispute reason is required.", nil)
	}
	return s.repo.Dispute(ctx, marketID, userID, reason)
}

func (s *Service) ReviewDispute(ctx context.Context, marketID, adminID uuid.UUID, action DisputeAction, outcome *Outcome, evidenceURL *string) error {
	if err := s.repo.ReviewDispute(ctx, marketID, adminID, action, outcome, evidenceURL); err != nil {
		return err
	}
	if s.payoutSvc == nil || action != DisputeActionOverrideOutcome || outcome == nil || *outcome == OutcomeCancelled {
		return nil
	}
	return s.payoutSvc.Payout(ctx, marketID, string(*outcome))
}

func (s *Service) ListDisputes(ctx context.Context) ([]*DisputeRecord, error) {
	return s.repo.ListDisputes(ctx)
}
