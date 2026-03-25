import type { Market } from '../types';
import { formatBpsPercent, getMarketOptions } from '../lib/utils';
import { Card, EmptyState, InlineNotice, SectionHeader } from './ui';

const palette = ['#4ccd82', '#ff6861', '#86b8ff', '#f3ae38', '#b88cff'];

export function marketOptionColor(label: string, fallback: string) {
  const upper = label.toUpperCase();
  if (upper === 'YES') return '#4ccd82';
  if (upper === 'NO') return '#ff6861';
  return fallback;
}

export function MarketProbabilityChart({
  market,
  title = 'Live probability chart',
  copy = 'Track how option probabilities moved over time, not just where they are now.',
  standout = false,
}: {
  market: Market;
  title?: string;
  copy?: string;
  standout?: boolean;
}) {
  if (!market.snapshots || market.snapshots.length < 2) {
    if (standout) {
      return (
        <Card className="chart-card">
          <SectionHeader title={title} copy={copy} />
          <EmptyState title="Chart still warming up" copy="Once this market has more live history, the standout chart will show how each option moved over time." />
        </Card>
      );
    }
    return <InlineNotice tone="neutral">Live line chart will appear after this market has more probability history.</InlineNotice>;
  }

  const options = getMarketOptions(market);
  const width = standout ? 720 : 640;
  const height = standout ? 260 : 220;
  const padding = 22;
  const maxX = Math.max(market.snapshots.length - 1, 1);

  const chart = (
    <>
      <svg viewBox={`0 0 ${width} ${height}`} className={`market-chart ${standout ? 'standout-chart' : ''}`} role="img" aria-label="Market probability chart">
        {options.map((option, optionIndex) => {
          const series = market.snapshots!.map((snapshot, snapshotIndex) => {
            const total = Object.values(snapshot.points || {}).reduce((sum, value) => sum + value, 0) || 1;
            const probability = (snapshot.points?.[option.label] || 0) / total;
            const x = padding + ((width - padding * 2) * snapshotIndex) / maxX;
            const y = height - padding - probability * (height - padding * 2);
            return { x, y };
          });
          const points = series.map((point) => `${point.x},${point.y}`).join(' ');
          const last = series[series.length - 1];
          const color = marketOptionColor(option.label, palette[optionIndex % palette.length]);
          return (
            <g key={option.id}>
              <polyline fill="none" stroke={color} strokeWidth="3" strokeDasharray={optionIndex === 0 ? '0' : optionIndex === 1 ? '8 6' : '3 7'} points={points} />
              <circle cx={last.x} cy={last.y} r="5" fill={color} />
              <text x={last.x + 10} y={last.y + (optionIndex * 14) - 8} fill={color} fontSize="12" fontWeight="600">{option.label}</text>
            </g>
          );
        })}
      </svg>
      <div className="row row-wrap">
        {options.map((option, optionIndex) => (
          <span className="position-chip" key={option.id} style={{ borderColor: marketOptionColor(option.label, palette[optionIndex % palette.length]) }}>
            {option.label} · {formatBpsPercent(option.probability_bps)}
          </span>
        ))}
      </div>
    </>
  );

  if (!standout) {
    return (
      <Card className="chart-card">
        <SectionHeader title={title} copy={copy} />
        {chart}
      </Card>
    );
  }

  return (
    <Card className="chart-card">
      <SectionHeader title={title} copy={copy} />
      {chart}
    </Card>
  );
}
