import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  TouchableOpacity,
  RefreshControl,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '../../stores/authStore';
import { api } from '../../api/client';

interface ShareItem {
  id: string;
  file_id: string;
  file_name: string;
  token: string;
  expires_at: string | null;
  permission: 'read' | 'write';
  access_count: number;
}

export function SharedScreen() {
  const { serverUrl, token } = useAuthStore();

  const { data: shares, isLoading, refetch } = useQuery({
    queryKey: ['shares'],
    queryFn: async () => {
      const response = await api.get(`${serverUrl}/api/v1/shares`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      return response.data.shares as ShareItem[];
    },
    enabled: !!serverUrl && !!token,
  });

  const renderItem = ({ item }: { item: ShareItem }) => (
    <TouchableOpacity style={styles.shareItem}>
      <View style={styles.iconContainer}>
        <Ionicons name="share-outline" size={32} color="#10b981" />
      </View>
      <View style={styles.shareInfo}>
        <Text style={styles.fileName} numberOfLines={1}>
          {item.file_name}
        </Text>
        <Text style={styles.shareMeta}>
          {item.permission === 'write' ? 'Can edit' : 'View only'}
          {item.expires_at ? ` • Expires ${formatDate(item.expires_at)}` : ' • No expiration'}
        </Text>
        <Text style={styles.accessCount}>
          {item.access_count} access{item.access_count !== 1 ? 'es' : ''}
        </Text>
      </View>
      <TouchableOpacity style={styles.moreButton}>
        <Ionicons name="copy-outline" size={20} color="#3b82f6" />
      </TouchableOpacity>
    </TouchableOpacity>
  );

  const renderEmpty = () => (
    <View style={styles.emptyContainer}>
      <Ionicons name="share-social" size={64} color="#334155" />
      <Text style={styles.emptyText}>No shared files</Text>
      <Text style={styles.emptySubtext}>Share files with others</Text>
    </View>
  );

  return (
    <SafeAreaView style={styles.container} edges={['bottom']}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Shared</Text>
      </View>

      <FlatList
        data={shares || []}
        renderItem={renderItem}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={isLoading} onRefresh={refetch} tintColor="#3b82f6" />
        }
        ListEmptyComponent={renderEmpty}
      />
    </SafeAreaView>
  );
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  });
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  header: {
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#f8fafc',
  },
  listContent: {
    padding: 16,
    flexGrow: 1,
  },
  shareItem: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 12,
    marginBottom: 8,
  },
  iconContainer: {
    width: 48,
    height: 48,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  shareInfo: {
    flex: 1,
  },
  fileName: {
    color: '#f8fafc',
    fontSize: 16,
    fontWeight: '500',
    marginBottom: 2,
  },
  shareMeta: {
    color: '#94a3b8',
    fontSize: 12,
    marginBottom: 2,
  },
  accessCount: {
    color: '#64748b',
    fontSize: 11,
  },
  moreButton: {
    padding: 8,
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: 64,
  },
  emptyText: {
    color: '#94a3b8',
    fontSize: 18,
    fontWeight: '600',
    marginTop: 16,
  },
  emptySubtext: {
    color: '#64748b',
    fontSize: 14,
    marginTop: 8,
  },
});
