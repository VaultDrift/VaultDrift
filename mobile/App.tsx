import React, { useEffect, useState } from 'react';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { NavigationContainer } from '@react-navigation/native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { StyleSheet } from 'react-native';

import { useAuthStore } from './src/stores/authStore';
import { useSettingsStore } from './src/stores/settingsStore';
import { RootNavigator } from './src/navigation/RootNavigator';
import { LoadingScreen } from './src/screens/LoadingScreen';
import { biometricAuth } from './src/utils/biometric';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

export default function App() {
  const [isReady, setIsReady] = useState(false);
  const { token, biometricEnabled, checkAuth } = useAuthStore();
  const { theme, loadSettings } = useSettingsStore();

  useEffect(() => {
    async function init() {
      // Load stored settings
      await loadSettings();

      // Check authentication status
      const isAuthenticated = await checkAuth();

      // If authenticated and biometric is enabled, prompt for it
      if (isAuthenticated && biometricEnabled) {
        const success = await biometricAuth.authenticate();
        if (!success) {
          // If biometric fails, still allow app to open but may lock features
          console.log('Biometric authentication failed or cancelled');
        }
      }

      setIsReady(true);
    }

    init();
  }, []);

  if (!isReady) {
    return <LoadingScreen />;
  }

  return (
    <GestureHandlerRootView style={styles.container}>
      <SafeAreaProvider>
        <QueryClientProvider client={queryClient}>
          <NavigationContainer>
            <StatusBar style={theme === 'dark' ? 'light' : 'dark'} />
            <RootNavigator />
          </NavigationContainer>
        </QueryClientProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
