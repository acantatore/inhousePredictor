import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import { categories } from '../lib/utils';
import type { UserSummary } from '../types';
import { Card, Field, InlineNotice, SectionHeader, Select, TextArea, TextInput } from '../components/ui';

function toIsoLocal(value: string) {
  return new Date(value).toISOString();
}

export function CreateMarketPage() {
  const navigate = useNavigate();
  const { token, user, refreshUser } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [resolverSearch, setResolverSearch] = useState('');
  const [resolverOptions, setResolverOptions] = useState<UserSummary[]>([]);
  const [form, setForm] = useState({
    question: '',
    description: '',
    category: 'general',
    options: ['YES', 'NO'],
    resolver_id: '',
    initial_liquidity: 500,
    closes_at: '',
    resolves_at: '',
  });

  useEffect(() => {
    if (!token) {
      return;
    }
    let cancelled = false;
    api.listUsers(token, { q: resolverSearch, limit: 20 })
      .then((users) => {
        if (cancelled) {
          return;
        }
        setResolverOptions((users || []).filter((candidate) => candidate.id !== user?.id));
      })
      .catch(() => {
        if (!cancelled) {
          setResolverOptions([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [resolverSearch, token, user?.id]);

  const selectedResolver = useMemo(
    () => resolverOptions.find((candidate) => candidate.id === form.resolver_id),
    [form.resolver_id, resolverOptions],
  );

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }

    setError(null);
    setMessage(null);
    setIsSubmitting(true);
    try {
      const market = await api.createMarket(token, {
        ...form,
        options: form.options.filter((option) => option.trim() !== ''),
        initial_liquidity: Number(form.initial_liquidity),
        closes_at: toIsoLocal(form.closes_at),
        resolves_at: toIsoLocal(form.resolves_at),
      });
      setMessage('Market created. We are taking you to the detail page now.');
      await refreshUser();
      navigate(`/markets/${market.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create this market.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <form className="stack-lg" onSubmit={submit}>
      <Card>
        <SectionHeader title="Create Market" copy="A good market is concrete, resolvable, and clear about who will call the outcome." />
        {user ? <InlineNotice tone="neutral">Your current balance is {user.balance} points. Initial liquidity comes from that balance, and creators cannot trade their own markets.</InlineNotice> : null}
      </Card>

      <Card className="form-card">
        <SectionHeader title="Question" copy="Ask one thing that can be resolved clearly by the deadline." />
        <Field label="Market question" hint="Concrete beats clever. Write the exact thing the resolver will judge.">
          <TextInput value={form.question} onChange={(event) => setForm((current) => ({ ...current, question: event.target.value }))} />
        </Field>
        <Field label="Context" hint="Explain why the question matters and what teammates should know before buying YES or NO.">
          <TextArea rows={6} value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
        </Field>
        <Field label="Options" hint="Add 2 or more outcomes. Markets can now have more than two outcomes.">
          <div className="stack-sm">
            {form.options.map((option, index) => (
              <div className="row row-wrap" key={`${option}-${index}`}>
                <TextInput value={option} onChange={(event) => setForm((current) => ({ ...current, options: current.options.map((item, itemIndex) => itemIndex === index ? event.target.value : item) }))} />
                {form.options.length > 2 ? (
                  <button className="ghost-button" type="button" onClick={() => setForm((current) => ({ ...current, options: current.options.filter((_, itemIndex) => itemIndex !== index) }))}>
                    Remove
                  </button>
                ) : null}
              </div>
            ))}
            <button className="secondary-button" type="button" onClick={() => setForm((current) => ({ ...current, options: [...current.options, `Option ${current.options.length + 1}`] }))}>
              Add option
            </button>
          </div>
        </Field>
      </Card>

      <Card className="form-card">
        <SectionHeader title="Resolver" copy="The resolver is responsible for recording the outcome and linking evidence when the time comes." />
        <Field label="Category">
          <Select value={form.category} onChange={(event) => setForm((current) => ({ ...current, category: event.target.value }))}>
            {categories.map((item) => (
              <option key={item.value} value={item.value}>{item.label}</option>
            ))}
          </Select>
        </Field>
        <Field label="Find resolver" hint="Search by teammate name or email, then choose the person who should call the outcome.">
          <TextInput value={resolverSearch} onChange={(event) => setResolverSearch(event.target.value)} placeholder="Search teammate" />
        </Field>
        <Field label="Resolver">
          <Select value={form.resolver_id} onChange={(event) => setForm((current) => ({ ...current, resolver_id: event.target.value }))}>
            <option value="">Select a resolver</option>
            {resolverOptions.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.name} - {candidate.email}
              </option>
            ))}
          </Select>
        </Field>
        {selectedResolver ? <InlineNotice tone="neutral">Resolver selected: {selectedResolver.name}. They will need to record the outcome and link evidence when the time comes.</InlineNotice> : null}
      </Card>

      <Card className="form-card">
        <SectionHeader title="Timing and liquidity" copy="Close time is when trading stops. Resolve time is when the answer should be knowable." />
        <div className="form-grid">
          <Field label="Closes at">
            <TextInput type="datetime-local" value={form.closes_at} onChange={(event) => setForm((current) => ({ ...current, closes_at: event.target.value }))} />
          </Field>
          <Field label="Resolves at">
            <TextInput type="datetime-local" value={form.resolves_at} onChange={(event) => setForm((current) => ({ ...current, resolves_at: event.target.value }))} />
          </Field>
          <Field label="Initial liquidity" hint="Minimum 100 points.">
            <TextInput type="number" min={100} step={50} value={form.initial_liquidity} onChange={(event) => setForm((current) => ({ ...current, initial_liquidity: Number(event.target.value) }))} />
          </Field>
        </div>
      </Card>

      <Card>
        <SectionHeader title="Review" copy="Check the trust rules before you create the market." />
        <div className="stack-sm">
          <InlineNotice tone="neutral">Creators cannot trade their own markets.</InlineNotice>
          <InlineNotice tone="neutral">Resolvers should be different from creators and should be able to link evidence at resolution time.</InlineNotice>
          <InlineNotice tone="neutral">If the recorded outcome looks wrong, teammates can submit a dispute during the dispute window.</InlineNotice>
          {message ? <InlineNotice tone="success">{message}</InlineNotice> : null}
          {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}
        </div>
        <button className="primary-button" disabled={isSubmitting} type="submit">
          {isSubmitting ? 'Creating market...' : 'Create market'}
        </button>
      </Card>
    </form>
  );
}
