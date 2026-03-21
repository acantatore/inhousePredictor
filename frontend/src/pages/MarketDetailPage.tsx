import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api, ApiError, getWsUrl } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import { formatBpsPercent, formatDate, formatPoints, getMarketOptions, getPositionForMarket, marketLeader, sentimentClass, sentimentLabel } from '../lib/utils';
import type { Market, Position, Trade, WsPriceUpdate } from '../types';
import { TradeTicket } from '../components/TradeTicket';
import { Card, DeadlineBlock, EmptyState, ErrorState, InlineNotice, LoadingState, SectionHeader, StatusPill } from '../components/ui';

export function MarketDetailPage() {
  const { id = '' } = useParams();
  const { token, user, refreshUser } = useAuth();
  const [market, setMarket] = useState<Market | null>(null);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resolveMessage, setResolveMessage] = useState<string | null>(null);
  const [liveState, setLiveState] = useState<'live' | 'offline'>('offline');

  const load = useCallback(async () => {
    if (!token || !id) {
      return;
    }
    setError(null);
    try {
      const [nextMarket, nextTrades, nextPositions] = await Promise.all([
        api.getMarket(token, id),
        api.getMarketTrades(token, id),
        api.getPositions(token),
      ]);
      setMarket(nextMarket);
      setTrades(nextTrades || []);
      setPositions(nextPositions || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not load this market right now.');
    } finally {
      setIsLoading(false);
    }
  }, [id, token]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!token || !id) {
      return;
    }
    const socket = new WebSocket(getWsUrl(token, id));
    socket.onopen = () => setLiveState('live');
    socket.onerror = () => setLiveState('offline');
    socket.onclose = () => setLiveState('offline');
    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as WsPriceUpdate;
      if (message.type !== 'price_update') {
        return;
      }
      const lastTrade = message.payload.last_trade;
      setMarket((current) =>
        current
          ? {
              ...current,
              yes_price: message.payload.yes_price ?? current.yes_price,
              no_price: message.payload.no_price ?? current.no_price,
              options: message.payload.options
                ? (current.options || []).map((option) => {
                    const updated = message.payload.options?.find((item) => item.option_id === option.id);
                    return updated ? { ...option, probability_bps: updated.probability_bps } : option;
                  })
                : current.options,
            }
          : current,
      );
      if (lastTrade) {
        setTrades((current) => [
          {
            id: `${Date.now()}`,
            user_id: '',
            market_id: id,
            side: lastTrade.side,
            shares: lastTrade.shares,
            cost: lastTrade.cost,
            option_id: lastTrade.option_id || undefined,
            option_label: lastTrade.option_label,
            yes_price_before: 0,
            yes_price_after: message.payload.yes_price ?? 0,
            created_at: new Date().toISOString(),
          },
          ...current,
        ].slice(0, 8));
      }
    };
    return () => socket.close();
  }, [id, token]);

  const position = useMemo(() => getPositionForMarket(positions, id), [id, positions]);
  const options = useMemo(() => (market ? getMarketOptions(market) : []), [market]);
  const leader = market ? marketLeader(market) : null;

  if (isLoading) {
    return <LoadingState title="Loading market detail" copy="Bringing together timing, evidence, activity, and the trade panel." />;
  }

  if (error || !market || !token || !user) {
    return <ErrorState title="Could not load this market" copy={error || 'This market may be unavailable.'} action={<Link className="secondary-button" to="/markets">Back to markets</Link>} />;
  }

  const canResolve = market.resolver_id === user.id && market.status === 'open';
  const canDispute = market.status === 'resolved' && Boolean(market.dispute_deadline) && new Date(market.dispute_deadline!).getTime() > Date.now();

  async function handleAfterTrade() {
    await Promise.all([load(), refreshUser()]);
  }

  return (
    <div className="detail-layout">
      <div className="stack-lg detail-main">
        <Card className="market-summary-card">
          <div className="market-row-topline">
            <span className="category-chip">{market.category}</span>
            <StatusPill status={market.status} />
            <span className={`live-pill live-${liveState}`}>{liveState === 'live' ? 'Live signal on' : 'Live signal offline'}</span>
            {leader ? <span className={`position-chip ${sentimentClass(market)}`}>{leader.label} · {sentimentLabel(market)}</span> : null}
          </div>
          <h2>{market.question}</h2>
          <p>{market.description}</p>
          <div className="info-grid">
            {options.map((option) => (
              <Card key={option.id} className="stat-card">
                <span>{option.label}</span>
                <strong>{formatBpsPercent(option.probability_bps)}</strong>
                <small>{formatPoints(option.collateral)} committed</small>
              </Card>
            ))}
          </div>
          <ProbabilityChart market={market} />
          <div className="deadline-grid">
            <DeadlineBlock label="Closes" value={market.closes_at} />
            <DeadlineBlock label="Resolves" value={market.resolves_at} />
            <DeadlineBlock label="Dispute deadline" value={market.dispute_deadline} />
          </div>
        </Card>

        <div className="info-grid">
          <Card>
            <SectionHeader title="Trust and timeline" copy="Resolver identity, evidence, and dispute state stay visible here." />
            <div className="stack-sm">
              <p><strong>Creator:</strong> {market.creator_name || 'Unknown teammate'}</p>
              <p><strong>Resolver:</strong> {market.resolver_name || 'Assigned teammate'}</p>
              <p><strong>Initial liquidity:</strong> {formatPoints(market.initial_liquidity)}</p>
              {options.length ? <p><strong>Options:</strong> {options.map((option) => option.label).join(', ')}</p> : null}
              {market.evidence_url ? (
                <p><strong>Evidence:</strong> <a href={market.evidence_url} target="_blank" rel="noreferrer">Open evidence link</a></p>
              ) : (
                <InlineNotice tone="warning">Evidence will appear here once resolution is recorded.</InlineNotice>
              )}
              {market.status === 'disputed' ? <InlineNotice tone="warning">This market is under dispute review. Trading is closed while an admin checks the outcome.</InlineNotice> : null}
              {market.outcome ? <InlineNotice tone="success">Outcome recorded: {market.outcome.toUpperCase()}.</InlineNotice> : null}
            </div>
          </Card>

          <Card>
            <SectionHeader title="Your activity" copy="Private to you: holdings and what action is still available." />
            {position ? (
              <div className="stack-sm">
                {position.holdings?.length
                  ? position.holdings.map((holding) => <p key={holding.option_id}><strong>{holding.option_label} shares:</strong> {holding.shares.toFixed(1)}</p>)
                  : [<p key="yes"><strong>YES shares:</strong> {position.yes_shares.toFixed(1)}</p>, <p key="no"><strong>NO shares:</strong> {position.no_shares.toFixed(1)}</p>]
                }
              </div>
            ) : (
              <EmptyState title="No position yet" copy="You have not bought into this market yet. Review the context, then act when you are ready." />
            )}
            <div className="stack-sm">
              {canResolve ? <Link className="secondary-button" to={`/markets/${market.id}/resolve`}>Resolve market</Link> : null}
              {canDispute ? <Link className="secondary-button" to={`/markets/${market.id}/dispute`}>Submit dispute</Link> : null}
            </div>
          </Card>
        </div>

        <Card>
          <SectionHeader title="Activity summary" copy="Recent anonymous trade activity helps you understand momentum without turning the page into a public trade tape." />
          {trades.length === 0 ? (
            <EmptyState title="No recent activity yet" copy="This market has not seen recent trades. The first new signal can still be useful when the question matters." />
          ) : (
            <div className="stack-sm">
              {trades.slice(0, 6).map((trade, index) => (
                <div className="activity-row" key={`${trade.id}-${index}`}>
                  <span>{trade.side === 'yes' ? 'YES buy' : 'NO buy'}</span>
                  <strong>{formatPoints(trade.cost)}</strong>
                  <small>{formatDate(trade.created_at)}</small>
                </div>
              ))}
            </div>
          )}
        </Card>

        {resolveMessage ? <InlineNotice tone="success">{resolveMessage}</InlineNotice> : null}
      </div>

      <aside className="detail-side">
        <TradeTicket market={market} token={token} user={user} onTraded={handleAfterTrade} />
      </aside>
    </div>
  );
}

function ProbabilityChart({ market }: { market: Market }) {
  if (!market.snapshots || market.snapshots.length < 2) {
    return <InlineNotice tone="neutral">Live line chart will appear after this market has more probability history.</InlineNotice>;
  }
  const options = getMarketOptions(market);
  const width = 640;
  const height = 220;
  const padding = 20;
  const maxX = Math.max(market.snapshots.length - 1, 1);
  const colors = ['#2f8f5b', '#b8574f', '#587081', '#b77b27', '#8f6ab0'];

  return (
    <Card className="chart-card">
      <SectionHeader title="Live probability chart" copy="Track how option probabilities moved over time, not just where they are now." />
      <svg viewBox={`0 0 ${width} ${height}`} className="market-chart" role="img" aria-label="Market probability chart">
        {options.map((option, optionIndex) => {
          const points = market.snapshots!.map((snapshot, snapshotIndex) => {
            const total = Object.values(snapshot.points || {}).reduce((sum, value) => sum + value, 0) || 1;
            const probability = (snapshot.points?.[option.label] || 0) / total;
            const x = padding + ((width - padding * 2) * snapshotIndex) / maxX;
            const y = height - padding - probability * (height - padding * 2);
            return `${x},${y}`;
          }).join(' ');
          return <polyline key={option.id} fill="none" stroke={colors[optionIndex % colors.length]} strokeWidth="3" points={points} />;
        })}
      </svg>
      <div className="row row-wrap">
        {options.map((option, optionIndex) => <span className="position-chip" key={option.id} style={{ borderColor: colors[optionIndex % colors.length] }}>{option.label}</span>)}
      </div>
    </Card>
  );
}
