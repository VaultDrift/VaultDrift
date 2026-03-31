import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { FilesScreen } from '../screens/main/FilesScreen';
import { SharedScreen } from '../screens/main/SharedScreen';
import { UploadsScreen } from '../screens/main/UploadsScreen';
import { SettingsScreen } from '../screens/main/SettingsScreen';
import { useSettingsStore } from '../stores/settingsStore';

export type MainTabParamList = {
  Files: undefined;
  Shared: undefined;
  Uploads: undefined;
  Settings: undefined;
};

const Tab = createBottomTabNavigator<MainTabParamList>();

export function MainNavigator() {
  const { theme } = useSettingsStore();
  const isDark = theme === 'dark';

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        tabBarIcon: ({ focused, color, size }) => {
          let iconName: keyof typeof Ionicons.glyphMap = 'help';

          if (route.name === 'Files') {
            iconName = focused ? 'folder' : 'folder-outline';
          } else if (route.name === 'Shared') {
            iconName = focused ? 'share-social' : 'share-social-outline';
          } else if (route.name === 'Uploads') {
            iconName = focused ? 'cloud-upload' : 'cloud-upload-outline';
          } else if (route.name === 'Settings') {
            iconName = focused ? 'settings' : 'settings-outline';
          }

          return <Ionicons name={iconName} size={size} color={color} />;
        },
        tabBarActiveTintColor: '#3b82f6',
        tabBarInactiveTintColor: isDark ? '#94a3b8' : '#64748b',
        tabBarStyle: {
          backgroundColor: isDark ? '#1e293b' : '#ffffff',
          borderTopColor: isDark ? '#334155' : '#e2e8f0',
        },
        headerStyle: {
          backgroundColor: isDark ? '#0f172a' : '#ffffff',
        },
        headerTintColor: isDark ? '#f8fafc' : '#0f172a',
        tabBarLabelStyle: {
          fontSize: 12,
        },
      })}
    >
      <Tab.Screen
        name="Files"
        component={FilesScreen}
        options={{ title: 'My Files' }}
      />
      <Tab.Screen
        name="Shared"
        component={SharedScreen}
        options={{ title: 'Shared' }}
      />
      <Tab.Screen
        name="Uploads"
        component={UploadsScreen}
        options={{ title: 'Uploads' }}
      />
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: 'Settings' }}
      />
    </Tab.Navigator>
  );
}
