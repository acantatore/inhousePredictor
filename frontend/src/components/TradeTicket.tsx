import { useMemo, useState } from 'react';
import { api, ApiError } from '../lib/api';
import { estimateShares, formatPercent, formatPoints, getTradeDisabledReason } from '../lib/utils';
import type { Market, TradeSide, User } from '../types';
import { Card, Field, InlineNotice, TextInput } from './ui';

export function TradeTicket({
  market,
  token,
  user,
  onTraded,
}: {
  market: Market;
  token: string;
  user: User;
  onTraded: () => Promise<void>;
}) {
  const [side, setSide] = useState<TradeSide>('yes');
  const [cost, setCost] = useState(200);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const currentProbability = side === 'yes' ? market.yes_price : market.no_price;
  const estimatedShares = useMemo(() => estimateShares(cost, currentProbability), [cost, currentProbability]);
  const disabledReason = getTradeDisabledReason(market, user.id, user.balance, cost);

  async function submitTrade(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setMessage(null);

    if (disabledReason) {
      setError(disabledReason);
      return;
    }

    try {
      setIsSubmitting(true);
      await api.trade(token, market.id, { side, cost });
      setMessage('Trade placed. Your balance and market signal are refreshing now.');
      await onTraded();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not place this trade. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Card className="trade-ticket">
      <div className="section-header">
        <div>
          <h2>Trade Ticket</h2>
          <p>Use points, not jargon. We show the current crowd signal before you act.</p>
        </div>
      </div>

      <div className="trade-toggle" role="tablist" aria-label="Trade side">
        <button className={side === 'yes' ? 'trade-toggle-active yes-tone' : 'yes-tone'} type="button" onClick={() => setSide('yes')}>
          Buy YES
        </button>
        <button className={side === 'no' ? 'trade-toggle-active no-tone' : 'no-tone'} type="button" onClick={() => setSide('no')}>
          Buy NO
        </button>
      </div>

      <form className="stack-md" onSubmit={submitTrade}>
        <Field label="Points to spend" hint={`Current balance ${formatPoints(user.balance)}`}>
          <TextInput
            min={1}
            step={50}
            type="number"
            value={cost}
            onChange={(event) => setCost(Number(event.target.value))}
          />
        </Field>

        <div className="ticket-summary">
          <div>
            <span>Current crowd signal</span>
            <strong>{side === 'yes' ? `YES ${formatPercent(market.yes_price)}` : `NO ${formatPercent(market.no_price)}`}</strong>
          </div>
          <div>
            <span>Estimated shares</span>
            <strong>{estimatedShares.toFixed(1)}</strong>
          </div>
        </div>

        <p className="helper-copy">Trading closes at {new Date(market.closes_at).toLocaleString()}. Creators cannot trade their own markets.</p>

        {disabledReason ? <InlineNotice tone="warning">{disabledReason}</InlineNotice> : null}
        {message ? <InlineNotice tone="success">{message}</InlineNotice> : null}
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}

        <button className="primary-button" disabled={isSubmitting || Boolean(disabledReason)} type="submit">
          {isSubmitting ? 'Placing trade...' : side === 'yes' ? 'Buy YES' : 'Buy NO'}
        </button>
      </form>
    </Card>
  );
}
