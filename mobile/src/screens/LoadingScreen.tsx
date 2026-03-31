import React from 'react';
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native';

export function LoadingScreen() {
  return (
    <View style={styles.container}>
      <View style={styles.logoCircle}>
        <ActivityIndicator size="large" color="#3b82f6" />
      </View>
      <Text style={styles.text}>VaultDrift</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
    justifyContent: 'center',
    alignItems: 'center',
  },
  logoCircle: {
    width: 120,
    height: 120,
    borderRadius: 60,
    backgroundColor: '#1e293b',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
    borderWidth: 2,
    borderColor: '#334155',
  },
  text: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#f8fafc',
  },
});
