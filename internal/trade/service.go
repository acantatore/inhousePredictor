package trade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/ws"
)

type Service struct {
	repo *Repository
	hub  *ws.Hub
}

const maxSerializationRetries = 3

func NewService(repo *Repository, hub *ws.Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

type TradeRequest struct {
	UserID   uuid.UUID
	MarketID uuid.UUID
	Side     Side
	Cost     int64
}

func (s *Service) Execute(ctx context.Context, req TradeRequest) (*Trade, error) {
	if req.Cost <= 0 {
		return nil, httpx.NewError(400, httpx.CodeValidation, "Cost must be positive.", fmt.Errorf("cost must be positive"))
	}
	var result *executeResult
	var err error
	for attempt := 0; attempt < maxSerializationRetries; attempt++ {
		result, err = s.repo.execute(ctx, executeParams{
			UserID:   req.UserID,
			MarketID: req.MarketID,
			Side:     req.Side,
			Cost:     req.Cost,
		})
		if err == nil {
			break
		}
		if !httpx.IsSerialization(err) {
			return nil, err
		}
	}
	if err != nil {
		return nil, httpx.NewError(409, httpx.CodeMarketBusy, "This market is busy right now. Please try again.", err)
	}

	if s.hub != nil {
		// Broadcast new price to all subscribers of this market.
		s.hub.Broadcast(ws.Message{
			Type:     ws.MsgPriceUpdate,
			MarketID: req.MarketID.String(),
			Payload: ws.PriceUpdatePayload{
				YesPrice: result.NewPool.YesPrice(),
				NoPrice:  result.NewPool.NoPrice(),
				LastTrade: &ws.TradeUpdate{
					Side:   string(req.Side),
					Shares: result.Trade.Shares,
					Cost:   result.Trade.Cost,
				},
			},
		})
	}

	return result.Trade, nil
}

func (s *Service) GetPositions(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	if err := s.FinalizeEligiblePayouts(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetPositions(ctx, userID)
}

func (s *Service) GetMarketTrades(ctx context.Context, marketID uuid.UUID) ([]*Trade, error) {
	return s.repo.GetMarketTrades(ctx, marketID, 50)
}

func (s *Service) Payout(ctx context.Context, marketID uuid.UUID, outcome string) error {
	return s.repo.Payout(ctx, marketID, outcome)
}

func (s *Service) FinalizeEligiblePayouts(ctx context.Context) error {
	ids, outcomes, err := s.repo.ReadyForPayout(ctx)
	if err != nil {
		return err
	}
	for i := range ids {
		if err := s.repo.Payout(ctx, ids[i], outcomes[i]); err != nil {
			return err
		}
	}
	return nil
}
