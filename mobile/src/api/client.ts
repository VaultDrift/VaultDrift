import axios from 'axios';
import { Platform } from 'react-native';

export const api = axios.create({
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
    'X-Client-Version': '1.0.0',
    'X-Client-Platform': Platform.OS,
  },
});

// Response interceptor for token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // If 401 and not already retrying
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        // Import dynamically to avoid circular dependency
        const { useAuthStore } = require('../stores/authStore');
        const refreshed = await useAuthStore.getState().refreshAuth();

        if (refreshed) {
          const token = useAuthStore.getState().token;
          originalRequest.headers['Authorization'] = `Bearer ${token}`;
          return api(originalRequest);
        }
      } catch (refreshError) {
        console.error('Token refresh failed:', refreshError);
      }

      // If refresh failed, logout
      const { useAuthStore } = require('../stores/authStore');
      await useAuthStore.getState().logout();
    }

    return Promise.reject(error);
  }
);
