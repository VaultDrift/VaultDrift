import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  TouchableOpacity,
  ProgressBarAndroid,
  ProgressViewIOS,
  Platform,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import * as DocumentPicker from 'expo-document-picker';
import * as ImagePicker from 'expo-image-picker';

interface UploadTask {
  id: string;
  fileName: string;
  progress: number;
  status: 'pending' | 'uploading' | 'completed' | 'failed';
  size: number;
  uploadedBytes: number;
}

// Mock upload tasks for demo
const mockUploads: UploadTask[] = [
  {
    id: '1',
    fileName: 'document.pdf',
    progress: 75,
    status: 'uploading',
    size: 5242880,
    uploadedBytes: 3932160,
  },
];

export function UploadsScreen() {
  const pickDocument = async () => {
    try {
      const result = await DocumentPicker.getDocumentAsync({
        type: '*/*',
        copyToCacheDirectory: true,
      });

      if (result.canceled === false) {
        console.log('Picked document:', result.assets[0]);
        // Start upload
      }
    } catch (error) {
      console.error('Error picking document:', error);
    }
  };

  const pickImage = async () => {
    try {
      const result = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ImagePicker.MediaTypeOptions.All,
        allowsEditing: false,
        quality: 1,
      });

      if (!result.canceled) {
        console.log('Picked image:', result.assets[0]);
        // Start upload
      }
    } catch (error) {
      console.error('Error picking image:', error);
    }
  };

  const takePhoto = async () => {
    try {
      const result = await ImagePicker.launchCameraAsync({
        allowsEditing: false,
        quality: 1,
      });

      if (!result.canceled) {
        console.log('Taken photo:', result.assets[0]);
        // Start upload
      }
    } catch (error) {
      console.error('Error taking photo:', error);
    }
  };

  const renderItem = ({ item }: { item: UploadTask }) => (
    <View style={styles.uploadItem}>
      <View style={styles.iconContainer}>
        <Ionicons
          name={getStatusIcon(item.status)}
          size={32}
          color={getStatusColor(item.status)}
        />
      </View>
      <View style={styles.uploadInfo}>
        <Text style={styles.fileName} numberOfLines={1}>
          {item.fileName}
        </Text>
        <Text style={styles.uploadMeta}>
          {formatSize(item.uploadedBytes)} of {formatSize(item.size)}
        </Text>
        {item.status === 'uploading' && (
          <View style={styles.progressContainer}>
            {Platform.OS === 'android' ? (
              <ProgressBarAndroid
                styleAttr="Horizontal"
                indeterminate={false}
                progress={item.progress / 100}
                color="#3b82f6"
              />
            ) : (
              <ProgressViewIOS progress={item.progress / 100} progressTintColor="#3b82f6" />
            )}
            <Text style={styles.progressText}>{item.progress}%</Text>
          </View>
        )}
      </View>
      {item.status === 'failed' && (
        <TouchableOpacity style={styles.retryButton}>
          <Ionicons name="refresh" size={20} color="#3b82f6" />
        </TouchableOpacity>
      )}
    </View>
  );

  const renderEmpty = () => (
    <View style={styles.emptyContainer}>
      <Ionicons name="cloud-upload" size={64} color="#334155" />
      <Text style={styles.emptyText}>No active uploads</Text>
    </View>
  );

  return (
    <SafeAreaView style={styles.container} edges={['bottom']}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Uploads</Text>
      </View>

      {/* Upload Options */}
      <View style={styles.optionsContainer}>
        <TouchableOpacity style={styles.optionButton} onPress={pickDocument}>
          <Ionicons name="document" size={24} color="#3b82f6" />
          <Text style={styles.optionText}>Files</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.optionButton} onPress={pickImage}>
          <Ionicons name="images" size={24} color="#10b981" />
          <Text style={styles.optionText}>Photos</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.optionButton} onPress={takePhoto}>
          <Ionicons name="camera" size={24} color="#f59e0b" />
          <Text style={styles.optionText}>Camera</Text>
        </TouchableOpacity>
      </View>

      {/* Upload List */}
      <FlatList
        data={mockUploads}
        renderItem={renderItem}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.listContent}
        ListEmptyComponent={renderEmpty}
      />
    </SafeAreaView>
  );
}

function getStatusIcon(status: UploadTask['status']): keyof typeof Ionicons.glyphMap {
  switch (status) {
    case 'completed':
      return 'checkmark-circle';
    case 'failed':
      return 'close-circle';
    case 'uploading':
      return 'cloud-upload';
    default:
      return 'time';
  }
}

function getStatusColor(status: UploadTask['status']): string {
  switch (status) {
    case 'completed':
      return '#10b981';
    case 'failed':
      return '#ef4444';
    case 'uploading':
      return '#3b82f6';
    default:
      return '#64748b';
  }
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
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
  optionsContainer: {
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingBottom: 16,
    gap: 12,
  },
  optionButton: {
    flex: 1,
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#334155',
  },
  optionText: {
    color: '#f8fafc',
    marginTop: 8,
    fontSize: 12,
  },
  listContent: {
    padding: 16,
    flexGrow: 1,
  },
  uploadItem: {
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
  uploadInfo: {
    flex: 1,
  },
  fileName: {
    color: '#f8fafc',
    fontSize: 16,
    fontWeight: '500',
    marginBottom: 4,
  },
  uploadMeta: {
    color: '#64748b',
    fontSize: 12,
    marginBottom: 4,
  },
  progressContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  progressText: {
    color: '#3b82f6',
    fontSize: 12,
    marginLeft: 8,
    minWidth: 35,
  },
  retryButton: {
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
});
