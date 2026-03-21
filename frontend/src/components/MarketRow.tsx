import { Link } from 'react-router-dom';
import type { Market, Position } from '../types';
import { formatDate, getUserLeaning } from '../lib/utils';
import { Card, ProbabilitySplit, StatusPill } from './ui';

export function MarketRow({
  market,
  position,
  expanded,
}: {
  market: Market;
  position?: Position;
  expanded: boolean;
}) {
  const leaning = getUserLeaning(position);

  return (
    <Link className="market-link" to={`/markets/${market.id}`}>
      <Card className={`market-row ${expanded ? 'market-row-expanded' : ''}`}>
        <div>
          <div className="market-row-topline">
            <span className="category-chip">{market.category}</span>
            <StatusPill status={market.status} />
            {leaning ? <span className="position-chip">Your position: {leaning}</span> : null}
          </div>
          <h3>{market.question}</h3>
          {expanded ? <p className="market-row-copy">{market.description}</p> : null}
          <div className="market-row-meta">
            <span>Resolver {market.resolver_name || 'Assigned teammate'}</span>
            <span>Closes {formatDate(market.closes_at)}</span>
          </div>
        </div>
        <div className="market-row-side">
          <ProbabilitySplit yesPrice={market.yes_price} noPrice={market.no_price} />
        </div>
      </Card>
    </Link>
  );
}
