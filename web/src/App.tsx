import { Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from '@/stores/auth';
import { Layout } from '@/components/Layout';
import { LoginPage } from '@/pages/Login';
import { FilesPage } from '@/pages/Files';
import { SharedPage } from '@/pages/Shared';
import { RecentPage } from '@/pages/Recent';
import { TrashPage } from '@/pages/Trash';
import { SettingsPage } from '@/pages/Settings';

function App() {
  const { isAuthenticated, fetchUser } = useAuthStore();

  useEffect(() => {
    // Try to restore session on mount
    const token = localStorage.getItem('access_token');
    if (token && !isAuthenticated) {
      fetchUser();
    }
  }, [fetchUser, isAuthenticated]);

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/files" replace />} />
        <Route path="/files" element={<FilesPage />} />
        <Route path="/files/:folderId" element={<FilesPage />} />
        <Route path="/shared" element={<SharedPage />} />
        <Route path="/recent" element={<RecentPage />} />
        <Route path="/trash" element={<TrashPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </Layout>
  );
}

export default App;
