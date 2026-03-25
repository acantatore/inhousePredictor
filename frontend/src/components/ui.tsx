import { NavLink } from 'react-router-dom';
import type { ReactNode } from 'react';
import type { MarketStatus } from '../types';
import { formatDate, formatPercent, formatPoints, formatRelativeTime } from '../lib/utils';

export function AppShell({
  title,
  subtitle,
  balance,
  liveLabel,
  isAdmin,
  userName,
  onLogout,
  children,
}: {
  title?: string;
  subtitle?: string;
  balance?: number;
  liveLabel?: string;
  isAdmin?: boolean;
  userName?: string;
  onLogout: () => void;
  children: ReactNode;
}) {
  return (
    <div className="app-frame">
      <header className="topbar">
        <div>
          <p className="eyebrow">Shared company forecasting</p>
          <h1 className="topbar-title">{title || 'LooM'}</h1>
          {subtitle ? <p className="topbar-subtitle">{subtitle}</p> : null}
        </div>
        <div className="topbar-actions">
          {liveLabel ? <span className="live-pill live-live">{liveLabel}</span> : null}
          {typeof balance === 'number' ? <span className="balance-badge">Balance {formatPoints(balance)}</span> : null}
          <div className="profile-chip">
            <span>{userName || 'Teammate'}</span>
            {isAdmin ? <span className="status-pill status-admin">Admin</span> : null}
            <button className="ghost-button" type="button" onClick={onLogout}>
              Log out
            </button>
          </div>
        </div>
      </header>

      <div className="primary-nav-shell">
        <nav className="primary-nav" aria-label="Primary navigation">
          <NavItem to="/markets">Markets</NavItem>
          <NavItem to="/portfolio">Portfolio</NavItem>
          <NavItem to="/profile">Profile</NavItem>
          {isAdmin ? <NavItem to="/admin/disputes">Admin</NavItem> : null}
          <NavLink to="/create" className="primary-button nav-create-button nav-link-create">
            Create Market
          </NavLink>
        </nav>
      </div>

      <main className="page-shell">{children}</main>

      <nav className="mobile-nav" aria-label="Mobile navigation">
        <NavItem to="/markets">Markets</NavItem>
        <NavItem to="/portfolio">Portfolio</NavItem>
        <NavItem to={isAdmin ? '/admin/disputes' : '/profile'}>{isAdmin ? 'Admin' : 'Profile'}</NavItem>
      </nav>
    </div>
  );
}

function NavItem({ to, children }: { to: string; children: ReactNode }) {
  return (
    <NavLink to={to} className={({ isActive }) => `nav-link${isActive ? ' nav-link-active' : ''}`}>
      {children}
    </NavLink>
  );
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <section className={`card ${className}`.trim()}>{children}</section>;
}

export function SectionHeader({ title, copy, action }: { title: string; copy?: string; action?: ReactNode }) {
  return (
    <div className="section-header">
      <div>
        <h2>{title}</h2>
        {copy ? <p>{copy}</p> : null}
      </div>
      {action}
    </div>
  );
}

export function StatusPill({ status }: { status: MarketStatus | 'admin' }) {
  const label = status === 'admin' ? 'Admin' : status[0].toUpperCase() + status.slice(1);
  return <span className={`status-pill status-${status}`}>{label}</span>;
}

export function ProbabilitySplit({ yesPrice, noPrice }: { yesPrice: number; noPrice: number }) {
  return (
    <div className="probability-block" aria-label={`YES ${formatPercent(yesPrice)} and NO ${formatPercent(noPrice)}`}>
      <div className="probability-bar">
        <div className="probability-yes" style={{ width: `${yesPrice * 100}%` }} />
        <div className="probability-no" style={{ width: `${noPrice * 100}%` }} />
      </div>
      <div className="probability-meta">
        <span>YES {formatPercent(yesPrice)}</span>
        <span>NO {formatPercent(noPrice)}</span>
      </div>
    </div>
  );
}

export function DeadlineBlock({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="deadline-block">
      <span>{label}</span>
      <strong>{formatDate(value)}</strong>
      <small>{formatRelativeTime(value)}</small>
    </div>
  );
}

export function LoadingState({ title, copy }: { title: string; copy: string }) {
  return (
    <Card className="state-card">
      <h3>{title}</h3>
      <p>{copy}</p>
    </Card>
  );
}

export function ErrorState({ title, copy, action }: { title: string; copy: string; action?: ReactNode }) {
  return (
    <Card className="state-card state-error">
      <h3>{title}</h3>
      <p>{copy}</p>
      {action}
    </Card>
  );
}

export function EmptyState({ title, copy, action }: { title: string; copy: string; action?: ReactNode }) {
  return (
    <Card className="state-card state-empty">
      <h3>{title}</h3>
      <p>{copy}</p>
      {action}
    </Card>
  );
}

export function StatCard({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <Card className="stat-card">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </Card>
  );
}

export function InlineNotice({ tone = 'neutral', children }: { tone?: 'neutral' | 'success' | 'warning' | 'error'; children: ReactNode }) {
  return <div className={`inline-notice inline-${tone}`}>{children}</div>;
}

export function TextInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`input ${props.className || ''}`.trim()} />;
}

export function TextArea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea {...props} className={`input textarea ${props.className || ''}`.trim()} />;
}

export function Select(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return <select {...props} className={`input ${props.className || ''}`.trim()} />;
}

export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
      {hint ? <small className="field-hint">{hint}</small> : null}
    </label>
  );
}
