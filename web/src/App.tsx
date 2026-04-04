import { Routes, Route, Navigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { Layout } from '@/components/Layout';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { LoginPage } from '@/pages/Login';
import { FilesPage } from '@/pages/Files';
import { SharedPage } from '@/pages/Shared';
import { RecentPage } from '@/pages/Recent';
import { TrashPage } from '@/pages/Trash';
import { SettingsPage } from '@/pages/Settings';
import { AdminPage } from '@/pages/Admin';
import { SyncPage } from '@/pages/Sync';
import { NotFoundPage } from '@/pages/NotFound';

function App() {
  const { isAuthenticated, user } = useAuthStore();
  const [isInitializing, setIsInitializing] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (token) {
      useAuthStore.getState().fetchUser().finally(() => setIsInitializing(false));
    } else {
      setIsInitializing(false);
    }
  }, []);

  if (isInitializing) {
    return (
      <div className="h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <ErrorBoundary>
        <LoginPage />
      </ErrorBoundary>
    );
  }

  const isAdmin = user?.role === 'admin';

  return (
    <Layout>
      <ErrorBoundary>
        <Routes>
        <Route path="/" element={<Navigate to="/files" replace />} />
        <Route path="/files" element={<FilesPage />} />
        <Route path="/files/:folderId" element={<FilesPage />} />
        <Route path="/shared" element={<SharedPage />} />
        <Route path="/recent" element={<RecentPage />} />
        <Route path="/trash" element={<TrashPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/sync" element={<SyncPage />} />
        {isAdmin && <Route path="/admin" element={<AdminPage />} />}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      </ErrorBoundary>
    </Layout>
  );
}

export default App;
