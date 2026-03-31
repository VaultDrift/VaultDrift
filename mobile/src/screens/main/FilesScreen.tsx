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

interface FileItem {
  id: string;
  name: string;
  type: 'file' | 'folder';
  size: number;
  mime_type: string;
  created_at: string;
  updated_at: string;
}

export function FilesScreen() {
  const { serverUrl, token } = useAuthStore();

  const { data: files, isLoading, refetch } = useQuery({
    queryKey: ['files'],
    queryFn: async () => {
      const response = await api.get(`${serverUrl}/api/v1/files`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      return response.data as FileItem[];
    },
    enabled: !!serverUrl && !!token,
  });

  const renderItem = ({ item }: { item: FileItem }) => (
    <TouchableOpacity style={styles.fileItem}>
      <View style={styles.iconContainer}>
        {item.type === 'folder' ? (
          <Ionicons name="folder" size={40} color="#f59e0b" />
        ) : (
          <Ionicons name="document" size={40} color="#3b82f6" />
        )}
      </View>
      <View style={styles.fileInfo}>
        <Text style={styles.fileName} numberOfLines={1}>
          {item.name}
        </Text>
        <Text style={styles.fileMeta}>
          {item.type === 'file' ? formatSize(item.size) : 'Folder'} • {formatDate(item.updated_at)}
        </Text>
      </View>
      <TouchableOpacity style={styles.moreButton}>
        <Ionicons name="ellipsis-vertical" size={20} color="#64748b" />
      </TouchableOpacity>
    </TouchableOpacity>
  );

  const renderEmpty = () => (
    <View style={styles.emptyContainer}>
      <Ionicons name="folder-open" size={64} color="#334155" />
      <Text style={styles.emptyText}>No files yet</Text>
      <Text style={styles.emptySubtext}>Upload your first file</Text>
    </View>
  );

  return (
    <SafeAreaView style={styles.container} edges={['bottom']}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>My Files</Text>
        <TouchableOpacity style={styles.addButton}>
          <Ionicons name="add" size={24} color="#ffffff" />
        </TouchableOpacity>
      </View>

      <FlatList
        data={files || []}
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

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
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
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#f8fafc',
  },
  addButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#3b82f6',
    justifyContent: 'center',
    alignItems: 'center',
  },
  listContent: {
    padding: 16,
    flexGrow: 1,
  },
  fileItem: {
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
  fileInfo: {
    flex: 1,
  },
  fileName: {
    color: '#f8fafc',
    fontSize: 16,
    fontWeight: '500',
    marginBottom: 4,
  },
  fileMeta: {
    color: '#64748b',
    fontSize: 12,
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
