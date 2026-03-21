import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import type { UserSummary } from '../types';
import { Card, Field, InlineNotice, SectionHeader, TextArea, TextInput, Select } from '../components/ui';

export function CreateForecastQuestionPage() {
  const { token, user } = useAuth();
  const navigate = useNavigate();
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    program: 'payments',
    resolver_id: '',
    resolution_rule: 'Delivered by the stated date with the agreed milestone scope complete.',
    closes_at: '',
    resolves_at: '',
    contributor_ids: [] as string[],
  });

  useEffect(() => {
    if (!token) return;
    api.listUsers(token, { limit: 50 }).then((items) => setUsers(items || [])).catch(() => setUsers([]));
  }, [token]);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    setIsSubmitting(true);
    setError(null);
    setMessage(null);
    try {
      const question = await api.createForecastQuestion(token, {
        ...form,
        contributor_ids: Array.from(new Set([...form.contributor_ids, user?.id || ''])).filter(Boolean),
        closes_at: new Date(form.closes_at).toISOString(),
        resolves_at: new Date(form.resolves_at).toISOString(),
      });
      setMessage('Commitment forecast created.');
      navigate(`/forecast-questions/${question.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create this forecast question.');
    } finally {
      setIsSubmitting(false);
    }
  }

  function toggleContributor(id: string) {
    setForm((current) => ({
      ...current,
      contributor_ids: current.contributor_ids.includes(id)
        ? current.contributor_ids.filter((value) => value !== id)
        : [...current.contributor_ids, id],
    }));
  }

  return (
    <form className="stack-lg" onSubmit={onSubmit}>
      <Card>
        <SectionHeader title="Create commitment forecast" copy="Turn one roadmap commitment into a forecasting workflow with explicit ownership, rationale, and a visible confidence trail." />
        {message ? <InlineNotice tone="success">{message}</InlineNotice> : null}
        {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}
      </Card>
      <Card className="form-card">
        <Field label="Commitment title">
          <TextInput value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} placeholder="Will Team Payments ship milestone M3 by June 30?" />
        </Field>
        <Field label="Program">
          <TextInput value={form.program} onChange={(event) => setForm((current) => ({ ...current, program: event.target.value }))} />
        </Field>
        <Field label="Context">
          <TextArea value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
        </Field>
        <Field label="Resolution rule">
          <TextArea value={form.resolution_rule} onChange={(event) => setForm((current) => ({ ...current, resolution_rule: event.target.value }))} />
        </Field>
        <Field label="Resolver / review owner">
          <Select value={form.resolver_id} onChange={(event) => setForm((current) => ({ ...current, resolver_id: event.target.value }))}>
            <option value="">Choose a resolver</option>
            {users.filter((candidate) => candidate.id !== user?.id).map((candidate) => (
              <option key={candidate.id} value={candidate.id}>{candidate.name} - {candidate.email}</option>
            ))}
          </Select>
        </Field>
        <div className="info-grid">
          <Field label="Close forecasting">
            <TextInput type="datetime-local" value={form.closes_at} onChange={(event) => setForm((current) => ({ ...current, closes_at: event.target.value }))} />
          </Field>
          <Field label="Resolve by">
            <TextInput type="datetime-local" value={form.resolves_at} onChange={(event) => setForm((current) => ({ ...current, resolves_at: event.target.value }))} />
          </Field>
        </div>
      </Card>
      <Card>
        <SectionHeader title="Contributors" copy="Choose the people whose direct forecasts should shape the official internal forecast." />
        <div className="stack-sm">
          {users.map((candidate) => (
            <label key={candidate.id} className="checkbox-row">
              <input type="checkbox" checked={form.contributor_ids.includes(candidate.id)} onChange={() => toggleContributor(candidate.id)} />
              <span>{candidate.name} · {candidate.email}</span>
            </label>
          ))}
        </div>
      </Card>
      <div className="row row-wrap">
        <button className="primary-button" type="submit" disabled={isSubmitting}>{isSubmitting ? 'Creating...' : 'Create commitment forecast'}</button>
        <InlineNotice tone="neutral">Phase 1 will auto-create a paired shadow market for advanced signal comparison, but direct forecasts remain the official truth.</InlineNotice>
      </div>
    </form>
  );
}
