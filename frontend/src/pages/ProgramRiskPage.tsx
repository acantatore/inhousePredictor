import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { ProgramRiskView } from '../types';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader, StatusPill } from '../components/ui';
import { formatBpsPercent, formatDate } from '../lib/utils';

export function ProgramRiskPage() {
  const { program = '' } = useParams();
  const { token } = useAuth();
  const [view, setView] = useState<ProgramRiskView | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token || !program) return;
    setIsLoading(true);
    api.getProgramRisk(token, program)
      .then(setView)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load the program risk view.'))
      .finally(() => setIsLoading(false));
  }, [program, token]);

  if (isLoading) return <LoadingState title="Loading program risk" copy="Building the current confidence view for this program." />;
  if (error) return <ErrorState title="Could not load program risk" copy={error} />;
  if (!view || view.questions.length === 0) return <EmptyState title="No commitments yet" copy="Create a forecast question in this program to start the executive view." />;

  return (
    <div className="stack-lg">
      <Card>
        <SectionHeader title={`${view.program} program risk`} copy="One program-level slice showing official internal forecast, current risk, rationale excerpts, and external overlays." />
      </Card>
      <div className="stack-md">
        {view.questions.map((row) => (
          <Card key={row.question_id} className="stack-sm">
            <div className="row row-wrap">
              <strong>{row.title}</strong>
              <StatusPill status={row.status as any} />
            </div>
            <div className="info-grid">
              <div className="stat-card"><span>Official forecast</span><strong>{formatBpsPercent(row.official_probability_bps)}</strong></div>
              <div className="stat-card"><span>Current risk</span><strong>{formatBpsPercent(row.current_risk_bps)}</strong></div>
              <div className="stat-card"><span>7-day change</span><strong>{row.change_last_7_days_bps >= 0 ? '+' : ''}{formatBpsPercent(Math.abs(row.change_last_7_days_bps))}</strong></div>
              <div className="stat-card"><span>Resolve by</span><strong>{formatDate(row.resolves_at)}</strong></div>
            </div>
            <div className="stack-sm">
              {row.latest_rationale_excerpts.map((excerpt, index) => <small key={index}>{excerpt}</small>)}
              {typeof row.external_signal_probability === 'number' ? <small>External overlay: {formatBpsPercent(row.external_signal_probability)}</small> : null}
              <small>Resolution owner: {row.resolution_owner}</small>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
