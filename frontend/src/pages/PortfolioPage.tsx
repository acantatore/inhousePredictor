import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import { formatPercent, formatPoints } from '../lib/utils';
import type { MarketStatus, Position } from '../types';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader, StatCard, StatusPill } from '../components/ui';

const STARTING_BALANCE = 10000;

export function PortfolioPage() {
  const { token, user } = useAuth();
  const [positions, setPositions] = useState<Position[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      return;
    }
    api.getPositions(token)
      .then((nextPositions) => setPositions(nextPositions || []))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load your portfolio.'))
      .finally(() => setIsLoading(false));
  }, [token]);

  const stats = useMemo(() => derivePortfolioStats(positions, user?.balance ?? STARTING_BALANCE), [positions, user?.balance]);
  const awaiting = positions.filter((position) => ['open', 'closed', 'disputed'].includes(position.market_status || ''));
  const resolved = positions.filter((position) => (position.market_status || '') === 'resolved');

  if (isLoading) {
    return <LoadingState title="Loading portfolio" copy="Bringing together your open positions, awaiting resolution markets, and recent outcomes." />;
  }

  if (error) {
    return <ErrorState title="Could not load your portfolio" copy={error} />;
  }

  return (
    <div className="stack-lg">
      <section className="stats-grid">
        <StatCard label="Awaiting resolution" value={`${stats.awaitingResolution}`} detail="Markets where your forecast is still waiting on a final outcome." />
        <StatCard label="Points currently committed" value={formatPoints(stats.committedPoints)} detail="The total shares still tied up in open, closed, or disputed markets." />
        <StatCard label="Resolved calls" value={`${stats.resolvedCount}`} detail="Resolved markets where you held a view long enough to learn from the outcome." />
        <StatCard label="Forecast accuracy" value={formatPercent(stats.hitRate)} detail="Based on resolved markets where your larger holding leaned YES or NO." />
      </section>

      <section className="stats-grid">
        <StatCard label="Current balance" value={formatPoints(user?.balance ?? 0)} detail="What you can spend right now across new questions and markets." />
        <StatCard label="Points earned / lost" value={`${stats.netPoints >= 0 ? '+' : ''}${formatPoints(stats.netPoints)}`} detail="Compared with the starting balance of 10,000 points." />
        <StatCard label="Winning calls" value={`${stats.correctCount}`} detail="Resolved markets where your larger holding matched the recorded outcome." />
        <StatCard label="Incorrect calls" value={`${stats.incorrectCount}`} detail="Resolved markets where your larger holding missed the recorded outcome." />
      </section>

      <Card>
        <SectionHeader title="Awaiting Resolution" copy="These are the markets where you still have something to learn from the eventual outcome." />
        {awaiting.length === 0 ? (
          <EmptyState title="Nothing awaiting resolution" copy="You do not have any open or disputed holdings right now. Explore markets when a new team question needs a signal." action={<Link className="secondary-button" to="/markets">Explore markets</Link>} />
        ) : (
          <div className="stack-sm">
            {awaiting.map((position) => {
               return (
                 <Link className="activity-link" key={position.market_id} to={`/markets/${position.market_id}`}>
                   <div className="activity-row activity-row-rich">
                     <div>
                       <strong>{position.market_question || 'Market'}</strong>
                       <small>YES {position.yes_shares.toFixed(1)} / NO {position.no_shares.toFixed(1)}</small>
                     </div>
                     <StatusPill status={(position.market_status || 'open') as MarketStatus} />
                   </div>
                 </Link>
               );
            })}
          </div>
        )}
      </Card>

      <Card>
        <SectionHeader title="Resolved Outcomes" copy="A calm read on what your past calls turned into after evidence and deadlines played out." />
        {resolved.length === 0 ? (
          <EmptyState title="No resolved outcomes yet" copy="Once one of your markets resolves, the outcome will appear here alongside the position you held." />
        ) : (
          <div className="stack-sm">
            {resolved.map((position) => {
               return (
                 <Link className="activity-link" key={position.market_id} to={`/markets/${position.market_id}`}>
                   <div className="activity-row activity-row-rich">
                     <div>
                       <strong>{position.market_question || 'Market'}</strong>
                       <small>Outcome {position.market_outcome?.toUpperCase() || 'Pending'}</small>
                      </div>
                      <span className="mono-copy">YES {position.yes_shares.toFixed(1)} / NO {position.no_shares.toFixed(1)}</span>
                    </div>
                 </Link>
               );
            })}
          </div>
        )}
      </Card>
    </div>
  );
}

function derivePortfolioStats(positions: Position[], currentBalance: number) {
  let awaitingResolution = 0;
  let committedPoints = 0;
  let resolvedCount = 0;
  let correctCount = 0;
  let incorrectCount = 0;

  positions.forEach((position) => {
    const totalShares = position.yes_shares + position.no_shares;
    if (['open', 'closed', 'disputed'].includes(position.market_status || '')) {
      awaitingResolution += 1;
      committedPoints += totalShares;
      return;
    }
    if ((position.market_status || '') === 'resolved') {
      resolvedCount += 1;
      const leaning = position.yes_shares > position.no_shares ? 'yes' : position.no_shares > position.yes_shares ? 'no' : null;
      if (leaning && position.market_outcome && leaning === position.market_outcome) {
        correctCount += 1;
      } else if (leaning && position.market_outcome) {
        incorrectCount += 1;
      }
    }
  });

  return {
    awaitingResolution,
    committedPoints,
    resolvedCount,
    correctCount,
    incorrectCount,
    netPoints: currentBalance - STARTING_BALANCE,
    hitRate: resolvedCount ? correctCount / resolvedCount : 0,
  };
}
