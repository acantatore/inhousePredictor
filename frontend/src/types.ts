export type Category = 'people' | 'okrs' | 'slas' | 'financials' | 'general';
export type MarketStatus = 'open' | 'closed' | 'resolved' | 'disputed' | 'cancelled';
export type Outcome = 'yes' | 'no' | 'cancelled';
export type TradeSide = 'yes' | 'no';
export type DisputeAction = 'confirm_original' | 'override_outcome' | 'cancel_market';

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
  yes_price: number;
  no_price: number;
}

export interface Trade {
  id: string;
  user_id: string;
  market_id: string;
  side: TradeSide;
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
  yes_shares: number;
  no_shares: number;
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
    last_trade?: {
      side: TradeSide;
      shares: number;
      cost: number;
    };
  };
}
