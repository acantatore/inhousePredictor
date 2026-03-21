package market

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusOpen      Status = "open"
	StatusClosed    Status = "closed"
	StatusResolved  Status = "resolved"
	StatusDisputed  Status = "disputed"
	StatusCancelled Status = "cancelled"
)

type Outcome string

const (
	OutcomeYes       Outcome = "yes"
	OutcomeNo        Outcome = "no"
	OutcomeCancelled Outcome = "cancelled"
)

type Category string

const (
	CategoryPeople     Category = "people"
	CategoryOKRs       Category = "okrs"
	CategorySLAs       Category = "slas"
	CategoryFinancials Category = "financials"
	CategoryGeneral    Category = "general"
)

var ValidCategories = map[Category]bool{
	CategoryPeople: true, CategoryOKRs: true,
	CategorySLAs: true, CategoryFinancials: true, CategoryGeneral: true,
}

type Market struct {
	ID                 uuid.UUID  `json:"id"`
	Question           string     `json:"question"`
	Description        string     `json:"description"`
	Category           Category   `json:"category"`
	CreatorID          uuid.UUID  `json:"creator_id"`
	CreatorName        string     `json:"creator_name,omitempty"`
	ResolverID         uuid.UUID  `json:"resolver_id"`
	ResolverName       string     `json:"resolver_name,omitempty"`
	Status             Status     `json:"status"`
	Outcome            *Outcome   `json:"outcome,omitempty"`
	EvidenceURL        *string    `json:"evidence_url,omitempty"`
	InitialLiquidity   int64      `json:"initial_liquidity"`
	ClosesAt           time.Time  `json:"closes_at"`
	ResolvesAt         time.Time  `json:"resolves_at"`
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	DisputeDeadline    *time.Time `json:"dispute_deadline,omitempty"`
	IsShadow           bool       `json:"is_shadow,omitempty"`
	ForecastQuestionID *uuid.UUID `json:"forecast_question_id,omitempty"`
	Sentiment          string     `json:"sentiment,omitempty"`
	ChangeBps          int        `json:"change_bps,omitempty"`
	Options            []Option   `json:"options,omitempty"`
	Snapshots          []Snapshot `json:"snapshots,omitempty"`
	// Live pricing from pool
	YesPrice float64 `json:"yes_price"`
	NoPrice  float64 `json:"no_price"`
}

type Option struct {
	ID             uuid.UUID `json:"id"`
	MarketID       uuid.UUID `json:"market_id"`
	Label          string    `json:"label"`
	SortOrder      int       `json:"sort_order"`
	ProbabilityBps int       `json:"probability_bps"`
	Collateral     int64     `json:"collateral"`
	IsWinner       bool      `json:"is_winner,omitempty"`
}

type Snapshot struct {
	ID         uuid.UUID      `json:"id"`
	MarketID   uuid.UUID      `json:"market_id"`
	CapturedAt time.Time      `json:"captured_at"`
	Points     map[string]int `json:"points"`
}

type DisputeRecord struct {
	MarketID         uuid.UUID  `json:"market_id"`
	MarketQuestion   string     `json:"market_question"`
	DisputeID        uuid.UUID  `json:"dispute_id"`
	Reason           string     `json:"reason"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatorID        uuid.UUID  `json:"creator_id"`
	CreatorName      string     `json:"creator_name"`
	ResolverID       uuid.UUID  `json:"resolver_id"`
	ResolverName     string     `json:"resolver_name"`
	Status           Status     `json:"status"`
	Outcome          *Outcome   `json:"outcome,omitempty"`
	EvidenceURL      *string    `json:"evidence_url,omitempty"`
	ResolvedBy       *uuid.UUID `json:"resolved_by,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	DisputeDeadline  *time.Time `json:"dispute_deadline,omitempty"`
	InitialLiquidity int64      `json:"initial_liquidity"`
}

type Pool struct {
	MarketID        uuid.UUID
	YesReserve      float64
	NoReserve       float64
	TotalCollateral int64
}

type DisputeAction string

const (
	DisputeActionConfirmOriginal DisputeAction = "confirm_original"
	DisputeActionOverrideOutcome DisputeAction = "override_outcome"
	DisputeActionCancelMarket    DisputeAction = "cancel_market"
)
