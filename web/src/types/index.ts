// Types for VaultDrift Web UI

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  role: string;
  quota_bytes: number;
  used_bytes: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface File {
  id: string;
  user_id: string;
  parent_id: string | null;
  name: string;
  type: 'file' | 'folder';
  size_bytes: number;
  mime_type: string;
  is_trashed: boolean;
  trashed_at: string | null;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Share {
  id: string;
  file_id: string;
  share_type: 'link' | 'user';
  token?: string;
  expires_at?: string;
  max_downloads?: number;
  download_count: number;
  allow_upload: boolean;
  preview_only: boolean;
  permission: string;
  is_active: boolean;
  created_at: string;
}

export interface UploadProgress {
  id: string;
  file: File;
  progress: number;
  status: 'pending' | 'uploading' | 'completed' | 'error';
  speed: number;
  eta: number;
}

export interface WebSocketEvent {
  type: string;
  user_id?: string;
  file_id?: string;
  folder_id?: string;
  data: unknown;
  timestamp: number;
}

export type ViewMode = 'grid' | 'list';

export interface Breadcrumb {
  id: string;
  name: string;
}
