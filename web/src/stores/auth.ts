import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { User } from '@/types';
import { authApi } from '@/lib/api';
import { getWebSocketClient, disconnectWebSocket } from '@/lib/ws';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchUser: () => Promise<void>;
  initWebSocket: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (username: string, password: string) => {
        set({ isLoading: true });
        try {
          await authApi.login(username, password);
          const user = await authApi.getMe();
          set({ user, isAuthenticated: true });
          get().initWebSocket();
        } finally {
          set({ isLoading: false });
        }
      },

      logout: async () => {
        disconnectWebSocket();
        await authApi.logout();
        set({ user: null, isAuthenticated: false });
      },

      fetchUser: async () => {
        try {
          const user = await authApi.getMe();
          set({ user, isAuthenticated: true });
          get().initWebSocket();
        } catch {
          set({ user: null, isAuthenticated: false });
        }
      },

      initWebSocket: () => {
        const token = localStorage.getItem('access_token');
        if (token) {
          const ws = getWebSocketClient(token);
          ws?.connect();
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
    }
  )
);
