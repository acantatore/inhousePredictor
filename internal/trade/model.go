package trade

import (
	"time"

	"github.com/google/uuid"
)

type Side string

const (
	SideYes Side = "yes"
	SideNo  Side = "no"
)

type Trade struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	MarketID       uuid.UUID `json:"market_id"`
	Side           Side      `json:"side"`
	Shares         float64   `json:"shares"`
	Cost           int64     `json:"cost"`
	YesPriceBefore float64   `json:"yes_price_before"`
	YesPriceAfter  float64   `json:"yes_price_after"`
	CreatedAt      time.Time `json:"created_at"`
}

type Position struct {
	UserID    uuid.UUID `json:"user_id"`
	MarketID  uuid.UUID `json:"market_id"`
	YesShares float64   `json:"yes_shares"`
	NoShares  float64   `json:"no_shares"`
}
