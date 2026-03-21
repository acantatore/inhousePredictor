package forecast

import (
	"time"

	"github.com/google/uuid"
)

type QuestionStatus string

const (
	QuestionStatusOpen      QuestionStatus = "open"
	QuestionStatusClosed    QuestionStatus = "closed"
	QuestionStatusResolved  QuestionStatus = "resolved"
	QuestionStatusCancelled QuestionStatus = "cancelled"
)

type QuestionOutcome string

const (
	OutcomeDelivered    QuestionOutcome = "delivered"
	OutcomeNotDelivered QuestionOutcome = "not_delivered"
	OutcomeCancelled    QuestionOutcome = "cancelled"
)

type Question struct {
	ID                       uuid.UUID        `json:"id"`
	Title                    string           `json:"title"`
	Description              string           `json:"description"`
	Program                  string           `json:"program"`
	OwnerID                  uuid.UUID        `json:"owner_id"`
	OwnerName                string           `json:"owner_name,omitempty"`
	ResolverID               uuid.UUID        `json:"resolver_id"`
	ResolverName             string           `json:"resolver_name,omitempty"`
	Status                   QuestionStatus   `json:"status"`
	Outcome                  *QuestionOutcome `json:"outcome,omitempty"`
	ResolutionRule           string           `json:"resolution_rule"`
	RationalePolicyThreshold int              `json:"rationale_policy_threshold"`
	LinkedMarketID           *uuid.UUID       `json:"linked_market_id,omitempty"`
	ClosesAt                 time.Time        `json:"closes_at"`
	ResolvesAt               time.Time        `json:"resolves_at"`
	ResolvedAt               *time.Time       `json:"resolved_at,omitempty"`
	EvidenceURL              *string          `json:"evidence_url,omitempty"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
	Projection               *Projection      `json:"projection,omitempty"`
	Contributors             []Contributor    `json:"contributors,omitempty"`
	LatestRationales         []Revision       `json:"latest_rationales,omitempty"`
	Scores                   []ScoreRecord    `json:"scores,omitempty"`
	ExternalSignals          []ExternalSignal `json:"external_signals,omitempty"`
}

type Contributor struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	IsExecutive bool      `json:"is_executive"`
}

type Revision struct {
	ID                 uuid.UUID `json:"id"`
	ForecastQuestionID uuid.UUID `json:"forecast_question_id"`
	UserID             uuid.UUID `json:"user_id"`
	UserName           string    `json:"user_name,omitempty"`
	ProbabilityBps     int       `json:"probability_bps"`
	Rationale          string    `json:"rationale"`
	CreatedAt          time.Time `json:"created_at"`
}

type Projection struct {
	ForecastQuestionID     uuid.UUID `json:"forecast_question_id"`
	OfficialProbabilityBps int       `json:"official_probability_bps"`
	CurrentRiskBps         int       `json:"current_risk_bps"`
	ContributorCount       int       `json:"contributor_count"`
	LastChangeBps          int       `json:"last_change_bps"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type ScoreRecord struct {
	UserID         uuid.UUID `json:"user_id"`
	UserName       string    `json:"user_name,omitempty"`
	ProbabilityBps int       `json:"probability_bps"`
	OutcomeValue   int       `json:"outcome_value"`
	BrierScore     float64   `json:"brier_score"`
	CoverageScore  float64   `json:"coverage_score"`
	RevisionCount  int       `json:"revision_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type ExternalSignal struct {
	ID                 uuid.UUID `json:"id"`
	ForecastQuestionID uuid.UUID `json:"forecast_question_id"`
	Source             string    `json:"source"`
	ProbabilityBps     int       `json:"probability_bps"`
	Note               string    `json:"note"`
	CapturedAt         time.Time `json:"captured_at"`
}

type ProgramRiskView struct {
	Program   string           `json:"program"`
	Questions []ProgramRiskRow `json:"questions"`
}

type ProgramRiskRow struct {
	QuestionID                uuid.UUID      `json:"question_id"`
	Title                     string         `json:"title"`
	OfficialProbabilityBps    int            `json:"official_probability_bps"`
	CurrentRiskBps            int            `json:"current_risk_bps"`
	ChangeLast7DaysBps        int            `json:"change_last_7_days_bps"`
	LatestRationaleExcerpts   []string       `json:"latest_rationale_excerpts"`
	ResolutionOwner           string         `json:"resolution_owner"`
	ClosesAt                  time.Time      `json:"closes_at"`
	ResolvesAt                time.Time      `json:"resolves_at"`
	ExternalSignalProbability *int           `json:"external_signal_probability_bps,omitempty"`
	Status                    QuestionStatus `json:"status"`
}

type BackgroundJob struct {
	ID          uuid.UUID  `json:"id"`
	Kind        string     `json:"kind"`
	DedupeKey   string     `json:"dedupe_key"`
	Payload     []byte     `json:"payload"`
	RunAt       time.Time  `json:"run_at"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	LastError   *string    `json:"last_error,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
