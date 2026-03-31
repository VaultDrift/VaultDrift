import { create } from 'zustand';
import AsyncStorage from '@react-native-async-storage/async-storage';

interface SettingsState {
  theme: 'light' | 'dark' | 'system';
  language: string;
  autoUploadPhotos: boolean;
  autoUploadOnWifiOnly: boolean;
  downloadQuality: 'original' | 'high' | 'medium' | 'low';
  notificationsEnabled: boolean;
  cacheSize: number;

  // Actions
  loadSettings: () => Promise<void>;
  setTheme: (theme: 'light' | 'dark' | 'system') => Promise<void>;
  setLanguage: (lang: string) => Promise<void>;
  setAutoUploadPhotos: (enabled: boolean) => Promise<void>;
  setAutoUploadOnWifiOnly: (enabled: boolean) => Promise<void>;
  setDownloadQuality: (quality: 'original' | 'high' | 'medium' | 'low') => Promise<void>;
  setNotificationsEnabled: (enabled: boolean) => Promise<void>;
  clearCache: () => Promise<void>;
}

const SETTINGS_KEY = 'vaultdrift_settings';

const defaultSettings = {
  theme: 'system' as const,
  language: 'en',
  autoUploadPhotos: false,
  autoUploadOnWifiOnly: true,
  downloadQuality: 'high' as const,
  notificationsEnabled: true,
  cacheSize: 0,
};

export const useSettingsStore = create<SettingsState>((set, get) => ({
  ...defaultSettings,

  loadSettings: async () => {
    try {
      const stored = await AsyncStorage.getItem(SETTINGS_KEY);
      if (stored) {
        const settings = JSON.parse(stored);
        set({ ...defaultSettings, ...settings });
      }
    } catch (error) {
      console.error('Failed to load settings:', error);
    }
  },

  setTheme: async (theme) => {
    await saveSetting('theme', theme);
    set({ theme });
  },

  setLanguage: async (language) => {
    await saveSetting('language', language);
    set({ language });
  },

  setAutoUploadPhotos: async (autoUploadPhotos) => {
    await saveSetting('autoUploadPhotos', autoUploadPhotos);
    set({ autoUploadPhotos });
  },

  setAutoUploadOnWifiOnly: async (autoUploadOnWifiOnly) => {
    await saveSetting('autoUploadOnWifiOnly', autoUploadOnWifiOnly);
    set({ autoUploadOnWifiOnly });
  },

  setDownloadQuality: async (downloadQuality) => {
    await saveSetting('downloadQuality', downloadQuality);
    set({ downloadQuality });
  },

  setNotificationsEnabled: async (notificationsEnabled) => {
    await saveSetting('notificationsEnabled', notificationsEnabled);
    set({ notificationsEnabled });
  },

  clearCache: async () => {
    // Clear file cache
    set({ cacheSize: 0 });
  },
}));

async function saveSetting(key: string, value: any) {
  try {
    const stored = await AsyncStorage.getItem(SETTINGS_KEY);
    const settings = stored ? JSON.parse(stored) : {};
    settings[key] = value;
    await AsyncStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
  } catch (error) {
    console.error('Failed to save setting:', error);
  }
}
