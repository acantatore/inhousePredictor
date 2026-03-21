import { Navigate, Outlet, Route, Routes } from 'react-router-dom';
import { AuthPage } from '../pages/AuthPage';
import { MarketsPage } from '../pages/MarketsPage';
import { MarketDetailPage } from '../pages/MarketDetailPage';
import { PortfolioPage } from '../pages/PortfolioPage';
import { CreateMarketPage } from '../pages/CreateMarketPage';
import { ResolveMarketPage } from '../pages/ResolveMarketPage';
import { DisputePage } from '../pages/DisputePage';
import { AdminDisputesPage } from '../pages/AdminDisputesPage';
import { ProfilePage } from '../pages/ProfilePage';
import { ForecastQuestionsPage } from '../pages/ForecastQuestionsPage';
import { ForecastQuestionPage } from '../pages/ForecastQuestionPage';
import { CreateForecastQuestionPage } from '../pages/CreateForecastQuestionPage';
import { ProgramRiskPage } from '../pages/ProgramRiskPage';
import { ProgramsPage } from '../pages/ProgramsPage';
import { AppShell, LoadingState } from '../components/ui';
import { useAuth } from '../context/AuthContext';

function ProtectedLayout() {
  const { user, isLoading, logout } = useAuth();

  if (isLoading) {
    return <LoadingState title="Loading your workspace" copy="Checking your account and restoring your forecasting workspace." />;
  }

  if (!user) {
    return <Navigate to="/auth" replace />;
  }

  return (
    <AppShell
      balance={user.balance}
      isAdmin={user.is_admin}
      onLogout={logout}
      subtitle="Calm signal, visible trust, and clear next steps."
      title="InhousePredictor"
      userName={user.name}
    >
      <Outlet />
    </AppShell>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/auth" element={<AuthPage />} />
      <Route element={<ProtectedLayout />}> 
        <Route path="/" element={<Navigate to="/forecast-questions" replace />} />
        <Route path="/forecast-questions" element={<ForecastQuestionsPage />} />
        <Route path="/forecast-questions/new" element={<CreateForecastQuestionPage />} />
        <Route path="/forecast-questions/:id" element={<ForecastQuestionPage />} />
        <Route path="/programs" element={<ProgramsPage />} />
        <Route path="/programs/:program" element={<ProgramRiskPage />} />
        <Route path="/markets" element={<MarketsPage />} />
        <Route path="/markets/:id" element={<MarketDetailPage />} />
        <Route path="/markets/:id/resolve" element={<ResolveMarketPage />} />
        <Route path="/markets/:id/dispute" element={<DisputePage />} />
        <Route path="/portfolio" element={<PortfolioPage />} />
        <Route path="/create" element={<CreateMarketPage />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/admin/disputes" element={<AdminDisputesPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
