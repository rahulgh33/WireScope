// Main App Component with Routing and React Query

import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Layout } from '@/components/layout/Layout';
import { LoginPage } from '@/features/auth/LoginPage';
import { ProtectedRoute } from '@/features/auth/ProtectedRoute';
import { DashboardPage } from '@/features/dashboard/DashboardPage';
import { ClientsPage } from '@/features/clients/ClientsPage';
import { ClientDetailPage } from '@/features/clients/ClientDetailPage';
import { TargetsPage } from '@/features/targets/TargetsPage';
import { TargetDetailPage } from '@/features/targets/TargetDetailPage';
import { DiagnosticsPage } from '@/features/diagnostics/DiagnosticsPage';
import { AIAssistantPage } from '@/features/ai/AIAssistantPage';
import AdminPageComponent from '@/pages/AdminPage';
import { useAuthStore } from '@/stores/authStore';

// Create React Query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 30000, // 30 seconds
    },
  },
});

function AppRoutes() {
  const { isAuthenticated } = useAuthStore();

  return (
    <Routes>
      {/* Public Routes */}
      <Route
        path="/login"
        element={
          isAuthenticated ? <Navigate to="/" replace /> : <LoginPage />
        }
      />

      {/* Protected Routes */}
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Layout>
              <DashboardPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/clients"
        element={
          <ProtectedRoute>
            <Layout>
              <ClientsPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/clients/:clientId"
        element={
          <ProtectedRoute>
            <Layout>
              <ClientDetailPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/targets"
        element={
          <ProtectedRoute>
            <Layout>
              <TargetsPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/targets/:target"
        element={
          <ProtectedRoute>
            <Layout>
              <TargetDetailPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/diagnostics"
        element={
          <ProtectedRoute>
            <Layout>
              <DiagnosticsPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/ai"
        element={
          <ProtectedRoute>
            <Layout>
              <AIAssistantPage />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin"
        element={
          <ProtectedRoute requireRole={['admin']}>
            <Layout>
              <AdminPageComponent />
            </Layout>
          </ProtectedRoute>
        }
      />

      {/* 404 */}
      <Route
        path="*"
        element={
          <div className="min-h-screen flex items-center justify-center bg-background">
            <div className="text-center">
              <h1 className="text-4xl font-bold mb-2">404</h1>
              <p className="text-muted-foreground">Page not found</p>
            </div>
          </div>
        }
      />
    </Routes>
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </QueryClientProvider>
  );
}
