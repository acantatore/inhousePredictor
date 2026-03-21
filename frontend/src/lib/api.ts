import type {
  DisputeAction,
  DisputeRecord,
  ExternalSignalSnapshot,
  ForecastQuestion,
  ForecastQuestionOutcome,
  ForecastRevision,
  ForecastScoreRecord,
  LoginResponse,
  Market,
  Outcome,
  Position,
  ProgramRiskView,
  Trade,
  TradeSide,
  User,
  UserSummary,
} from '../types';

const DEFAULT_API_BASE_URL = '/api';

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

function trimSlash(value: string) {
  return value.replace(/\/$/, '');
}

export const API_BASE_URL = trimSlash(
  import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL,
);

export function getWsUrl(token: string, marketId?: string) {
  const localDevHost = ['localhost', '127.0.0.1'].includes(window.location.hostname);
  const wsOrigin = localDevHost ? 'ws://127.0.0.1:8080' : window.location.origin.replace(/^http/, 'ws');
  const wsBase = API_BASE_URL.startsWith('http') ? API_BASE_URL.replace(/^http/, 'ws') : `${wsOrigin}${API_BASE_URL}`;
  const url = new URL(`${wsBase}/ws`);
  url.searchParams.set('token', token);
  if (marketId) {
    url.searchParams.set('market_id', marketId);
  }
  return url.toString();
}

async function request<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(init.headers || {});
  headers.set('Content-Type', 'application/json');
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
    credentials: 'include',
  });

  const isJson = response.headers.get('content-type')?.includes('application/json');
  const body = isJson ? await response.json() : null;

  if (!response.ok) {
    const message = body?.error?.message || 'Something went wrong.';
    const code = body?.error?.code || 'internal_error';
    throw new ApiError(message, code, response.status);
  }

  return (body?.data ?? null) as T;
}

export const api = {
  register: (payload: { name: string; email: string; password: string }) =>
    request<User>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  login: (payload: { email: string; password: string }) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  me: (token: string) => request<User>('/me', {}, token),

  listUsers: (token: string, query?: { q?: string; limit?: number }) => {
    const search = new URLSearchParams();
    if (query?.q) {
      search.set('q', query.q);
    }
    if (typeof query?.limit === 'number') {
      search.set('limit', String(query.limit));
    }
    const suffix = search.toString() ? `?${search.toString()}` : '';
    return request<UserSummary[]>(`/users${suffix}`, {}, token);
  },

  listMarkets: (token: string, query?: { category?: string; status?: string }) => {
    const search = new URLSearchParams();
    if (query?.category) {
      search.set('category', query.category);
    }
    if (query?.status) {
      search.set('status', query.status);
    }
    const suffix = search.toString() ? `?${search.toString()}` : '';
    return request<Market[]>(`/markets${suffix}`, {}, token);
  },

  getMarket: (token: string, marketId: string) => request<Market>(`/markets/${marketId}`, {}, token),

  createMarket: (
    token: string,
    payload: {
      question: string;
      description: string;
      category: string;
      resolver_id: string;
      initial_liquidity: number;
      closes_at: string;
      resolves_at: string;
    },
  ) =>
    request<Market>(`/markets`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }, token),

  trade: (token: string, marketId: string, payload: { side: TradeSide; cost: number }) =>
    request<Trade>(`/markets/${marketId}/trade`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }, token),

  resolveMarket: (token: string, marketId: string, payload: { outcome: Outcome; evidence_url: string }) =>
    request<{ status: string }>(`/markets/${marketId}/resolve`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }, token),

  disputeMarket: (token: string, marketId: string, payload: { reason: string }) =>
    request<{ status: string }>(`/markets/${marketId}/dispute`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }, token),

  getMarketTrades: (token: string, marketId: string) => request<Trade[]>(`/markets/${marketId}/trades`, {}, token),

  getPositions: (token: string) => request<Position[]>('/positions', {}, token),

  listDisputes: (token: string) => request<DisputeRecord[]>('/admin/disputes', {}, token),

  listForecastQuestions: (token: string, query?: { program?: string; status?: string }) => {
    const search = new URLSearchParams();
    if (query?.program) search.set('program', query.program);
    if (query?.status) search.set('status', query.status);
    const suffix = search.toString() ? `?${search.toString()}` : '';
    return request<ForecastQuestion[]>(`/forecast-questions${suffix}`, {}, token);
  },

  getForecastQuestion: (token: string, questionId: string) => request<ForecastQuestion>(`/forecast-questions/${questionId}`, {}, token),

  createForecastQuestion: (token: string, payload: {
    title: string;
    description: string;
    program: string;
    resolver_id: string;
    resolution_rule: string;
    closes_at: string;
    resolves_at: string;
    contributor_ids: string[];
  }) => request<ForecastQuestion>('/forecast-questions', { method: 'POST', body: JSON.stringify(payload) }, token),

  submitForecast: (token: string, questionId: string, payload: { probability_bps: number; rationale: string }) =>
    request<ForecastQuestion>(`/forecast-questions/${questionId}/forecasts`, { method: 'POST', body: JSON.stringify(payload) }, token),

  resolveForecastQuestion: (token: string, questionId: string, payload: { outcome: ForecastQuestionOutcome; evidence_url: string }) =>
    request<ForecastQuestion>(`/forecast-questions/${questionId}/resolve`, { method: 'POST', body: JSON.stringify(payload) }, token),

  getProgramRisk: (token: string, program: string) => request<ProgramRiskView>(`/programs/${encodeURIComponent(program)}/risk`, {}, token),

  createExternalSignal: (token: string, questionId: string, payload: { source: string; probability_bps: number; note: string }) =>
    request<ExternalSignalSnapshot>(`/forecast-questions/${questionId}/external-signals`, { method: 'POST', body: JSON.stringify(payload) }, token),

  reviewDispute: (
    token: string,
    marketId: string,
    payload: { action: DisputeAction; outcome?: Outcome; evidence_url?: string },
  ) =>
    request<{ status: string }>(`/markets/${marketId}/review-dispute`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }, token),
};
