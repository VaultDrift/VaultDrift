import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios';
import { User, File, Share } from '@/types';

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

          accessToken = response.data.token;
          refreshToken = response.data.refresh_token;

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

// Auth API
export const authApi = {
  login: async (username: string, password: string) => {
    const response = await api.post('/auth/login', { username, password });
    accessToken = response.data.token;
    refreshToken = response.data.refresh_token;
    localStorage.setItem('access_token', accessToken!);
    localStorage.setItem('refresh_token', refreshToken!);
    return response.data;
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
    return response.data;
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
  list: async (parentId?: string): Promise<File[]> => {
    const params = parentId ? { parent_id: parentId } : {};
    const response = await api.get('/files', { params });
    return response.data.files || response.data;
  },

  get: async (id: string): Promise<File> => {
    const response = await api.get(`/files/${id}`);
    return response.data;
  },

  create: async (data: Partial<File>): Promise<File> => {
    const response = await api.post('/files', data);
    return response.data;
  },

  update: async (id: string, data: Partial<File>): Promise<File> => {
    const response = await api.put(`/files/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/files/${id}`);
  },

  search: async (query: string): Promise<File[]> => {
    const response = await api.get('/files/search', { params: { q: query } });
    return response.data.files || response.data;
  },

  recent: async (): Promise<File[]> => {
    const response = await api.get('/files/recent');
    return response.data.files || response.data;
  },
};

// Upload API
export const uploadApi = {
  init: async (name: string, size: number, parentId?: string) => {
    const response = await api.post('/uploads', {
      name,
      size_bytes: size,
      parent_id: parentId,
    });
    return response.data;
  },

  uploadChunk: async (uploadId: string, index: number, data: Blob, onProgress?: (progress: number) => void) => {
    const formData = new FormData();
    formData.append('chunk', data);

    const response = await api.put(`/uploads/${uploadId}/chunks/${index}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total));
        }
      },
    });
    return response.data;
  },

  complete: async (uploadId: string) => {
    const response = await api.post(`/uploads/${uploadId}/complete`);
    return response.data;
  },
};

// Shares API
export const sharesApi = {
  list: async (): Promise<Share[]> => {
    const response = await api.get('/shares');
    return response.data.shares || response.data;
  },

  create: async (fileId: string, options: Partial<Share>): Promise<Share> => {
    const response = await api.post('/shares', { file_id: fileId, ...options });
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/shares/${id}`);
  },
};

export default api;
