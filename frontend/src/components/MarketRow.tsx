import { Link } from 'react-router-dom';
import type { Market, Position } from '../types';
import { formatBpsPercent, formatDate, getMarketOptions, getUserLeaning, marketLeader, sentimentClass, sentimentLabel } from '../lib/utils';
import { Card, StatusPill } from './ui';

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
  const options = getMarketOptions(market);
  const leader = marketLeader(market);

  return (
    <Link className="market-link" to={`/markets/${market.id}`}>
      <Card className={`market-row ${expanded ? 'market-row-expanded' : ''}`}>
        <div>
          <div className="market-row-topline">
            <span className="category-chip">{market.category}</span>
            <StatusPill status={market.status} />
            {leaning ? <span className="position-chip">Your position: {leaning}</span> : null}
            {leader ? <span className={`position-chip ${sentimentClass(market)}`}>{leader.label} · {sentimentLabel(market)}</span> : null}
          </div>
          <h3>{market.question}</h3>
          {expanded ? <p className="market-row-copy">{market.description}</p> : null}
          <div className="market-row-meta">
            <span>Resolver {market.resolver_name || 'Assigned teammate'}</span>
            <span>Closes {formatDate(market.closes_at)}</span>
          </div>
        </div>
        <div className="market-row-side">
          <div className="stack-sm market-option-stack">
            {options.slice(0, 3).map((option) => (
              <div className="activity-row" key={option.id}>
                <span>{option.label}</span>
                <strong>{formatBpsPercent(option.probability_bps)}</strong>
              </div>
            ))}
          </div>
        </div>
      </Card>
    </Link>
  );
}
