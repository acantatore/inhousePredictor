import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { ApiError } from '../lib/api';
import { useAuth } from '../context/AuthContext';
import { Card, Field, InlineNotice, TextInput } from '../components/ui';

export function AuthPage() {
  const { user, login, register } = useAuth();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', email: '', password: '' });

  if (user) {
    return <Navigate to="/forecast-questions" replace />;
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      if (mode === 'login') {
        await login({ email: form.email, password: form.password });
      } else {
        await register(form);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not continue. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="auth-layout">
      <section className="auth-copy">
        <p className="eyebrow">Warm, trustworthy forecasting</p>
          <h1>See what your team thinks before the next decision lands.</h1>
          <p>
            InhousePredictor helps teams turn roadmap commitments into visible forecasts with rationale, confidence history, and evidence-backed outcomes.
          </p>
        <div className="auth-highlights">
          <Card>
              <h2>What you can do</h2>
              <p>Create commitment forecasts, track confidence over time, compare shadow market signals, and keep leadership aligned on risk.</p>
            </Card>
          <Card>
            <h2>Why it feels safe</h2>
            <p>Resolver identity, deadlines, evidence, and dispute windows stay visible on the screen where decisions happen.</p>
          </Card>
        </div>
      </section>

      <Card className="auth-panel">
        <div className="trade-toggle auth-toggle">
          <button className={mode === 'login' ? 'trade-toggle-active' : ''} type="button" onClick={() => setMode('login')}>
            Login
          </button>
          <button className={mode === 'register' ? 'trade-toggle-active' : ''} type="button" onClick={() => setMode('register')}>
            Register
          </button>
        </div>

        <form className="stack-md" onSubmit={handleSubmit}>
          {mode === 'register' ? (
            <Field label="Name">
              <TextInput value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
            </Field>
          ) : null}

          <Field label="Work email">
            <TextInput type="email" value={form.email} onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))} />
          </Field>

          <Field label="Password" hint={mode === 'register' ? 'Use a strong password you can remember.' : undefined}>
            <TextInput type="password" value={form.password} onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))} />
          </Field>

          {error ? <InlineNotice tone="error">{error}</InlineNotice> : null}

          <button className="primary-button" disabled={isSubmitting} type="submit">
            {isSubmitting ? 'Working...' : mode === 'login' ? 'Login' : 'Create account'}
          </button>
        </form>
      </Card>
    </div>
  );
}
