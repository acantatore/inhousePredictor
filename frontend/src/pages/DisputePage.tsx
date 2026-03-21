import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { Market } from '../types';
import { Card, ErrorState, Field, InlineNotice, LoadingState, TextArea } from '../components/ui';

export function DisputePage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { token } = useAuth();
  const [market, setMarket] = useState<Market | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [reason, setReason] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      return;
    }
    api.getMarket(token, id)
      .then(setMarket)
      .catch((err) => setPageError(err instanceof ApiError ? err.message : 'Could not load this market.'))
      .finally(() => setIsLoading(false));
  }, [id, token]);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      await api.disputeMarket(token, id, { reason });
      navigate(`/markets/${id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not submit this dispute.');
    } finally {
      setIsSubmitting(false);
    }
  }

  if (isLoading) {
    return <LoadingState title="Loading dispute flow" copy="Checking whether this market is still inside the dispute window." />;
  }

  if (pageError || !market) {
    return <ErrorState title="Could not open dispute flow" copy={pageError || 'This market is not available.'} action={<Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>} />;
  }

  const disputeDeadlineOpen = Boolean(market.dispute_deadline) && new Date(market.dispute_deadline as string).getTime() > Date.now();
  if (market.status !== 'resolved' || !disputeDeadlineOpen) {
    return <ErrorState title="Dispute unavailable" copy="This market is not currently inside an active dispute window." action={<Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>} />;
  }

  return (
    <form className="stack-lg" onSubmit={submit}>
      <Card>
        <h2>Submit dispute</h2>
        <p>Use this when the recorded outcome or evidence does not support the result. An admin will review the dispute queue after you submit.</p>
        <InlineNotice tone="neutral">Dispute deadline: {market.dispute_deadline ? new Date(market.dispute_deadline).toLocaleString() : 'Not available'}.</InlineNotice>
      </Card>
      <Card className="form-card">
        <Field label="Reason" hint="Keep it focused. Explain what looks wrong and what evidence an admin should review next.">
          <TextArea rows={7} value={reason} onChange={(event) => setReason(event.target.value)} />
        </Field>
        <InlineNotice tone="neutral">After submission, the market moves into dispute review and no further trading is allowed.</InlineNotice>
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}
        <div className="button-row">
          <button className="primary-button" disabled={isSubmitting} type="submit">
            {isSubmitting ? 'Submitting dispute...' : 'Submit dispute'}
          </button>
          <Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>
        </div>
      </Card>
    </form>
  );
}
