import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { DisputeAction, DisputeRecord, Outcome } from '../types';
import { formatDate, formatPoints } from '../lib/utils';
import { Card, EmptyState, ErrorState, Field, InlineNotice, LoadingState, Select, TextInput } from '../components/ui';

export function AdminDisputesPage() {
  const { token, user } = useAuth();
  const [disputes, setDisputes] = useState<DisputeRecord[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      return;
    }
    api.listDisputes(token)
      .then((nextDisputes) => setDisputes(nextDisputes || []))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load the dispute queue.'))
      .finally(() => setIsLoading(false));
  }, [token]);

  if (!user?.is_admin) {
    return <ErrorState title="Admin access only" copy="This queue is reserved for admins handling disputes and operational edge cases." action={<Link className="secondary-button" to="/markets">Back to markets</Link>} />;
  }

  if (isLoading) {
    return <LoadingState title="Loading dispute queue" copy="Gathering unresolved disputes so admins can move them forward without leaving the product." />;
  }

  if (error) {
    return <ErrorState title="Could not load disputes" copy={error} />;
  }

  return (
    <div className="stack-lg">
      <Card>
        <h2>Admin disputes</h2>
        <p>A focused queue for trust-sensitive review work. Each row keeps the question, evidence, reason, and final action close together.</p>
      </Card>
      {disputes.length === 0 ? (
        <EmptyState title="No disputes to review" copy="The queue is clear right now. New disputes will appear here with their reason, evidence state, and action options." />
      ) : (
        <div className="stack-md">
          {disputes.map((dispute) => (
            <DisputeReviewCard key={dispute.dispute_id} dispute={dispute} onDone={() => setDisputes((current) => current.filter((item) => item.dispute_id !== dispute.dispute_id))} />
          ))}
        </div>
      )}
    </div>
  );
}

function DisputeReviewCard({ dispute, onDone }: { dispute: DisputeRecord; onDone: () => void }) {
  const { token } = useAuth();
  const [action, setAction] = useState<DisputeAction>('confirm_original');
  const [outcome, setOutcome] = useState<Outcome>('yes');
  const [evidenceUrl, setEvidenceUrl] = useState(dispute.evidence_url || '');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      await api.reviewDispute(token, dispute.market_id, {
        action,
        outcome: action === 'override_outcome' ? outcome : undefined,
        evidence_url: action === 'override_outcome' ? evidenceUrl : undefined,
      });
      onDone();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not complete this dispute action.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Card className="dispute-card">
      <div className="stack-sm">
        <div className="section-header">
          <div>
            <h3>{dispute.market_question}</h3>
            <p>Submitted {formatDate(dispute.created_at)}</p>
          </div>
          <Link className="secondary-button" to={`/markets/${dispute.market_id}`}>Open market</Link>
        </div>

        <div className="info-grid">
          <div className="stack-sm">
            <p><strong>Reason:</strong> {dispute.reason}</p>
            <p><strong>Creator:</strong> {dispute.creator_name}</p>
            <p><strong>Resolver:</strong> {dispute.resolver_name}</p>
          </div>
          <div className="stack-sm">
            <p><strong>Status:</strong> {dispute.status}</p>
            <p><strong>Evidence:</strong> {dispute.evidence_url ? 'Linked' : 'Missing'}</p>
            <p><strong>Initial liquidity:</strong> {formatPoints(dispute.initial_liquidity)}</p>
          </div>
        </div>
      </div>

      <form className="stack-sm" onSubmit={submit}>
        <Field label="Action">
          <Select value={action} onChange={(event) => setAction(event.target.value as DisputeAction)}>
            <option value="confirm_original">Confirm original outcome</option>
            <option value="override_outcome">Override outcome</option>
            <option value="cancel_market">Cancel market</option>
          </Select>
        </Field>

        {action === 'override_outcome' ? (
          <div className="form-grid">
            <Field label="Updated outcome">
              <Select value={outcome} onChange={(event) => setOutcome(event.target.value as Outcome)}>
                <option value="yes">YES</option>
                <option value="no">NO</option>
              </Select>
            </Field>
            <Field label="Updated evidence URL">
              <TextInput type="url" value={evidenceUrl} onChange={(event) => setEvidenceUrl(event.target.value)} />
            </Field>
          </div>
        ) : null}

        <InlineNotice tone="neutral">Use cancel only when the market should not resolve to YES or NO at all. Confirm or override when you still have a trustworthy final outcome.</InlineNotice>
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}
        <button className="primary-button" disabled={isSubmitting} type="submit">
          {isSubmitting ? 'Saving action...' : 'Complete dispute review'}
        </button>
      </form>
    </Card>
  );
}
