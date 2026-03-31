import { create } from 'zustand';
import * as SecureStore from 'expo-secure-store';
import { api } from '../api/client';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'user';
  storageUsed: number;
  storageQuota: number;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  biometricEnabled: boolean;
  serverUrl: string;

  // Actions
  login: (email: string, password: string, serverUrl: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshAuth: () => Promise<boolean>;
  checkAuth: () => Promise<boolean>;
  enableBiometric: () => Promise<void>;
  disableBiometric: () => Promise<void>;
  setServerUrl: (url: string) => void;
}

const TOKEN_KEY = 'vaultdrift_token';
const REFRESH_TOKEN_KEY = 'vaultdrift_refresh_token';
const USER_KEY = 'vaultdrift_user';
const SERVER_URL_KEY = 'vaultdrift_server_url';
const BIOMETRIC_ENABLED_KEY = 'vaultdrift_biometric_enabled';

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  refreshToken: null,
  user: null,
  isAuthenticated: false,
  biometricEnabled: false,
  serverUrl: '',

  login: async (email: string, password: string, serverUrl: string) => {
    try {
      const response = await api.post(`${serverUrl}/api/v1/auth/login`, {
        email,
        password,
      });

      const { token, refresh_token, user } = response.data;

      // Store securely
      await SecureStore.setItemAsync(TOKEN_KEY, token);
      await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refresh_token);
      await SecureStore.setItemAsync(USER_KEY, JSON.stringify(user));
      await SecureStore.setItemAsync(SERVER_URL_KEY, serverUrl);

      // Update API client
      api.defaults.baseURL = serverUrl;
      api.defaults.headers.common['Authorization'] = `Bearer ${token}`;

      set({
        token,
        refreshToken: refresh_token,
        user,
        isAuthenticated: true,
        serverUrl,
      });
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  },

  logout: async () => {
    try {
      // Call logout endpoint if token exists
      const { token, serverUrl } = get();
      if (token && serverUrl) {
        await api.post(`${serverUrl}/api/v1/auth/logout`, {}, {
          headers: { Authorization: `Bearer ${token}` }
        }).catch(() => {}); // Ignore errors
      }
    } finally {
      // Clear stored data
      await SecureStore.deleteItemAsync(TOKEN_KEY);
      await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
      await SecureStore.deleteItemAsync(USER_KEY);
      await SecureStore.deleteItemAsync(BIOMETRIC_ENABLED_KEY);

      // Clear API client
      delete api.defaults.headers.common['Authorization'];

      set({
        token: null,
        refreshToken: null,
        user: null,
        isAuthenticated: false,
        biometricEnabled: false,
      });
    }
  },

  refreshAuth: async () => {
    try {
      const { refreshToken, serverUrl } = get();
      if (!refreshToken || !serverUrl) return false;

      const response = await api.post(`${serverUrl}/api/v1/auth/refresh`, {
        refresh_token: refreshToken,
      });

      const { token, refresh_token } = response.data;

      await SecureStore.setItemAsync(TOKEN_KEY, token);
      await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refresh_token);

      api.defaults.headers.common['Authorization'] = `Bearer ${token}`;

      set({ token, refreshToken: refresh_token });
      return true;
    } catch (error) {
      console.error('Token refresh failed:', error);
      return false;
    }
  },

  checkAuth: async () => {
    try {
      const [token, refreshToken, userStr, serverUrl, biometricEnabled] = await Promise.all([
        SecureStore.getItemAsync(TOKEN_KEY),
        SecureStore.getItemAsync(REFRESH_TOKEN_KEY),
        SecureStore.getItemAsync(USER_KEY),
        SecureStore.getItemAsync(SERVER_URL_KEY),
        SecureStore.getItemAsync(BIOMETRIC_ENABLED_KEY),
      ]);

      if (token && serverUrl) {
        api.defaults.baseURL = serverUrl;
        api.defaults.headers.common['Authorization'] = `Bearer ${token}`;

        const user = userStr ? JSON.parse(userStr) : null;

        set({
          token,
          refreshToken,
          user,
          isAuthenticated: true,
          serverUrl,
          biometricEnabled: biometricEnabled === 'true',
        });
        return true;
      }

      return false;
    } catch (error) {
      console.error('Auth check failed:', error);
      return false;
    }
  },

  enableBiometric: async () => {
    await SecureStore.setItemAsync(BIOMETRIC_ENABLED_KEY, 'true');
    set({ biometricEnabled: true });
  },

  disableBiometric: async () => {
    await SecureStore.deleteItemAsync(BIOMETRIC_ENABLED_KEY);
    set({ biometricEnabled: false });
  },

  setServerUrl: (url: string) => {
    set({ serverUrl: url });
  },
}));
