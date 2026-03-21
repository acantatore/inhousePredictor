import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { ForecastQuestion } from '../types';
import { Card, EmptyState, ErrorState, LoadingState, SectionHeader } from '../components/ui';

export function ProgramsPage() {
  const { token } = useAuth();
  const [questions, setQuestions] = useState<ForecastQuestion[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    api.listForecastQuestions(token)
      .then((items) => setQuestions(items || []))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Could not load programs.'))
      .finally(() => setIsLoading(false));
  }, [token]);

  const groups = useMemo(() => {
    const byProgram = new Map<string, ForecastQuestion[]>();
    questions.forEach((question) => {
      const existing = byProgram.get(question.program) || [];
      existing.push(question);
      byProgram.set(question.program, existing);
    });
    return Array.from(byProgram.entries()).sort((a, b) => a[0].localeCompare(b[0]));
  }, [questions]);

  if (isLoading) return <LoadingState title="Loading programs" copy="Gathering the current roadmap forecast slices by program." />;
  if (error) return <ErrorState title="Could not load programs" copy={error} />;
  if (groups.length === 0) return <EmptyState title="No programs yet" copy="Create a forecast question first and its program will show up here." />;

  return (
    <div className="stack-lg">
      <Card>
        <SectionHeader title="Program views" copy="Choose the program-level slice you want to review. Each one shows the official internal forecast, current risk, rationale excerpts, and external overlays." />
      </Card>
      <div className="stack-md">
        {groups.map(([program, items]) => (
          <Link className="activity-link" key={program} to={`/programs/${encodeURIComponent(program)}`}>
            <Card className="activity-row activity-row-rich">
              <div>
                <strong>{program}</strong>
                <small>{items.length} commitment forecasts in this slice</small>
              </div>
              <strong>{Math.round(items.reduce((sum, item) => sum + (item.projection?.current_risk_bps || 0), 0) / Math.max(items.length, 1) / 100)}% avg risk</strong>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
