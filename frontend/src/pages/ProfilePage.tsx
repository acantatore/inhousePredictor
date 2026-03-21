import { Card, SectionHeader } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import { formatDate, formatPoints } from '../lib/utils';

export function ProfilePage() {
  const { user } = useAuth();

  if (!user) {
    return null;
  }

  return (
    <div className="stack-lg">
      <Card>
        <SectionHeader
          title="Profile"
          copy="A quick read on the teammate account behind your forecasting activity."
        />
        <div className="info-grid">
          <div className="stack-sm">
            <p><strong>Name:</strong> {user.name}</p>
            <p><strong>Email:</strong> {user.email}</p>
            <p><strong>Joined:</strong> {formatDate(user.created_at)}</p>
          </div>
          <div className="stack-sm">
            <p><strong>Balance:</strong> {formatPoints(user.balance)}</p>
            <p><strong>Role:</strong> {user.is_admin ? 'Admin' : 'Employee'}</p>
            <p><strong>Visibility:</strong> Your balance and positions stay private to you.</p>
          </div>
        </div>
      </Card>
    </div>
  );
}
