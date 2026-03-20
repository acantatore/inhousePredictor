package market

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/cpmm"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateParams struct {
	Question         string
	Description      string
	Category         Category
	CreatorID        uuid.UUID
	ResolverID       uuid.UUID
	InitialLiquidity int64
	ClosesAt         time.Time
	ResolvesAt       time.Time
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Market, error) {
	if !ValidCategories[p.Category] {
		return nil, fmt.Errorf("invalid category: must be one of people, okrs, slas, financials, general")
	}
	if p.InitialLiquidity < 100 {
		return nil, fmt.Errorf("minimum initial liquidity is 100 points")
	}
	if p.CreatorID == p.ResolverID {
		return nil, fmt.Errorf("creator and resolver must be different people")
	}
	if p.ClosesAt.Before(time.Now()) {
		return nil, fmt.Errorf("closes_at must be in the future")
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
		K:               pool.K,
		TotalCollateral: p.InitialLiquidity,
	}
	if err := s.repo.Create(ctx, m, dbPool); err != nil {
		return nil, err
	}
	m.YesPrice = 0.5
	m.NoPrice = 0.5
	return m, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Market, error) {
	m, _, err := s.repo.GetByID(ctx, id)
	return m, err
}

func (s *Service) List(ctx context.Context, category Category, status Status) ([]*Market, error) {
	return s.repo.List(ctx, category, status)
}

func (s *Service) Resolve(ctx context.Context, marketID, resolverID uuid.UUID, outcome Outcome, evidenceURL string) error {
	if outcome != OutcomeYes && outcome != OutcomeNo {
		return fmt.Errorf("outcome must be 'yes' or 'no'")
	}
	return s.repo.Resolve(ctx, ResolveParams{
		MarketID:    marketID,
		ResolverID:  resolverID,
		Outcome:     outcome,
		EvidenceURL: evidenceURL,
	})
}

func (s *Service) Dispute(ctx context.Context, marketID, userID uuid.UUID, reason string) error {
	return s.repo.Dispute(ctx, marketID, userID, reason)
}
