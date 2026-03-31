import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Switch,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import { useAuthStore } from '../../stores/authStore';
import { useSettingsStore } from '../../stores/settingsStore';
import { biometricAuth } from '../../utils/biometric';

export function SettingsScreen() {
  const { user, logout, biometricEnabled, enableBiometric, disableBiometric } = useAuthStore();
  const {
    theme,
    autoUploadPhotos,
    autoUploadOnWifiOnly,
    notificationsEnabled,
    setTheme,
    setAutoUploadPhotos,
    setAutoUploadOnWifiOnly,
    setNotificationsEnabled,
  } = useSettingsStore();

  const handleLogout = () => {
    Alert.alert(
      'Logout',
      'Are you sure you want to logout?',
      [
        { text: 'Cancel', style: 'cancel' },
        { text: 'Logout', style: 'destructive', onPress: () => logout() },
      ]
    );
  };

  const toggleBiometric = async () => {
    if (biometricEnabled) {
      await disableBiometric();
    } else {
      const available = await biometricAuth.isAvailable();
      if (available) {
        await enableBiometric();
      } else {
        Alert.alert('Not Available', 'Biometric authentication is not available on this device.');
      }
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <SafeAreaView style={styles.container} edges={['bottom']}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Profile Section */}
        <View style={styles.profileSection}>
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>
              {user?.name?.charAt(0).toUpperCase() || 'U'}
            </Text>
          </View>
          <View style={styles.profileInfo}>
            <Text style={styles.name}>{user?.name || 'User'}</Text>
            <Text style={styles.email}>{user?.email}</Text>
          </View>
        </View>

        {/* Storage Usage */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Storage</Text>
          <View style={styles.storageCard}>
            <View style={styles.storageHeader}>
              <Ionicons name="cloud" size={24} color="#3b82f6" />
              <Text style={styles.storageText}>
                {formatSize(user?.storageUsed || 0)} of {formatSize(user?.storageQuota || 0)}
              </Text>
            </View>
            <View style={styles.progressBar}>
              <View
                style={[
                  styles.progressFill,
                  {
                    width: `${Math.min(
                      ((user?.storageUsed || 0) / (user?.storageQuota || 1)) * 100,
                      100
                    )}%`,
                  },
                ]}
              />
            </View>
          </View>
        </View>

        {/* Security Section */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Security</Text>
          <View style={styles.card}>
            <SettingItem
              icon="finger-print"
              color="#10b981"
              title="Biometric Unlock"
              subtitle="Use Face ID / Fingerprint"
              rightElement={
                <Switch
                  value={biometricEnabled}
                  onValueChange={toggleBiometric}
                  trackColor={{ false: '#334155', true: '#3b82f6' }}
                />
              }
            />
          </View>
        </View>

        {/* Preferences Section */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Preferences</Text>
          <View style={styles.card}>
            <SettingItem
              icon="moon"
              color="#8b5cf6"
              title="Dark Mode"
              subtitle="Automatic"
              rightElement={
                <Switch
                  value={theme === 'dark'}
                  onValueChange={(value) => setTheme(value ? 'dark' : 'light')}
                  trackColor={{ false: '#334155', true: '#3b82f6' }}
                />
              }
            />
            <SettingItem
              icon="image"
              color="#f59e0b"
              title="Auto Upload Photos"
              subtitle="Upload new photos automatically"
              rightElement={
                <Switch
                  value={autoUploadPhotos}
                  onValueChange={setAutoUploadPhotos}
                  trackColor={{ false: '#334155', true: '#3b82f6' }}
                />
              }
              showBorder
            />
            {autoUploadPhotos && (
              <SettingItem
                icon="wifi"
                color="#3b82f6"
                title="Wi-Fi Only"
                subtitle="Only upload on Wi-Fi"
                rightElement={
                  <Switch
                    value={autoUploadOnWifiOnly}
                    onValueChange={setAutoUploadOnWifiOnly}
                    trackColor={{ false: '#334155', true: '#3b82f6' }}
                  />
                }
                showBorder
              />
            )}
            <SettingItem
              icon="notifications"
              color="#ef4444"
              title="Notifications"
              subtitle="Push notifications"
              rightElement={
                <Switch
                  value={notificationsEnabled}
                  onValueChange={setNotificationsEnabled}
                  trackColor={{ false: '#334155', true: '#3b82f6' }}
                />
              }
              showBorder
            />
          </View>
        </View>

        {/* About Section */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>About</Text>
          <View style={styles.card}>
            <SettingItem
              icon="information-circle"
              color="#64748b"
              title="Version"
              subtitle="1.0.0"
            />
            <SettingItem
              icon="shield-checkmark"
              color="#10b981"
              title="Privacy Policy"
              onPress={() => {}}
              showBorder
            />
            <SettingItem
              icon="document-text"
              color="#3b82f6"
              title="Terms of Service"
              onPress={() => {}}
              showBorder
            />
          </View>
        </View>

        {/* Logout */}
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Ionicons name="log-out" size={20} color="#ef4444" />
          <Text style={styles.logoutText}>Logout</Text>
        </TouchableOpacity>

        <View style={styles.footer}>
          <Text style={styles.footerText}>VaultDrift Mobile v1.0.0</Text>
          <Text style={styles.footerSubtext}>Secure Cloud Storage</Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

interface SettingItemProps {
  icon: keyof typeof Ionicons.glyphMap;
  color: string;
  title: string;
  subtitle?: string;
  rightElement?: React.ReactNode;
  onPress?: () => void;
  showBorder?: boolean;
}

function SettingItem({ icon, color, title, subtitle, rightElement, onPress, showBorder }: SettingItemProps) {
  return (
    <TouchableOpacity
      style={[styles.settingItem, showBorder && styles.settingItemBorder]}
      onPress={onPress}
      disabled={!onPress}
    >
      <View style={[styles.iconWrapper, { backgroundColor: color + '20' }]}>
        <Ionicons name={icon} size={20} color={color} />
      </View>
      <View style={styles.settingInfo}>
        <Text style={styles.settingTitle}>{title}</Text>
        {subtitle && <Text style={styles.settingSubtitle}>{subtitle}</Text>}
      </View>
      {rightElement}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  scrollContent: {
    padding: 16,
  },
  profileSection: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 24,
  },
  avatar: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: '#3b82f6',
    justifyContent: 'center',
    alignItems: 'center',
  },
  avatarText: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#ffffff',
  },
  profileInfo: {
    marginLeft: 16,
  },
  name: {
    fontSize: 20,
    fontWeight: '600',
    color: '#f8fafc',
  },
  email: {
    fontSize: 14,
    color: '#94a3b8',
    marginTop: 2,
  },
  section: {
    marginBottom: 24,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 8,
    marginLeft: 4,
  },
  storageCard: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 16,
    borderWidth: 1,
    borderColor: '#334155',
  },
  storageHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  storageText: {
    color: '#f8fafc',
    fontSize: 14,
    marginLeft: 12,
  },
  progressBar: {
    height: 6,
    backgroundColor: '#334155',
    borderRadius: 3,
  },
  progressFill: {
    height: '100%',
    backgroundColor: '#3b82f6',
    borderRadius: 3,
  },
  card: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    overflow: 'hidden',
  },
  settingItem: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
  },
  settingItemBorder: {
    borderTopWidth: 1,
    borderTopColor: '#334155',
  },
  iconWrapper: {
    width: 36,
    height: 36,
    borderRadius: 10,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  settingInfo: {
    flex: 1,
  },
  settingTitle: {
    color: '#f8fafc',
    fontSize: 15,
  },
  settingSubtitle: {
    color: '#64748b',
    fontSize: 12,
    marginTop: 2,
  },
  logoutButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 16,
    marginTop: 8,
    borderWidth: 1,
    borderColor: '#ef4444',
  },
  logoutText: {
    color: '#ef4444',
    fontSize: 16,
    fontWeight: '600',
    marginLeft: 8,
  },
  footer: {
    alignItems: 'center',
    marginTop: 24,
    marginBottom: 16,
  },
  footerText: {
    color: '#64748b',
    fontSize: 12,
  },
  footerSubtext: {
    color: '#475569',
    fontSize: 11,
    marginTop: 2,
  },
});
