import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { ForecastQuestion } from '../types';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader, StatusPill } from '../components/ui';
import { formatDate, formatBpsPercent } from '../lib/utils';

export function ForecastQuestionsPage() {
  const { token } = useAuth();
  const [questions, setQuestions] = useState<ForecastQuestion[]>([]);
  const [program, setProgram] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    setIsLoading(true);
    api.listForecastQuestions(token, program ? { program } : undefined)
      .then((items) => setQuestions(items || []))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load commitment forecasts.'))
      .finally(() => setIsLoading(false));
  }, [program, token]);

  const programs = useMemo(() => [...new Set(questions.map((question) => question.program))], [questions]);

  if (isLoading) {
    return <LoadingState title="Loading commitment forecasts" copy="Gathering the latest internal view of roadmap confidence." />;
  }
  if (error) {
    return <ErrorState title="Could not load commitment forecasts" copy={error} />;
  }

  return (
    <div className="stack-lg">
      <Card>
        <SectionHeader
          title="Commitment Forecasts"
          copy="A forecasting-first workspace for roadmap commitments, rationale, and visible confidence over time."
          action={<Link className="primary-button" to="/forecast-questions/new">Create commitment forecast</Link>}
        />
        <div className="filter-row">
          <label className="field">
            <span className="field-label">Program filter</span>
            <input className="input" value={program} onChange={(event) => setProgram(event.target.value)} placeholder="Type a program name" />
          </label>
        </div>
        {programs.length ? <p className="field-hint">Programs in view: {programs.join(', ')}</p> : null}
      </Card>

      {questions.length === 0 ? (
        <EmptyState
          title="No commitment forecasts yet"
          copy="Start with one roadmap commitment that matters to leadership. The goal is to capture confidence, rationale, and movement over time in one place."
          action={<Link className="primary-button" to="/forecast-questions/new">Create the first forecast question</Link>}
        />
      ) : (
        <div className="stack-md">
          {questions.map((question) => (
            <Link key={question.id} to={`/forecast-questions/${question.id}`} className="activity-link">
              <Card className="activity-row-rich">
                <div className="stack-sm">
                  <div className="row row-wrap">
                    <strong>{question.title}</strong>
                    <StatusPill status={question.status as any} />
                  </div>
                  <small>{question.program} · Resolver {question.resolver_name || 'Unassigned'}</small>
                  <small>{question.description}</small>
                </div>
                <div className="info-grid">
                  <div className="stack-sm">
                    <span className="eyebrow">Market signal</span>
                    <strong>{typeof question.linked_market_yes_price === 'number' ? formatBpsPercent(Math.round(question.linked_market_yes_price * 10000)) : formatBpsPercent(question.projection?.official_probability_bps || 0)}</strong>
                  </div>
                  <div className="stack-sm">
                    <span className="eyebrow">Current risk</span>
                    <strong>{formatBpsPercent(question.projection?.current_risk_bps || 0)}</strong>
                  </div>
                  <div className="stack-sm">
                    <span className="eyebrow">Contributors</span>
                    <strong>{question.projection?.contributor_count || 0}</strong>
                  </div>
                  <div className="stack-sm">
                    <span className="eyebrow">Close</span>
                    <strong>{formatDate(question.closes_at)}</strong>
                  </div>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
