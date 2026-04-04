import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios';
import { User, File, Share, ReceivedShare, Breadcrumb } from '@/types';

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || '';

// Token management
let accessToken: string | null = localStorage.getItem('access_token');
let refreshToken: string | null = localStorage.getItem('refresh_token');

// Axios instance
const api: AxiosInstance = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000,
});

// Request interceptor - add auth token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (accessToken && config.headers) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor - handle token refresh
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && originalRequest) {
      // Try to refresh token
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_BASE_URL}/api/v1/auth/refresh`, {
            refresh_token: refreshToken,
          });

          const tokens = unwrap<{ token: string; refresh_token: string }>(response.data);
          accessToken = tokens.token;
          refreshToken = tokens.refresh_token;

          localStorage.setItem('access_token', accessToken!);
          localStorage.setItem('refresh_token', refreshToken!);

          // Retry original request
          originalRequest.headers.Authorization = `Bearer ${accessToken}`;
          return api(originalRequest);
        } catch (refreshError) {
          // Refresh failed, logout
          logout();
          window.location.href = '/login';
          return Promise.reject(refreshError);
        }
      } else {
        logout();
        window.location.href = '/login';
      }
    }

    return Promise.reject(error);
  }
);

// unwrap extracts the payload from SuccessResponse {success: true, data: ...}
function unwrap<T>(responseData: unknown): T {
  const d = responseData as { success?: boolean; data?: unknown };
  if (d && typeof d === 'object' && 'success' in d && 'data' in d) {
    return d.data as T;
  }
  return responseData as T;
}

// Admin types
interface AdminStats {
  users?: { total: number; active: number };
  storage?: { total_files: number; total_bytes: number; total_chunks: number; active_shares: number; storage_backend: string; [key: string]: unknown };
}

interface SystemHealth {
  status?: string;
  system?: { goroutines: number; cpu_cores: number; memory_alloc_mb: number; gc_count: number; [key: string]: unknown };
  database?: string;
  storage?: string;
  [key: string]: unknown;
}

export interface SyncDevice {
  id: string;
  name: string;
  device_type: string;
  os: string;
  last_sync_at: string | null;
  is_active: boolean;
  [key: string]: unknown;
}

export interface SyncStatus {
  status?: string;
  total_bytes_synced?: number;
  pending_files?: number;
  [key: string]: unknown;
}

interface LoginResponse {
  token?: string;
  refresh_token?: string;
  expires_at?: string;
  session_id?: string;
  username?: string;
  requires_totp?: boolean;
  totp_session?: string;
}

// Auth API
export const authApi = {
  login: async (username: string, password: string): Promise<LoginResponse> => {
    const response = await api.post('/auth/login', { username, password });
    const data = unwrap<LoginResponse>(response.data);
    if (data.token && data.refresh_token) {
      accessToken = data.token;
      refreshToken = data.refresh_token;
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
    }
    return data;
  },

  verifyTotp: async (totpSession: string, code: string): Promise<LoginResponse> => {
    const response = await api.post('/auth/totp/verify', { session: totpSession, code });
    const data = unwrap<LoginResponse>(response.data);
    if (data.token && data.refresh_token) {
      accessToken = data.token;
      refreshToken = data.refresh_token;
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
    }
    return data;
  },

  logout: async () => {
    try {
      await api.post('/auth/logout');
    } finally {
      logout();
    }
  },

  getMe: async (): Promise<User> => {
    const response = await api.get('/users/me');
    return unwrap<User>(response.data);
  },

  getProfile: async (): Promise<User> => {
    const response = await api.get('/user/profile');
    return unwrap<User>(response.data);
  },

  updateProfile: async (data: { display_name?: string; email?: string }): Promise<User> => {
    const response = await api.put('/user/profile', data);
    return unwrap<User>(response.data);
  },

  changePassword: async (currentPassword: string, newPassword: string): Promise<void> => {
    await api.put('/user/password', { current_password: currentPassword, new_password: newPassword });
  },
};

function logout() {
  accessToken = null;
  refreshToken = null;
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
}

// Files API
export const filesApi = {
  list: async (parentId?: string, options?: { limit?: number; offset?: number }): Promise<File[]> => {
    const params: Record<string, string | number> = {};
    if (parentId) params.parent_id = parentId;
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    const response = await api.get('/files', { params });
    const data = unwrap<{ files?: File[] } | File[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { files?: File[] }).files || [];
  },

  get: async (id: string): Promise<File> => {
    const response = await api.get(`/files/${id}`);
    return unwrap<File>(response.data);
  },

  create: async (data: Partial<File>): Promise<File> => {
    const response = await api.post('/files', data);
    return unwrap<File>(response.data);
  },

  update: async (id: string, data: Partial<File>): Promise<File> => {
    const response = await api.put(`/files/${id}`, data);
    return unwrap<File>(response.data);
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/files/${id}`);
  },

  search: async (query: string): Promise<File[]> => {
    const response = await api.get('/search', { params: { q: query } });
    const data = unwrap<{ files?: File[] } | File[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { files?: File[] }).files || [];
  },

  recent: async (options?: { limit?: number; offset?: number }): Promise<File[]> => {
    const params: Record<string, string | number> = {};
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    const response = await api.get('/files/recent', { params });
    const data = unwrap<{ files?: File[] } | File[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { files?: File[] }).files || [];
  },

  getBreadcrumbs: async (folderId: string): Promise<Breadcrumb[]> => {
    const response = await api.get(`/folders/${folderId}/breadcrumbs`);
    const data = unwrap<{ breadcrumbs?: Breadcrumb[] }>(response.data);
    return data.breadcrumbs || [];
  },

  download: async (id: string): Promise<void> => {
    const response = await api.get(`/files/${id}/download`, { responseType: 'blob' });
    const blob = new Blob([response.data]);
    const contentDisposition = response.headers['content-disposition'];
    let filename = 'download';
    if (contentDisposition) {
      const match = contentDisposition.match(/filename="?(.+?)"?$/);
      if (match) filename = match[1];
    }
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  },
};

// Upload API
export const uploadApi = {
  init: async (name: string, size: number, parentId?: string, signal?: AbortSignal) => {
    const response = await api.post('/uploads', {
      name,
      size,
      parent_id: parentId,
    }, { signal });
    const data = unwrap<{ session_id: string; chunk_size: number; total_chunks: number }>(response.data);
    return {
      upload_id: data.session_id,
      chunk_size: data.chunk_size,
      total_chunks: data.total_chunks,
    };
  },

  uploadChunk: async (uploadId: string, index: number, data: Blob, onProgress?: (progress: number) => void, signal?: AbortSignal) => {
    const response = await api.put(`/uploads/${uploadId}/chunks/${index}`, data, {
      headers: { 'Content-Type': 'application/octet-stream' },
      signal,
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total));
        }
      },
    });
    return unwrap(response.data);
  },

  complete: async (uploadId: string, signal?: AbortSignal) => {
    const response = await api.post(`/uploads/${uploadId}/complete`, null, { signal });
    return unwrap(response.data);
  },
};

// Shares API
export const sharesApi = {
  list: async (fileId?: string): Promise<Share[]> => {
    const params = fileId ? { file_id: fileId } : {};
    const response = await api.get('/shares', { params });
    const data = unwrap<{ shares?: Share[] } | Share[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { shares?: Share[] }).shares || [];
  },

  getReceived: async (): Promise<ReceivedShare[]> => {
    const response = await api.get('/shares/received');
    const data = unwrap<{ shares?: ReceivedShare[] } | ReceivedShare[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { shares?: ReceivedShare[] }).shares || [];
  },

  create: async (fileId: string, options: Partial<Share>): Promise<Share> => {
    const response = await api.post(`/files/${fileId}/shares`, options);
    return unwrap<Share>(response.data);
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/shares/${id}`);
  },
};

// Admin API
export const adminApi = {
  getUsers: async (): Promise<User[]> => {
    const response = await api.get('/admin/users');
    const data = unwrap<{ users?: User[] } | User[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { users?: User[] }).users || [];
  },

  createUser: async (data: Partial<User>): Promise<User> => {
    const response = await api.post('/admin/users', data);
    return unwrap<User>(response.data);
  },

  updateUser: async (id: string, data: Partial<User>): Promise<User> => {
    const response = await api.put(`/admin/users/${id}`, data);
    return unwrap<User>(response.data);
  },

  deleteUser: async (id: string): Promise<void> => {
    await api.delete(`/admin/users/${id}`);
  },

  getStats: async (): Promise<AdminStats> => {
    const response = await api.get('/admin/stats');
    return unwrap<AdminStats>(response.data);
  },

  getSystemHealth: async (): Promise<SystemHealth> => {
    const response = await api.get('/admin/health');
    return unwrap<SystemHealth>(response.data);
  },
};

// Sync API
export const syncApi = {
  getDevices: async (): Promise<SyncDevice[]> => {
    const response = await api.get('/sync/devices');
    const data = unwrap<{ devices?: SyncDevice[] } | SyncDevice[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { devices?: SyncDevice[] }).devices || [];
  },

  getSessions: async () => {
    const response = await api.get('/sync/sessions');
    const data = unwrap<{ sessions?: unknown[] } | unknown[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { sessions?: unknown[] }).sessions || [];
  },

  revokeDevice: async (deviceId: string): Promise<void> => {
    await api.delete(`/sync/devices/${deviceId}`);
  },

  getSyncStatus: async (): Promise<SyncStatus> => {
    const response = await api.get('/sync/status');
    return unwrap<SyncStatus>(response.data);
  },
};

// Preview API
export const previewApi = {
  getStreamUrl: (fileId: string) => {
    const base = API_BASE_URL || '';
    return `${base}/api/v1/files/${fileId}/stream`;
  },

  getThumbnailUrl: (fileId: string) => {
    const base = API_BASE_URL || '';
    return `${base}/api/v1/thumbnails/${fileId}`;
  },

  getHlsPlaylistUrl: (fileId: string) => {
    const base = API_BASE_URL || '';
    return `${base}/api/v1/media/${fileId}/playlist.m3u8`;
  },
};

// Trash API
export const trashApi = {
  list: async (options?: { limit?: number; offset?: number }): Promise<File[]> => {
    const params: Record<string, string | number> = {};
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    const response = await api.get('/trash', { params });
    const data = unwrap<{ items?: File[] } | File[]>(response.data);
    if (Array.isArray(data)) return data;
    return (data as { items?: File[] }).items || [];
  },

  restore: async (id: string): Promise<void> => {
    await api.post(`/trash/${id}/restore`);
  },

  permanentDelete: async (id: string): Promise<void> => {
    await api.delete(`/trash/${id}`);
  },

  emptyTrash: async (): Promise<void> => {
    await api.delete('/trash');
  },
};

// Helper: fetch authenticated blob URL
export async function getAuthBlobUrl(url: string): Promise<string> {
  const response = await api.get(url, { responseType: 'blob' });
  return URL.createObjectURL(new Blob([response.data]));
}

export default api;
