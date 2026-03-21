package trade

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PayoutService struct {
	db *pgxpool.Pool
}

func NewPayoutService(db *pgxpool.Pool) *PayoutService {
	return &PayoutService{db: db}
}

func (s *PayoutService) Payout(ctx context.Context, marketID uuid.UUID, outcome string) error {
	repo := NewRepository(s.db)
	return repo.Payout(ctx, marketID, outcome)
}

func (s *PayoutService) FinalizeEligiblePayouts(ctx context.Context) error {
	repo := NewRepository(s.db)
	ids, outcomes, err := repo.ReadyForPayout(ctx)
	if err != nil {
		return err
	}
	for i := range ids {
		if err := repo.Payout(ctx, ids[i], outcomes[i]); err != nil {
			return err
		}
	}
	return nil
}
