import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './context/AuthContext';
import { AppLayout } from './components/layout/AppLayout';

// Pages
import { Login } from './pages/auth/Login';
import { Register } from './pages/auth/Register';
import { StudentOverview } from './pages/student/Overview';
import { StudentChallenges } from './pages/student/Challenges';
import { StudentChallengeDetail } from './pages/student/ChallengeDetail';
import { StudentSubmissions } from './pages/student/Submissions';
import { StudentLeaderboards } from './pages/student/Leaderboards';
import { StudentActivityHeatmap } from './pages/student/ActivityHeatmap';
import { StudentProfile } from './pages/student/Profile';
import { StudentConnectedAccounts } from './pages/student/ConnectedAccounts';
import { StudentGroups } from './pages/student/Groups';

import { AdminDashboard } from './pages/admin/Dashboard';
import { AdminChallenges } from './pages/admin/Challenges';
import { AdminStudents } from './pages/admin/Students';
import { AdminGroups } from './pages/admin/Groups';
import { AdminConsistency } from './pages/admin/Consistency';
import { AdminAuditLogs } from './pages/admin/AuditLogs';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

const ProtectedRoute: React.FC<{ children: React.ReactNode; adminOnly?: boolean }> = ({
  children,
  adminOnly = false,
}) => {
  const { user, isLoading, isCollegeAdmin } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#fafafa] flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (adminOnly && !isCollegeAdmin) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
};

const RoleBasedHome: React.FC = () => {
  const { isCollegeAdmin } = useAuth();
  if (isCollegeAdmin) {
    return <AdminDashboard />;
  }
  return <StudentOverview />;
};

const StudentOnlyRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isCollegeAdmin } = useAuth();
  if (isCollegeAdmin) {
    return <Navigate to="/admin" replace />;
  }
  return <>{children}</>;
};

const GitHubOAuthRedirectHandler: React.FC = () => {
  return <Navigate to={`/settings/connected-accounts${window.location.search}`} replace />;
};

export const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            {/* Public Auth Routes */}
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />

            {/* Direct GitHub OAuth redirects to frontend port 5173 */}
            <Route path="/auth/github/callback" element={<GitHubOAuthRedirectHandler />} />
            <Route path="/github/callback" element={<GitHubOAuthRedirectHandler />} />

            {/* Authenticated Workspace */}
            <Route
              path="/"
              element={
                <ProtectedRoute>
                  <AppLayout />
                </ProtectedRoute>
              }
            >
              {/* Role-based home screen (AdminDashboard for college admins, StudentOverview for students) */}
              <Route index element={<RoleBasedHome />} />

              {/* Student Experience */}
              <Route path="challenges" element={<StudentOnlyRoute><StudentChallenges /></StudentOnlyRoute>} />
              <Route path="challenges/:id" element={<StudentOnlyRoute><StudentChallengeDetail /></StudentOnlyRoute>} />
              <Route path="submissions" element={<StudentOnlyRoute><StudentSubmissions /></StudentOnlyRoute>} />
              <Route path="groups" element={<StudentOnlyRoute><StudentGroups /></StudentOnlyRoute>} />
              <Route path="leaderboards" element={<StudentLeaderboards />} />
              <Route path="activity" element={<StudentOnlyRoute><StudentActivityHeatmap /></StudentOnlyRoute>} />
              <Route path="profile" element={<StudentOnlyRoute><StudentProfile /></StudentOnlyRoute>} />
              <Route path="settings/connected-accounts" element={<StudentOnlyRoute><StudentConnectedAccounts /></StudentOnlyRoute>} />

              {/* College Administrator Portal */}
              <Route
                path="admin"
                element={
                  <ProtectedRoute adminOnly>
                    <AdminDashboard />
                  </ProtectedRoute>
                }
              />
              <Route
                path="admin/challenges"
                element={
                  <ProtectedRoute adminOnly>
                    <AdminChallenges />
                  </ProtectedRoute>
                }
              />
              <Route
                path="admin/students"
                element={
                  <ProtectedRoute adminOnly>
                    <AdminStudents />
                  </ProtectedRoute>
                }
              />
              <Route
                path="admin/leaderboards"
                element={
                  <ProtectedRoute adminOnly>
                    <StudentLeaderboards />
                  </ProtectedRoute>
                }
              />
              <Route
                path="admin/consistency"
                element={
                  <ProtectedRoute adminOnly>
                    <AdminConsistency />
                  </ProtectedRoute>
                }
              />
              <Route
                path="admin/audit-logs"
                element={
                  <ProtectedRoute adminOnly>
                    <AdminAuditLogs />
                  </ProtectedRoute>
                }
              />
            </Route>

            {/* Fallback */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
};
