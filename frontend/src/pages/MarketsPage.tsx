import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError, getWsUrl } from '../lib/api';
import { categories, formatBpsPercent, formatDate, getMarketOptions, getPositionForMarket, marketLeader, partitionMarkets, sentimentClass, sentimentLabel } from '../lib/utils';
import { useAuth } from '../context/AuthContext';
import type { Market, Position, WsPriceUpdate } from '../types';
import { MarketProbabilityChart, marketOptionColor } from '../components/MarketProbabilityChart';
import { MarketRow } from '../components/MarketRow';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader, StatusPill } from '../components/ui';

export function MarketsPage() {
  const { token } = useAuth();
  const [markets, setMarkets] = useState<Market[]>([]);
  const [standoutMarketDetail, setStandoutMarketDetail] = useState<Market | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState('');
  const [viewMode, setViewMode] = useState<'compact' | 'expanded'>('compact');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [liveState, setLiveState] = useState<'connecting' | 'live' | 'offline'>('connecting');
  const [standoutIndex, setStandoutIndex] = useState(0);

  useEffect(() => {
    if (!token) {
      return;
    }

    let isMounted = true;
    setIsLoading(true);
    setError(null);

    Promise.all([
      api.listMarkets(token, { category: category || undefined, status: status || undefined }),
      api.getPositions(token),
    ])
      .then(([nextMarkets, nextPositions]) => {
        if (!isMounted) {
          return;
        }
        setMarkets(nextMarkets || []);
        setPositions(nextPositions || []);
      })
      .catch((err) => {
        if (!isMounted) {
          return;
        }
        setError(err instanceof ApiError ? err.message : 'Could not load markets right now.');
      })
      .finally(() => {
        if (isMounted) {
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [token, category, status]);

  useEffect(() => {
    if (!token) {
      return;
    }
    const socket = new WebSocket(getWsUrl(token));
    socket.onopen = () => setLiveState('live');
    socket.onerror = () => setLiveState('offline');
    socket.onclose = () => setLiveState('offline');
    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as WsPriceUpdate;
      if (message.type !== 'price_update') {
        return;
      }
      setMarkets((current) =>
        current.map((market) =>
          market.id === message.market_id
            ? {
                ...market,
                yes_price: message.payload.yes_price ?? market.yes_price,
                no_price: message.payload.no_price ?? market.no_price,
              }
            : market,
        ),
      );
    };
    return () => socket.close();
  }, [token]);

  const sections = useMemo(() => partitionMarkets(markets), [markets]);
  const standoutMarkets = useMemo(() => {
    const openMarkets = markets.filter((market) => market.status === 'open');
    const ranked = [...openMarkets].sort((a, b) => {
      const aVolume = (a.options || []).reduce((sum, option) => sum + option.collateral, a.initial_liquidity);
      const bVolume = (b.options || []).reduce((sum, option) => sum + option.collateral, b.initial_liquidity);
      return bVolume - aVolume;
    });
    const multiOption = ranked.filter((market) => (market.options || []).length > 2);
    return (multiOption.length > 0 ? multiOption : ranked).slice(0, 3);
  }, [markets]);
  const standoutMarket = standoutMarkets[standoutIndex] || null;

  useEffect(() => {
    setStandoutIndex((current) => {
      if (standoutMarkets.length === 0) {
        return 0;
      }
      return Math.min(current, standoutMarkets.length - 1);
    });
  }, [standoutMarkets]);

  useEffect(() => {
    if (!token || !standoutMarket) {
      setStandoutMarketDetail(null);
      return;
    }
    let cancelled = false;
    api.getMarket(token, standoutMarket.id)
      .then((market) => {
        if (!cancelled) {
          setStandoutMarketDetail(market);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setStandoutMarketDetail(standoutMarket);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [standoutMarket?.id, token]);

  if (isLoading) {
    return <LoadingState title="Loading markets" copy="Pulling together what needs attention now, plus your current positions." />;
  }

  if (error) {
    return <ErrorState title="Could not load markets" copy={error} action={<button className="primary-button" onClick={() => window.location.reload()}>Try again</button>} />;
  }

  return (
    <div className="stack-lg">
      <Card className="controls-card">
        <div className="controls-grid">
          <label>
            Category
            <select className="input" value={category} onChange={(event) => setCategory(event.target.value)}>
              <option value="">All categories</option>
              {categories.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
          <label>
            Status
            <select className="input" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">All statuses</option>
              <option value="open">Open</option>
              <option value="resolved">Resolved</option>
              <option value="disputed">Disputed</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </label>
          <div>
            <span className="field-label">View</span>
            <div className="trade-toggle view-toggle">
              <button className={viewMode === 'compact' ? 'trade-toggle-active' : ''} type="button" onClick={() => setViewMode('compact')}>
                Compact
              </button>
              <button className={viewMode === 'expanded' ? 'trade-toggle-active' : ''} type="button" onClick={() => setViewMode('expanded')}>
                Expanded
              </button>
            </div>
          </div>
        </div>
      </Card>

      <section className="stack-md">
        <SectionHeader title="Market movers" />
        <div className="ticker-ribbon">
          {markets.slice(0, 6).map((market) => {
            const leader = marketLeader(market);
            return (
              <Link key={`${market.id}-ticker`} to={`/markets/${market.id}`} className="activity-link">
                <Card className="ticker-card ticker-card-compact">
                  <div className="row row-wrap ticker-card-top">
                    <strong className="ticker-card-title">{market.question}</strong>
                    <span className={`position-chip ${sentimentClass(market)}`}>{sentimentLabel(market)}</span>
                  </div>
                  <div className="ticker-card-grid ticker-card-grid-full">
                    {getMarketOptions(market).map((option) => (
                      <div className="ticker-stat" key={option.id}>
                        <span>{option.label}</span>
                        <strong>{formatBpsPercent(option.probability_bps)}</strong>
                      </div>
                    ))}
                  </div>
                  <small>{leader ? `${leader.label} leads · closes ${formatDate(market.closes_at)}` : 'No signal yet'}</small>
                </Card>
              </Link>
            );
          })}
        </div>
      </section>

      {(standoutMarketDetail || standoutMarket) ? <StandoutMarketCard market={standoutMarketDetail || standoutMarket!} index={standoutIndex} total={standoutMarkets.length} onPrevious={() => setStandoutIndex((current) => (current === 0 ? standoutMarkets.length - 1 : current - 1))} onNext={() => setStandoutIndex((current) => (current === standoutMarkets.length - 1 ? 0 : current + 1))} /> : null}

      <MarketSection
        title="Closing Soon"
        copy="Questions that need attention before the window closes."
        markets={sections.closingSoon}
        positions={positions}
        expanded={viewMode === 'expanded'}
      />

      <MarketSection
        title="Open"
        copy="The broader queue of active company forecasts."
        markets={sections.open}
        positions={positions}
        expanded={viewMode === 'expanded'}
      />

      <MarketSection
        title="Recently Resolved"
        copy="Past outcomes, evidence, and any markets still in dispute review."
        markets={sections.resolved}
        positions={positions}
        expanded={viewMode === 'expanded'}
        emptyCopy="No markets have resolved yet. Once outcomes are recorded, evidence and dispute status will show here."
      />
    </div>
  );
}

function MarketSection({
  title,
  copy,
  markets,
  positions,
  expanded,
  emptyCopy,
}: {
  title: string;
  copy: string;
  markets: Market[];
  positions: Position[];
  expanded: boolean;
  emptyCopy?: string;
}) {
  return (
    <section className="stack-md">
      <SectionHeader title={title} copy={copy} />
      {markets.length === 0 ? (
        <EmptyState
          title={`Nothing in ${title.toLowerCase()} right now`}
          copy={emptyCopy || 'No markets are open right now. You can create one when your team has a question worth forecasting.'}
          action={<Link className="secondary-button" to="/create">Create market</Link>}
        />
      ) : (
        <div className="stack-sm">
          {markets.map((market) => (
            <MarketRow key={market.id} market={market} position={getPositionForMarket(positions, market.id)} expanded={expanded} />
          ))}
        </div>
      )}
    </section>
  );
}

function StandoutMarketCard({
  market,
  index,
  total,
  onPrevious,
  onNext,
}: {
  market: Market;
  index: number;
  total: number;
  onPrevious: () => void;
  onNext: () => void;
}) {
  const options = getMarketOptions(market).slice().sort((a, b) => b.probability_bps - a.probability_bps);
  const palette = ['#4ccd82', '#ff6861', '#86b8ff', '#f3ae38', '#b88cff'];

  return (
    <Card className="standout-market-card">
      <div className="standout-header">
        <div>
          <p className="eyebrow">Most active prediction</p>
          <h3>{market.question}</h3>
          <p>Inspired by `main.png`: one standout chart that makes the current probability structure and option spread legible at a glance.</p>
        </div>
        <div className="standout-meta">
          <span className="position-chip">{index + 1} / {total}</span>
          <span className={`position-chip ${sentimentClass(market)}`}>{sentimentLabel(market)}</span>
          {total > 1 ? <button className="ghost-button" type="button" onClick={onPrevious}>Previous slide</button> : null}
          {total > 1 ? <button className="ghost-button" type="button" onClick={onNext}>Next slide</button> : null}
          <Link className="secondary-button" to={`/markets/${market.id}`}>Open market</Link>
        </div>
      </div>

      <div className="standout-layout">
        <div className="standout-legend">
        {options.map((option, index) => (
          <div className="standout-row" key={option.id}>
            <div className="standout-option-label">
              <span className="legend-dot" style={{ background: marketOptionColor(option.label, palette[index % palette.length]) }} />
              <strong>{option.label}</strong>
            </div>
            <strong>{formatBpsPercent(option.probability_bps)}</strong>
            </div>
          ))}
        </div>

        <div className="standout-chart-shell">
          <MarketProbabilityChart
            market={market}
            standout
            title="Most active prediction"
            copy="Inspired by `main.png`: a single standout chart showing the full option spread over time."
          />
        </div>
      </div>
    </Card>
  );
}
