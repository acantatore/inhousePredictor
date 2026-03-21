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
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	MarketID       uuid.UUID  `json:"market_id"`
	Side           Side       `json:"side"`
	OptionID       *uuid.UUID `json:"option_id,omitempty"`
	OptionLabel    string     `json:"option_label,omitempty"`
	Shares         float64    `json:"shares"`
	Cost           int64      `json:"cost"`
	UserBalance    int64      `json:"user_balance,omitempty"`
	YesPriceBefore float64    `json:"yes_price_before"`
	YesPriceAfter  float64    `json:"yes_price_after"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Position struct {
	UserID          uuid.UUID       `json:"user_id"`
	MarketID        uuid.UUID       `json:"market_id"`
	MarketQuestion  string          `json:"market_question,omitempty"`
	MarketStatus    string          `json:"market_status,omitempty"`
	MarketOutcome   *string         `json:"market_outcome,omitempty"`
	WinningOptionID *uuid.UUID      `json:"winning_option_id,omitempty"`
	YesShares       float64         `json:"yes_shares"`
	NoShares        float64         `json:"no_shares"`
	Holdings        []OptionHolding `json:"holdings,omitempty"`
}

type OptionHolding struct {
	OptionID    uuid.UUID `json:"option_id"`
	OptionLabel string    `json:"option_label"`
	Shares      float64   `json:"shares"`
}

type MarketSnapshot struct {
	Status          string
	CreatorID       uuid.UUID
	ClosesAt        time.Time
	YesReserve      float64
	NoReserve       float64
	TotalCollateral int64
}
