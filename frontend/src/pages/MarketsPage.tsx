import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError, getWsUrl } from '../lib/api';
import { categories, getPositionForMarket, partitionMarkets } from '../lib/utils';
import { useAuth } from '../context/AuthContext';
import type { Market, Position, WsPriceUpdate } from '../types';
import { MarketRow } from '../components/MarketRow';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader, StatusPill } from '../components/ui';

export function MarketsPage() {
  const { token } = useAuth();
  const [markets, setMarkets] = useState<Market[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState('');
  const [viewMode, setViewMode] = useState<'compact' | 'expanded'>('compact');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [liveState, setLiveState] = useState<'connecting' | 'live' | 'offline'>('connecting');

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
        setMarkets(nextMarkets);
        setPositions(nextPositions);
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

  if (isLoading) {
    return <LoadingState title="Loading markets" copy="Pulling together what needs attention now, plus your current positions." />;
  }

  if (error) {
    return <ErrorState title="Could not load markets" copy={error} action={<button className="primary-button" onClick={() => window.location.reload()}>Try again</button>} />;
  }

  return (
    <div className="stack-lg">
      <Card className="hero-card">
        <div>
          <p className="eyebrow">Markets</p>
          <h2>Forecast the questions your team already talks about.</h2>
          <p>
            Browse what is closing soon, see the current crowd signal, and jump into a market when you have enough context to contribute.
          </p>
        </div>
        <div className="hero-side">
          <span className={`live-pill live-${liveState}`}>{liveState === 'live' ? 'Live updates connected' : liveState === 'connecting' ? 'Connecting live updates...' : 'Live updates offline'}</span>
          <Link className="primary-button" to="/create">
            Create market
          </Link>
        </div>
      </Card>

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
