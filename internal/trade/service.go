package trade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/ws"
)

type Service struct {
	repo *Repository
	hub  *ws.Hub
}

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
		return nil, fmt.Errorf("cost must be positive")
	}
	result, err := s.repo.execute(ctx, executeParams{
		UserID:   req.UserID,
		MarketID: req.MarketID,
		Side:     req.Side,
		Cost:     req.Cost,
	})
	if err != nil {
		return nil, err
	}

	// Broadcast new price to all subscribers of this market
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

	return result.Trade, nil
}

func (s *Service) GetPositions(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	return s.repo.GetPositions(ctx, userID)
}

func (s *Service) GetMarketTrades(ctx context.Context, marketID uuid.UUID) ([]*Trade, error) {
	return s.repo.GetMarketTrades(ctx, marketID, 50)
}

func (s *Service) Payout(ctx context.Context, marketID uuid.UUID, outcome string) error {
	side := SideYes
	if outcome == "no" {
		side = SideNo
	}
	return s.repo.Payout(ctx, marketID, side)
}
