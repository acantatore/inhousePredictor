import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { Market } from '../types';
import { Card, ErrorState, Field, InlineNotice, LoadingState, Select, TextInput } from '../components/ui';

export function ResolveMarketPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { token, user } = useAuth();
  const [market, setMarket] = useState<Market | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [outcome, setOutcome] = useState<'yes' | 'no'>('yes');
  const [evidenceUrl, setEvidenceUrl] = useState('');
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
      await api.resolveMarket(token, id, { outcome, evidence_url: evidenceUrl });
      navigate(`/markets/${id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not record this resolution.');
    } finally {
      setIsSubmitting(false);
    }
  }

  if (isLoading) {
    return <LoadingState title="Loading resolution flow" copy="Checking resolver access, deadlines, and the evidence requirements for this market." />;
  }

  if (pageError || !market || !user) {
    return <ErrorState title="Could not open resolution flow" copy={pageError || 'This market is not available.'} action={<Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>} />;
  }

  if (market.resolver_id !== user.id || market.status !== 'open') {
    return <ErrorState title="Resolution unavailable" copy="Only the assigned resolver can record the outcome while this market is still open for resolution." action={<Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>} />;
  }

  return (
    <form className="stack-lg" onSubmit={submit}>
      <Card>
        <h2>Resolve market</h2>
        <p>Confirm the outcome plainly, attach evidence, and remember that teammates can dispute the result during the dispute window.</p>
        <InlineNotice tone="neutral">Resolver: {market.resolver_name || 'Assigned teammate'}.</InlineNotice>
      </Card>
      <Card className="form-card">
        <Field label="Outcome">
          <Select value={outcome} onChange={(event) => setOutcome(event.target.value as 'yes' | 'no')}>
            <option value="yes">YES</option>
            <option value="no">NO</option>
          </Select>
        </Field>
        <Field label="Evidence URL" hint="Link the source a teammate should review if they want to understand or challenge the outcome.">
          <TextInput type="url" value={evidenceUrl} onChange={(event) => setEvidenceUrl(event.target.value)} />
        </Field>
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}
        <div className="button-row">
          <button className="primary-button" disabled={isSubmitting} type="submit">
            {isSubmitting ? 'Recording resolution...' : 'Resolve market'}
          </button>
          <Link className="secondary-button" to={`/markets/${id}`}>Back to market</Link>
        </div>
      </Card>
    </form>
  );
}
