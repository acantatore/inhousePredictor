export type Category = 'people' | 'okrs' | 'slas' | 'financials' | 'general';
export type MarketStatus = 'open' | 'closed' | 'resolved' | 'disputed' | 'cancelled';
export type Outcome = 'yes' | 'no' | 'cancelled';
export type TradeSide = 'yes' | 'no' | 'sell_yes' | 'sell_no';
export type DisputeAction = 'confirm_original' | 'override_outcome' | 'cancel_market';
export type ForecastQuestionStatus = 'open' | 'closed' | 'resolved' | 'cancelled';
export type ForecastQuestionOutcome = 'delivered' | 'not_delivered' | 'cancelled';

export interface User {
  id: string;
  name: string;
  email: string;
  balance: number;
  is_admin: boolean;
  created_at: string;
}

export interface UserSummary {
	id: string;
	name: string;
	email: string;
	is_admin: boolean;
}

export interface Market {
  id: string;
  question: string;
  description: string;
  category: Category;
  creator_id: string;
  creator_name?: string;
  resolver_id: string;
  resolver_name?: string;
  status: MarketStatus;
  outcome?: Outcome | null;
  evidence_url?: string | null;
  initial_liquidity: number;
  closes_at: string;
  resolves_at: string;
  created_at: string;
  resolved_at?: string | null;
  dispute_deadline?: string | null;
  sentiment?: string;
  change_bps?: number;
  options?: MarketOption[];
  snapshots?: MarketSnapshot[];
  yes_price: number;
  no_price: number;
}

export interface MarketOption {
  id: string;
  market_id: string;
  label: string;
  sort_order: number;
  probability_bps: number;
  collateral: number;
  is_winner?: boolean;
}

export interface MarketSnapshot {
  id: string;
  market_id: string;
  captured_at: string;
  points: Record<string, number>;
}

export interface Trade {
  id: string;
  user_id: string;
  market_id: string;
  side: TradeSide;
  option_id?: string | null;
  option_label?: string;
  shares: number;
  cost: number;
  user_balance?: number;
  yes_price_before: number;
  yes_price_after: number;
  created_at: string;
}

export interface Position {
  user_id: string;
  market_id: string;
  market_question?: string;
  market_status?: string;
  market_outcome?: Outcome | null;
  winning_option_id?: string | null;
  yes_shares: number;
  no_shares: number;
  holdings?: Array<{
    option_id: string;
    option_label: string;
    shares: number;
  }>;
}

export interface DisputeRecord {
  market_id: string;
  market_question: string;
  dispute_id: string;
  reason: string;
  created_at: string;
  creator_id: string;
  creator_name: string;
  resolver_id: string;
  resolver_name: string;
  status: MarketStatus;
  outcome?: Outcome | null;
  evidence_url?: string | null;
  resolved_by?: string | null;
  resolved_at?: string | null;
  dispute_deadline?: string | null;
  initial_liquidity: number;
}

export interface ForecastContributor {
  user_id: string;
  name: string;
  email: string;
  is_executive: boolean;
}

export interface ForecastRevision {
  id: string;
  forecast_question_id: string;
  user_id: string;
  user_name?: string;
  probability_bps: number;
  rationale: string;
  created_at: string;
}

export interface ForecastProjection {
  forecast_question_id: string;
  official_probability_bps: number;
  current_risk_bps: number;
  contributor_count: number;
  last_change_bps: number;
  updated_at: string;
}

export interface ForecastScoreRecord {
  user_id: string;
  user_name?: string;
  probability_bps: number;
  outcome_value: number;
  brier_score: number;
  coverage_score: number;
  revision_count: number;
  created_at: string;
}

export interface ExternalSignalSnapshot {
  id: string;
  forecast_question_id: string;
  source: string;
  probability_bps: number;
  note: string;
  captured_at: string;
}

export interface ForecastQuestion {
  id: string;
  title: string;
  description: string;
  program: string;
  owner_id: string;
  owner_name?: string;
  resolver_id: string;
  resolver_name?: string;
  status: ForecastQuestionStatus;
  outcome?: ForecastQuestionOutcome | null;
  resolution_rule: string;
  rationale_policy_threshold: number;
  linked_market_id?: string | null;
  linked_market_yes_price?: number;
  linked_market_no_price?: number;
  closes_at: string;
  resolves_at: string;
  resolved_at?: string | null;
  evidence_url?: string | null;
  created_at: string;
  updated_at: string;
  projection?: ForecastProjection;
  contributors?: ForecastContributor[];
  latest_rationales?: ForecastRevision[];
  scores?: ForecastScoreRecord[];
  external_signals?: ExternalSignalSnapshot[];
}

export interface ProgramRiskRow {
  question_id: string;
  title: string;
  official_probability_bps: number;
  current_risk_bps: number;
  change_last_7_days_bps: number;
  latest_rationale_excerpts: string[];
  resolution_owner: string;
  closes_at: string;
  resolves_at: string;
  external_signal_probability?: number | null;
  status: ForecastQuestionStatus;
}

export interface ProgramRiskView {
  program: string;
  questions: ProgramRiskRow[];
}

export interface ApiErrorShape {
  error: {
    code: string;
    message: string;
  };
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface WsPriceUpdate {
  type: 'price_update' | 'market_update';
  market_id: string;
  payload: {
    yes_price?: number;
    no_price?: number;
    options?: Array<{
      option_id: string;
      label: string;
      probability_bps: number;
    }>;
    last_trade?: {
      side: TradeSide;
      option_id?: string | null;
      option_label?: string;
      shares: number;
      cost: number;
    };
  };
}
