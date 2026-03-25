import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { ExternalSignalSnapshot, ForecastQuestion, ForecastQuestionOutcome, Market, Position } from '../types';
import { Card, EmptyState, ErrorState, Field, InlineNotice, LoadingState, SectionHeader, StatusPill, TextArea } from '../components/ui';
import { formatBpsPercent, formatDate } from '../lib/utils';
import { TradeTicket } from '../components/TradeTicket';

export function ForecastQuestionPage() {
  const { id = '' } = useParams();
  const { token, user, refreshUser } = useAuth();
  const [question, setQuestion] = useState<ForecastQuestion | null>(null);
  const [linkedMarket, setLinkedMarket] = useState<Market | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [probability, setProbability] = useState(6200);
  const [rationale, setRationale] = useState('');
  const [signal, setSignal] = useState<Pick<ExternalSignalSnapshot, 'source' | 'probability_bps' | 'note'>>({ source: 'Polymarket', probability_bps: 5400, note: '' });

  const isContributor = useMemo(() => question?.contributors?.some((candidate) => candidate.user_id === user?.id) ?? false, [question, user?.id]);
  const isResolver = question?.resolver_id === user?.id;

  useEffect(() => {
    if (!token) return;
    setIsLoading(true);
    api.getForecastQuestion(token, id)
      .then((item) => {
        setQuestion(item);
        setProbability(item.projection?.official_probability_bps || 5000);
        if (item.linked_market_id) {
          void api.getMarket(token, item.linked_market_id).then(setLinkedMarket).catch(() => setLinkedMarket(null));
          void api.getPositions(token).then(setPositions).catch(() => setPositions([]));
        } else {
          setLinkedMarket(null);
          setPositions([]);
        }
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load this commitment forecast.'))
      .finally(() => setIsLoading(false));
  }, [id, token]);

  async function submitForecast(event: React.FormEvent) {
    event.preventDefault();
    if (!token) return;
    try {
      const next = await api.submitForecast(token, id, { probability_bps: probability, rationale });
      setQuestion(next);
      setRationale('');
      setError(null);
      if (next.linked_market_id) {
        const nextMarket = await api.getMarket(token, next.linked_market_id);
        setLinkedMarket(nextMarket);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save this forecast.');
    }
  }

  async function resolve(outcome: ForecastQuestionOutcome) {
    if (!token) return;
    try {
      const next = await api.resolveForecastQuestion(token, id, { outcome, evidence_url: 'https://example.com/evidence/roadmap-forecast' });
      setQuestion(next);
      setError(null);
      if (next.linked_market_id) {
        const nextMarket = await api.getMarket(token, next.linked_market_id);
        setLinkedMarket(nextMarket);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not resolve this commitment.');
    }
  }

  async function addExternalSignal(event: React.FormEvent) {
    event.preventDefault();
    if (!token) return;
    try {
      await api.createExternalSignal(token, id, signal);
      const next = await api.getForecastQuestion(token, id);
      setQuestion(next);
      setError(null);
      if (next.linked_market_id) {
        const nextMarket = await api.getMarket(token, next.linked_market_id);
        setLinkedMarket(nextMarket);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the external signal.');
    }
  }

  async function reloadLinkedMarket() {
    if (!token || !question?.linked_market_id) return;
    const [nextMarket, nextPositions] = await Promise.all([
      api.getMarket(token, question.linked_market_id),
      api.getPositions(token),
    ]);
    setLinkedMarket(nextMarket);
    setPositions(nextPositions || []);
    await refreshUser();
  }

  if (isLoading) {
    return <LoadingState title="Loading commitment forecast" copy="Pulling the latest official internal forecast, rationale trail, and score state." />;
  }
  if (error && !question) {
    return <ErrorState title="Could not load commitment forecast" copy={error} />;
  }
  if (!question) {
    return <EmptyState title="Forecast question not found" copy="This commitment may have been removed or never existed." />;
  }

  return (
    <div className="stack-lg">
      <Card>
        <SectionHeader title={question.title} copy={question.description} action={<StatusPill status={question.status as any} />} />
        <div className="info-grid">
          <div className="stat-card">
            <span>Market signal</span>
            <strong>{linkedMarket ? formatBpsPercent(Math.round(linkedMarket.yes_price * 10000)) : 'Pending'}</strong>
            <small>Tradeable commitment signal derived from the paired market.</small>
          </div>
          <div className="stat-card">
            <span>Official internal forecast</span>
            <strong>{formatBpsPercent(question.projection?.official_probability_bps || 0)}</strong>
            <small>Median of the latest active contributor forecasts.</small>
          </div>
          <div className="stat-card">
            <span>Current risk</span>
            <strong>{formatBpsPercent(question.projection?.current_risk_bps || 0)}</strong>
            <small>Displayed as `1 - official forecast`.</small>
          </div>
          <div className="stat-card">
            <span>Shadow market</span>
            <strong>{question.linked_market_id ? 'Trade enabled' : 'Pending'}</strong>
            <small>The paired market is available for betting directly on this commitment.</small>
          </div>
        </div>
        <div className="row row-wrap">
          <InlineNotice tone="neutral">Program {question.program}</InlineNotice>
          <InlineNotice tone="neutral">Resolver {question.resolver_name}</InlineNotice>
          <InlineNotice tone="neutral">Close {formatDate(question.closes_at)}</InlineNotice>
          <InlineNotice tone="neutral">Resolve {formatDate(question.resolves_at)}</InlineNotice>
        </div>
      </Card>

      {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}

      <div className="two-column-layout">
        <div className="stack-md">
          <InlineNotice tone="neutral">For commitments, the paired market is now the primary participation surface. Trade here to express conviction with points.</InlineNotice>
          {linkedMarket && user && token ? <TradeTicket market={linkedMarket} token={token} user={user} positions={positions} onTraded={reloadLinkedMarket} /> : <InlineNotice tone="warning">The paired market is still being prepared.</InlineNotice>}
        </div>

        <Card className="stack-md">
          <SectionHeader title="Forecast update" copy="Submit a direct probability and explain why it changed. Revisions are append-only." />
          {isContributor && question.status === 'open' ? (
            <form className="stack-md" onSubmit={submitForecast}>
              <Field label="Probability">
                <input className="input" type="range" min={0} max={10000} step={100} value={probability} onChange={(event) => setProbability(Number(event.target.value))} />
              </Field>
              <InlineNotice tone="neutral">Current submission: {formatBpsPercent(probability)}</InlineNotice>
              <Field label="Rationale">
                <TextArea value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="What changed? What evidence matters?" />
              </Field>
              <button className="primary-button" type="submit">Save forecast revision</button>
            </form>
          ) : (
            <InlineNotice tone="warning">Only selected contributors can update this forecast while it remains open.</InlineNotice>
          )}
        </Card>

        <Card className="stack-md">
          <SectionHeader title="Signal comparison" copy="The paired market and any external source stay secondary. The official internal forecast remains the canonical answer." />
          <div className="stack-sm">
            <div className="activity-row"><span>Official internal forecast</span><strong>{formatBpsPercent(question.projection?.official_probability_bps || 0)}</strong></div>
            {linkedMarket ? <div className="activity-row"><span>Paired market YES price</span><strong>{formatBpsPercent(Math.round(linkedMarket.yes_price * 10000))}</strong></div> : null}
            {question.external_signals?.map((item) => (
              <div className="activity-row" key={item.id}><span>{item.source}</span><strong>{formatBpsPercent(item.probability_bps)}</strong></div>
            ))}
          </div>
          {user?.is_admin ? (
            <form className="stack-md" onSubmit={addExternalSignal}>
              <Field label="External source"><input className="input" value={signal.source} onChange={(event) => setSignal((current) => ({ ...current, source: event.target.value }))} /></Field>
              <Field label="Probability (bps)"><input className="input" type="number" min={0} max={10000} step={100} value={signal.probability_bps} onChange={(event) => setSignal((current) => ({ ...current, probability_bps: Number(event.target.value) }))} /></Field>
              <Field label="Note"><TextArea value={signal.note} onChange={(event) => setSignal((current) => ({ ...current, note: event.target.value }))} /></Field>
              <button className="secondary-button" type="submit">Add external signal snapshot</button>
            </form>
          ) : null}
        </Card>
      </div>

      <Card>
        <SectionHeader title="Rationale trail" copy="Show why confidence moved, not just where it landed." />
        <div className="stack-sm">
          {(question.latest_rationales || []).map((revision) => (
            <div className="question-card" key={revision.id}>
              <div className="row row-wrap"><strong>{revision.user_name}</strong><span>{formatBpsPercent(revision.probability_bps)} · {formatDate(revision.created_at)}</span></div>
              <p>{revision.rationale}</p>
            </div>
          ))}
        </div>
      </Card>

      <Card>
        <SectionHeader title="Contributor scorecard" copy="Brier score is the official individual score. Coverage stays secondary." />
        <div className="stack-sm">
          {(question.scores || []).length === 0 ? <InlineNotice tone="neutral">Scores appear after resolution.</InlineNotice> : null}
          {(question.scores || []).map((score) => (
            <div className="activity-row activity-row-rich" key={score.user_id}>
              <div>
                <strong>{score.user_name}</strong>
                <small>Probability at close: {formatBpsPercent(score.probability_bps)}</small>
              </div>
              <div className="stack-sm align-end">
                <strong>Brier {score.brier_score.toFixed(3)}</strong>
                <small>Coverage {(score.coverage_score * 100).toFixed(0)}%</small>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {isResolver && question.status !== 'resolved' ? (
        <Card>
          <SectionHeader title="Resolve commitment" copy="The resolver records the final outcome with evidence. Scoring and notifications use this final record." />
          <div className="row row-wrap">
            <button className="primary-button" type="button" onClick={() => resolve('delivered')}>Mark delivered</button>
            <button className="secondary-button" type="button" onClick={() => resolve('not_delivered')}>Mark not delivered</button>
            <button className="ghost-button" type="button" onClick={() => resolve('cancelled')}>Cancel</button>
          </div>
        </Card>
      ) : null}
    </div>
  );
}
