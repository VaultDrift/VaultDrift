import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import { useNavigation } from '@react-navigation/native';
import { useAuthStore } from '../../stores/authStore';

export function ServerSetupScreen() {
  const navigation = useNavigation();
  const { serverUrl, setServerUrl } = useAuthStore();

  const [url, setUrl] = useState(serverUrl || 'http://localhost:8080');
  const [testing, setTesting] = useState(false);

  const handleSave = async () => {
    if (!url.trim()) {
      Alert.alert('Error', 'Please enter a server URL');
      return;
    }

    // Basic URL validation
    let serverUrl = url.trim();
    if (!serverUrl.startsWith('http://') && !serverUrl.startsWith('https://')) {
      serverUrl = 'https://' + serverUrl;
    }

    setTesting(true);
    try {
      // Test connection
      const response = await fetch(`${serverUrl}/health`, {
        method: 'GET',
        timeout: 5000,
      } as any);

      if (response.ok) {
        setServerUrl(serverUrl);
        Alert.alert('Success', 'Server connection successful');
        navigation.goBack();
      } else {
        Alert.alert('Error', 'Server returned an error. Please check the URL.');
      }
    } catch (error) {
      Alert.alert(
        'Connection Failed',
        'Could not connect to server. Please check the URL and try again.'
      );
    } finally {
      setTesting(false);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => navigation.goBack()}>
          <Ionicons name="close" size={28} color="#f8fafc" />
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Server Settings</Text>
        <View style={{ width: 28 }} />
      </View>

      <View style={styles.content}>
        <View style={styles.iconContainer}>
          <Ionicons name="globe" size={64} color="#3b82f6" />
        </View>

        <Text style={styles.description}>
          Enter your VaultDrift server URL. This should be the address where your server is hosted.
        </Text>

        <View style={styles.inputContainer}>
          <Text style={styles.label}>Server URL</Text>
          <TextInput
            style={styles.input}
            placeholder="https://vaultdrift.example.com"
            placeholderTextColor="#64748b"
            value={url}
            onChangeText={setUrl}
            autoCapitalize="none"
            keyboardType="url"
            autoCorrect={false}
          />
        </View>

        <View style={styles.examplesContainer}>
          <Text style={styles.examplesTitle}>Examples:</Text>
          <Text style={styles.example}>• http://192.168.1.100:8080 (local network)</Text>
          <Text style={styles.example}>• https://vaultdrift.example.com</Text>
          <Text style={styles.example}>• http://10.0.2.2:8080 (Android emulator)</Text>
        </View>

        <TouchableOpacity
          style={[styles.saveButton, testing && styles.saveButtonDisabled]}
          onPress={handleSave}
          disabled={testing}
        >
          {testing ? (
            <ActivityIndicator color="#ffffff" />
          ) : (
            <>
              <Ionicons name="checkmark" size={20} color="#ffffff" />
              <Text style={styles.saveButtonText}>Save & Test</Text>
            </>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#f8fafc',
  },
  content: {
    flex: 1,
    padding: 24,
  },
  iconContainer: {
    alignItems: 'center',
    marginBottom: 24,
  },
  description: {
    color: '#94a3b8',
    fontSize: 14,
    textAlign: 'center',
    marginBottom: 32,
  },
  inputContainer: {
    marginBottom: 24,
  },
  label: {
    color: '#f8fafc',
    fontSize: 14,
    fontWeight: '500',
    marginBottom: 8,
  },
  input: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    paddingHorizontal: 16,
    paddingVertical: 14,
    color: '#f8fafc',
    fontSize: 16,
  },
  examplesContainer: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 16,
    marginBottom: 24,
  },
  examplesTitle: {
    color: '#94a3b8',
    fontSize: 12,
    fontWeight: '600',
    marginBottom: 8,
    textTransform: 'uppercase',
  },
  example: {
    color: '#64748b',
    fontSize: 13,
    marginBottom: 4,
  },
  saveButton: {
    backgroundColor: '#3b82f6',
    borderRadius: 12,
    height: 56,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
  },
  saveButtonDisabled: {
    opacity: 0.7,
  },
  saveButtonText: {
    color: '#ffffff',
    fontSize: 16,
    fontWeight: '600',
    marginLeft: 8,
  },
});
