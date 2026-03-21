package market

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakePayoutService struct {
	called         int
	finalizeCalled int
	lastMarketID   uuid.UUID
	lastOutcome    string
	finalizeErr    error
	payoutErr      error
}

func (f *fakePayoutService) Payout(_ context.Context, marketID uuid.UUID, outcome string) error {
	f.called++
	f.lastMarketID = marketID
	f.lastOutcome = outcome
	return f.payoutErr
}

func (f *fakePayoutService) FinalizeEligiblePayouts(context.Context) error {
	f.finalizeCalled++
	return f.finalizeErr
}

func TestCreateRejectsInvalidTiming(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.Create(context.Background(), CreateParams{
		Question:         "Will it ship?",
		Description:      "desc",
		Category:         CategoryGeneral,
		CreatorID:        uuid.New(),
		ResolverID:       uuid.New(),
		InitialLiquidity: 100,
		ClosesAt:         time.Now().Add(2 * time.Hour),
		ResolvesAt:       time.Now().Add(time.Hour),
	})
	require.Error(t, err)
}

func TestResolveValidatesEvidenceURL(t *testing.T) {
	svc := NewService(&Repository{}, &fakePayoutService{})
	err := svc.Resolve(context.Background(), ResolveRequest{MarketID: uuid.New(), ResolverID: uuid.New(), Outcome: OutcomeYes, EvidenceURL: "not-a-url"})
	require.Error(t, err)
}
