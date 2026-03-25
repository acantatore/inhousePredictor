import { useMemo, useState } from 'react';
import { api, ApiError } from '../lib/api';
import { estimateShares, formatBpsPercent, formatPoints, getMarketOptions, getTradeDisabledReason, getPositionForMarket } from '../lib/utils';
import type { Market, MarketOption, Position, TradeSide, User } from '../types';
import { Card, Field, InlineNotice, TextInput } from './ui';

export function TradeTicket({
  market,
  token,
  user,
  positions,
  onTraded,
}: {
  market: Market;
  token: string;
  user: User;
  positions: Position[];
  onTraded: () => Promise<void>;
}) {
  const options = useMemo(() => getMarketOptions(market), [market]);
  const [selectedOptionId, setSelectedOptionId] = useState<string>(() => options[0]?.id || '');
  const [cost, setCost] = useState(200);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<'buy' | 'sell'>('buy');

  const selectedOption = options.find((option) => option.id === selectedOptionId) || options[0];
  const currentProbability = selectedOption ? selectedOption.probability_bps / 10000 : 0;
  const estimatedShares = useMemo(() => estimateShares(cost, currentProbability), [cost, currentProbability]);
  // For sell mode: estimate proceeds based on current price (approximate, actual may vary due to slippage)
  const estimatedProceeds = useMemo(() => {
    if (mode === 'sell' && cost > 0) {
      return Math.floor(cost * currentProbability);
    }
    return 0;
  }, [mode, cost, currentProbability]);
  const disabledReason = getTradeDisabledReason(market, user.id, user.balance, cost);

  // Get user position for this market
  const position = useMemo(() => getPositionForMarket(positions, market.id), [positions, market.id]);
  const yesShares = position?.yes_shares || 0;
  const noShares = position?.no_shares || 0;
  const hasYesPosition = yesShares > 0;
  const hasNoPosition = noShares > 0;
  const hasPosition = hasYesPosition || hasNoPosition;

  async function submitTrade(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setMessage(null);

    if (mode === 'buy' && disabledReason) {
      setError(disabledReason);
      return;
    }

    // Validate sell orders
    if (mode === 'sell') {
      if (selectedOption?.label === 'YES' && yesShares < cost) {
        setError(`You only have ${yesShares.toFixed(2)} YES shares to sell.`);
        return;
      }
      if (selectedOption?.label === 'NO' && noShares < cost) {
        setError(`You only have ${noShares.toFixed(2)} NO shares to sell.`);
        return;
      }
    }

    try {
      setIsSubmitting(true);
      const side: TradeSide = mode === 'buy'
        ? selectedOption?.label.toLowerCase() as TradeSide
        : (`sell_${selectedOption?.label.toLowerCase()}` as TradeSide);
      await api.trade(token, market.id, { side, option_id: selectedOption?.id, cost });
      setMessage(mode === 'buy' ? 'Trade placed. Your balance and market signal are refreshing now.' : 'Shares sold. Your balance has been updated.');
      setCost(200);
      await onTraded();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not place this trade. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  // Determine which options can be sold
  const canSellYes = hasYesPosition && selectedOption?.label === 'YES';
  const canSellNo = hasNoPosition && selectedOption?.label === 'NO';
  const canSellSelected = (mode === 'sell' && ((selectedOption?.label === 'YES' && hasYesPosition) || (selectedOption?.label === 'NO' && hasNoPosition)));

  return (
    <Card className="trade-ticket">
      <div className="section-header">
        <div>
          <h2>Trade Ticket</h2>
          <p>Use points, not jargon. We show the current crowd signal before you act.</p>
        </div>
      </div>

      {/* Buy/Sell Toggle */}
      {hasPosition && (
        <div className="trade-toggle auth-toggle">
          <button
            type="button"
            className={mode === 'buy' ? 'trade-toggle-active' : ''}
            onClick={() => setMode('buy')}
          >
            Buy
          </button>
          <button
            type="button"
            className={mode === 'sell' ? 'trade-toggle-active' : ''}
            onClick={() => setMode('sell')}
          >
            Sell
          </button>
        </div>
      )}

      <div className="stack-sm" role="tablist" aria-label="Trade option">
        {options.map((option) => {
          const isSellable = mode === 'sell' && ((option.label === 'YES' && hasYesPosition) || (option.label === 'NO' && hasNoPosition));
          const isDisabled = mode === 'buy' ? Boolean(disabledReason) : !isSellable;
          return (
            <button
              key={option.id}
              className={selectedOptionId === option.id ? 'trade-toggle-active trade-option-button' : 'trade-option-button'}
              type="button"
              onClick={() => setSelectedOptionId(option.id)}
              disabled={isDisabled}
            >
              {option.label} · {formatBpsPercent(option.probability_bps)}
              {mode === 'sell' && option.label === 'YES' && hasYesPosition && ` · ${yesShares.toFixed(1)} shares`}
              {mode === 'sell' && option.label === 'NO' && hasNoPosition && ` · ${noShares.toFixed(1)} shares`}
            </button>
          );
        })}
      </div>

      <form className="stack-md" onSubmit={submitTrade}>
        <Field
          label={mode === 'buy' ? 'Points to spend' : 'Shares to sell'}
          hint={mode === 'buy' ? `Current balance ${formatPoints(user.balance)}` : undefined}
        >
          <TextInput
            min={1}
            step={mode === 'buy' ? 50 : 0.01}
            type="number"
            value={cost}
            disabled={mode === 'buy' ? Boolean(disabledReason) : !canSellSelected}
            onChange={(event) => setCost(Number(event.target.value))}
          />
        </Field>

        <div className="ticket-summary">
          <div>
            <span>Current crowd signal</span>
            <strong>{selectedOption ? `${selectedOption.label} ${formatBpsPercent(selectedOption.probability_bps)}` : 'N/A'}</strong>
          </div>
          {mode === 'buy' && (
            <div>
              <span>Estimated shares</span>
              <strong>{estimatedShares.toFixed(1)}</strong>
            </div>
          )}
          {mode === 'sell' && hasPosition && (
            <>
              <div>
                <span>Your position</span>
                <strong>{selectedOption?.label === 'YES' ? `${yesShares.toFixed(1)} YES` : `${noShares.toFixed(1)} NO`}</strong>
              </div>
              <div>
                <span>Est. proceeds</span>
                <strong>{formatPoints(estimatedProceeds)}</strong>
              </div>
            </>
          )}
        </div>

        <p className="helper-copy">
          {mode === 'buy'
            ? `Trading closes at ${new Date(market.closes_at).toLocaleString()}. Creators cannot trade their own markets.`
            : `Sell shares back to the pool. Proceeds depend on current market price. You have ${yesShares.toFixed(1)} YES and ${noShares.toFixed(1)} NO shares.`}
        </p>

        {disabledReason && mode === 'buy' ? <InlineNotice tone="warning">{disabledReason}</InlineNotice> : null}
        {message ? <InlineNotice tone="success">{message}</InlineNotice> : null}
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}

        <button
          className={mode === 'sell' ? 'secondary-button' : 'primary-button'}
          disabled={isSubmitting || (mode === 'buy' && Boolean(disabledReason)) || (mode === 'sell' && !canSellSelected)}
          type="submit"
        >
          {isSubmitting
            ? mode === 'buy' ? 'Placing trade...' : 'Selling shares...'
            : mode === 'buy'
              ? selectedOption ? `Buy ${selectedOption.label}` : 'Buy option'
              : selectedOption ? `Sell ${selectedOption.label}` : 'Sell option'}
        </button>
      </form>
    </Card>
  );
}
