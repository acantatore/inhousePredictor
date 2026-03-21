import type { ForecastQuestion, ForecastQuestionStatus, Market, MarketOption, MarketStatus, Position } from '../types';

export const categories = [
  { value: 'people', label: 'People' },
  { value: 'okrs', label: 'OKRs' },
  { value: 'slas', label: 'SLAs' },
  { value: 'financials', label: 'Financials' },
  { value: 'general', label: 'General' },
];

export const statuses: Array<{ value: MarketStatus; label: string }> = [
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'disputed', label: 'Disputed' },
  { value: 'cancelled', label: 'Cancelled' },
];

export function formatPercent(value: number) {
  return `${Math.round(value * 100)}%`;
}

export function formatBpsPercent(value: number) {
  return `${(value / 100).toFixed(value % 100 === 0 ? 0 : 1)}%`;
}

export const forecastStatuses: Array<{ value: ForecastQuestionStatus; label: string }> = [
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'cancelled', label: 'Cancelled' },
];

export function formatPoints(value: number) {
  return `${new Intl.NumberFormat().format(Math.round(value))} pts`;
}

export function formatDate(value?: string | null) {
  if (!value) {
    return 'Not available';
  }
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(new Date(value));
}

export function formatRelativeTime(value?: string | null) {
  if (!value) {
    return 'No date set';
  }
  const diff = new Date(value).getTime() - Date.now();
  const absMinutes = Math.round(Math.abs(diff) / 60000);
  if (absMinutes < 60) {
    return diff >= 0 ? `in ${absMinutes}m` : `${absMinutes}m ago`;
  }
  const absHours = Math.round(absMinutes / 60);
  if (absHours < 48) {
    return diff >= 0 ? `in ${absHours}h` : `${absHours}h ago`;
  }
  const absDays = Math.round(absHours / 24);
  return diff >= 0 ? `in ${absDays}d` : `${absDays}d ago`;
}

export function getPositionForMarket(positions: Position[], marketId: string) {
  return positions.find((position) => position.market_id === marketId);
}

export function getUserLeaning(position?: Position) {
  if (!position) {
    return null;
  }
  if (position.yes_shares > position.no_shares) {
    return 'YES';
  }
  if (position.no_shares > position.yes_shares) {
    return 'NO';
  }
  return 'Mixed';
}

export function getTradeDisabledReason(market: Market, viewerId?: string, balance?: number, cost?: number) {
  if (new Date(market.closes_at).getTime() <= Date.now()) {
    return 'Trading is closed for this market.';
  }
  if (market.status !== 'open') {
    return 'Trading is closed for this market.';
  }
  if (!viewerId) {
    return 'You need to sign in to trade.';
  }
  if (market.creator_id === viewerId) {
    return 'You cannot trade a market you created.';
  }
  if (typeof balance === 'number' && typeof cost === 'number' && cost > balance) {
    return 'You do not have enough points for this trade.';
  }
  return null;
}

export function getMarketOptions(market: Market): MarketOption[] {
  if (market.options && market.options.length > 0) {
    return market.options;
  }
  return [
    { id: `${market.id}-yes`, market_id: market.id, label: 'YES', sort_order: 0, probability_bps: Math.round(market.yes_price * 10000), collateral: 0 },
    { id: `${market.id}-no`, market_id: market.id, label: 'NO', sort_order: 1, probability_bps: Math.round(market.no_price * 10000), collateral: 0 },
  ];
}

export function marketLeader(market: Market) {
  const options = getMarketOptions(market).slice().sort((a, b) => b.probability_bps - a.probability_bps);
  return options[0] || null;
}

export function sentimentLabel(market: Market) {
  const leader = marketLeader(market);
  if (!leader) return 'Flat';
  const direction = market.sentiment === 'down' ? 'Down' : market.sentiment === 'up' ? 'Up' : 'Flat';
  return `${direction} ${formatBpsPercent(Math.abs(market.change_bps || 0))}`;
}

export function sentimentClass(market: Market) {
  if (market.sentiment === 'up') return 'sentiment-up';
  if (market.sentiment === 'down') return 'sentiment-down';
  return 'sentiment-flat';
}

export function isClosingSoon(market: Market) {
  if (market.status !== 'open') {
    return false;
  }
  return new Date(market.closes_at).getTime() - Date.now() < 1000 * 60 * 60 * 48;
}

export function partitionMarkets(markets: Market[]) {
  const closingSoon = markets
    .filter((market) => isClosingSoon(market))
    .sort((a, b) => new Date(a.closes_at).getTime() - new Date(b.closes_at).getTime());

  const open = markets
    .filter((market) => market.status === 'open' && !isClosingSoon(market))
    .sort((a, b) => new Date(a.closes_at).getTime() - new Date(b.closes_at).getTime());

  const resolved = markets
    .filter((market) => market.status === 'resolved' || market.status === 'disputed' || market.status === 'cancelled')
    .sort((a, b) => new Date(b.resolved_at || b.created_at).getTime() - new Date(a.resolved_at || a.created_at).getTime());

  return { closingSoon, open, resolved };
}

export function estimateShares(cost: number, probability: number) {
  if (!cost || probability <= 0) {
    return 0;
  }
  return cost / Math.max(probability, 0.05);
}

export function isForecastClosed(question: ForecastQuestion) {
  return question.status !== 'open' || new Date(question.closes_at).getTime() <= Date.now();
}

export function derivePortfolioStats(markets: Market[], positions: Position[]) {
  const byMarket = new Map(markets.map((market) => [market.id, market]));
  let awaitingResolution = 0;
  let committedPoints = 0;
  let resolvedCount = 0;
  let correctCount = 0;

  positions.forEach((position) => {
    const market = byMarket.get(position.market_id);
    if (!market) {
      return;
    }

    if (market.status === 'open' || market.status === 'closed' || market.status === 'disputed') {
      awaitingResolution += 1;
      committedPoints += position.yes_shares * market.yes_price + position.no_shares * market.no_price;
      return;
    }

    if (market.status === 'resolved' && market.outcome && market.outcome !== 'cancelled') {
      resolvedCount += 1;
      const leaning = position.yes_shares > position.no_shares ? 'yes' : position.no_shares > position.yes_shares ? 'no' : null;
      if (leaning && leaning === market.outcome) {
        correctCount += 1;
      }
    }
  });

  return {
    awaitingResolution,
    committedPoints,
    resolvedCount,
    hitRate: resolvedCount ? correctCount / resolvedCount : 0,
  };
}
