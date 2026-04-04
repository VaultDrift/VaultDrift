import { useState, useCallback, useRef, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import { useDropzone } from 'react-dropzone';
import {
  Folder,
  File,
  Download,
  Upload,
  FolderPlus,
  Search,
  Grid,
  List,
  Loader2,
  ChevronRight,
  Trash2,
  Pencil,
  MoreHorizontal,
  Share,
} from 'lucide-react';
import { toast } from 'sonner';
import { filesApi, uploadApi, sharesApi, previewApi, getAuthBlobUrl } from '@/lib/api';
import { File as FileType, ViewMode, Breadcrumb, Share as ShareType } from '@/types';
import { FilePreview } from '@/components/FilePreview';
import { cn, formatBytes, formatDate } from '@/lib/utils';

export function FilesPage() {
  const { folderId } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [viewMode, setViewMode] = useState<ViewMode>(
    () => (localStorage.getItem('viewMode') as ViewMode) || 'list'
  );
  const [searchQuery, setSearchQuery] = useState('');
  const [uploadProgress, setUploadProgress] = useState<{
    active: boolean;
    fileName: string;
    fileIndex: number;
    totalFiles: number;
    percent: number;
  } | null>(null);
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [breadcrumbs, setBreadcrumbs] = useState<Breadcrumb[]>([]);
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [previewFile, setPreviewFile] = useState<FileType | null>(null);
  const [sharingFile, setSharingFile] = useState<FileType | null>(null);
  const [shareForm, setShareForm] = useState({
    share_type: 'link' as 'link' | 'user',
    password: '',
    expires_days: '',
    permission: 'read' as 'read' | 'write',
  });

  // Pagination state
  const PAGE_SIZE = 50;
  const [pageOffset, setPageOffset] = useState(0);

  // Upload abort controller
  const abortControllerRef = useRef<AbortController | null>(null);

  // Fetch files
  const { data: files, isLoading } = useQuery({
    queryKey: ['files', folderId, pageOffset],
    queryFn: () => filesApi.list(folderId, { limit: PAGE_SIZE, offset: pageOffset }),
  });

  // Server-side search
  const { data: searchResults } = useQuery({
    queryKey: ['files', 'search', searchQuery],
    queryFn: () => filesApi.search(searchQuery),
    enabled: searchQuery.length >= 3,
  });

  // Fetch breadcrumbs from server
  const { data: serverBreadcrumbs } = useQuery({
    queryKey: ['breadcrumbs', folderId],
    queryFn: () => filesApi.getBreadcrumbs(folderId!),
    enabled: !!folderId,
  });

  // Sync server breadcrumbs to state
  useEffect(() => {
    if (!folderId) {
      setBreadcrumbs([]);
    } else if (serverBreadcrumbs) {
      setBreadcrumbs(serverBreadcrumbs);
    }
  }, [folderId, serverBreadcrumbs]);

  // Reset pagination when navigating between folders
  useEffect(() => {
    setPageOffset(0);
  }, [folderId]);

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: filesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files', folderId] });
      toast.success('File deleted');
    },
    onError: () => toast.error('Failed to delete file'),
  });

  // Rename mutation
  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      filesApi.update(id, { name } as Partial<FileType>),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files', folderId] });
      toast.success('Renamed successfully');
      setRenamingId(null);
    },
    onError: () => toast.error('Failed to rename'),
  });

  // Create folder mutation
  const createFolderMutation = useMutation({
    mutationFn: (name: string) =>
      filesApi.create({
        name,
        type: 'folder',
        parent_id: folderId,
      } as Partial<FileType>),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files', folderId] });
      toast.success('Folder created');
      setShowNewFolder(false);
      setNewFolderName('');
    },
    onError: () => toast.error('Failed to create folder'),
  });

  // Share creation mutation
  const createShareMutation = useMutation({
    mutationFn: ({ fileId, options }: { fileId: string; options: Partial<ShareType> }) =>
      sharesApi.create(fileId, options),
    onSuccess: (share) => {
      toast.success(share.token
        ? `Share created: ${window.location.origin}/s/${share.token}`
        : 'Share created');
      setSharingFile(null);
      setShareForm({ share_type: 'link', password: '', expires_days: '', permission: 'read' });
    },
    onError: () => toast.error('Failed to create share'),
  });

  // Upload handling
  const onDrop = useCallback(
    async (acceptedFiles: globalThis.File[]) => {
      const controller = new AbortController();
      abortControllerRef.current = controller;
      setUploadProgress({ active: true, fileName: '', fileIndex: 0, totalFiles: acceptedFiles.length, percent: 0 });
      try {
        for (let fi = 0; fi < acceptedFiles.length; fi++) {
          const file = acceptedFiles[fi];
          setUploadProgress({ active: true, fileName: file.name, fileIndex: fi + 1, totalFiles: acceptedFiles.length, percent: 0 });

          const init = await uploadApi.init(file.name, file.size, folderId, controller.signal);
          const chunkSize = init.chunk_size || 4 * 1024 * 1024; // 4MB default
          const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));

          for (let i = 0; i < totalChunks; i++) {
            const start = i * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const blob = file.slice(start, end);
            await uploadApi.uploadChunk(init.upload_id, i, blob, undefined, controller.signal);
            setUploadProgress(prev => prev ? { ...prev, percent: Math.round(((i + 1) / totalChunks) * 100) } : prev);
          }
          await uploadApi.complete(init.upload_id, controller.signal);
        }
        queryClient.invalidateQueries({ queryKey: ['files', folderId] });
        toast.success('Upload complete');
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          toast.info('Upload cancelled');
        } else {
          toast.error('Upload failed');
        }
      } finally {
        abortControllerRef.current = null;
        setUploadProgress(null);
      }
    },
    [folderId, queryClient]
  );

  // Cancel upload handler
  const handleCancelUpload = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    noClick: true,
  });

  // Handle folder navigation
  const handleNavigate = (file: FileType) => {
    if (file.type === 'folder') {
      navigate(`/files/${file.id}`);
    }
  };

  // Handle breadcrumb click
  const handleBreadcrumbClick = (index: number) => {
    if (index === -1) {
      setBreadcrumbs([]);
      navigate('/files');
    } else {
      const crumb = breadcrumbs[index];
      setBreadcrumbs((prev) => prev.slice(0, index + 1));
      navigate(`/files/${crumb.id}`);
    }
  };

  // Start rename
  const startRename = (file: FileType) => {
    setRenamingId(file.id);
    setRenameValue(file.name);
  };

  // Confirm rename
  const confirmRename = () => {
    if (renamingId && renameValue.trim()) {
      renameMutation.mutate({ id: renamingId, name: renameValue.trim() });
    } else {
      setRenamingId(null);
    }
  };

  // Download file
  const handleDownload = (file: FileType) => {
    filesApi.download(file.id).catch(() => toast.error('Download failed'));
  };

  // Display files: use server-side search when query has 3+ chars, otherwise show all
  const displayFiles = searchQuery.length >= 3 ? searchResults : files;

  return (
    <div
      {...getRootProps()}
      className={cn('h-full flex flex-col', isDragActive && 'bg-primary/5')}
    >
      <input {...getInputProps()} />

      {/* Toolbar */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card shrink-0">
        <div className="flex items-center gap-4 flex-1 min-w-0">
          {/* Breadcrumbs */}
          <nav className="flex items-center gap-1 text-sm min-w-0">
            <button
              onClick={() => handleBreadcrumbClick(-1)}
              className={cn(
                'hover:text-foreground transition-colors shrink-0',
                !folderId ? 'text-foreground font-semibold' : 'text-muted-foreground'
              )}
            >
              My Files
            </button>
            {breadcrumbs.map((crumb, index) => (
              <span key={crumb.id} className="flex items-center gap-1 shrink-0">
                <ChevronRight className="w-3 h-3 text-muted-foreground" />
                <button
                  onClick={() => handleBreadcrumbClick(index)}
                  className={cn(
                    'hover:text-foreground transition-colors',
                    index === breadcrumbs.length - 1
                      ? 'text-foreground font-semibold'
                      : 'text-muted-foreground'
                  )}
                >
                  {crumb.name}
                </button>
              </span>
            ))}
          </nav>

          <div className="relative max-w-md flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {/* View toggle */}
          <div className="flex items-center bg-muted rounded-lg p-1">
            <button
              onClick={() => { setViewMode('grid'); localStorage.setItem('viewMode', 'grid'); }}
              className={cn(
                'p-2 rounded-md transition-colors',
                viewMode === 'grid' ? 'bg-background shadow-sm' : 'hover:bg-background/50'
              )}
            >
              <Grid className="w-4 h-4" />
            </button>
            <button
              onClick={() => { setViewMode('list'); localStorage.setItem('viewMode', 'list'); }}
              className={cn(
                'p-2 rounded-md transition-colors',
                viewMode === 'list' ? 'bg-background shadow-sm' : 'hover:bg-background/50'
              )}
            >
              <List className="w-4 h-4" />
            </button>
          </div>

          <button
            onClick={() => setShowNewFolder(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border hover:bg-accent transition-colors"
          >
            <FolderPlus className="w-4 h-4" />
            <span>New Folder</span>
          </button>

          <label className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors cursor-pointer">
            <Upload className="w-4 h-4" />
            <span>Upload</span>
            <input
              type="file"
              multiple
              className="hidden"
              onChange={(e) => e.target.files && onDrop(Array.from(e.target.files))}
            />
          </label>
        </div>
      </header>

      {/* New Folder Dialog */}
      {showNewFolder && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border p-6 w-full max-w-sm shadow-xl">
            <h2 className="text-lg font-semibold mb-4">Create New Folder</h2>
            <input
              type="text"
              placeholder="Folder name"
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && newFolderName.trim()) {
                  createFolderMutation.mutate(newFolderName.trim());
                }
              }}
              className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
              autoFocus
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => {
                  setShowNewFolder(false);
                  setNewFolderName('');
                }}
                className="px-4 py-2 rounded-lg border hover:bg-accent transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => createFolderMutation.mutate(newFolderName.trim())}
                disabled={!newFolderName.trim() || createFolderMutation.isPending}
                className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {createFolderMutation.isPending ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Rename Dialog */}
      {renamingId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border p-6 w-full max-w-sm shadow-xl">
            <h2 className="text-lg font-semibold mb-4">Rename</h2>
            <input
              type="text"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') confirmRename();
                if (e.key === 'Escape') setRenamingId(null);
              }}
              className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
              autoFocus
              onFocus={(e) => {
                // Select name without extension
                const dotIndex = renameValue.lastIndexOf('.');
                if (dotIndex > 0) {
                  e.target.setSelectionRange(0, dotIndex);
                } else {
                  e.target.select();
                }
              }}
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setRenamingId(null)}
                className="px-4 py-2 rounded-lg border hover:bg-accent transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmRename}
                disabled={!renameValue.trim() || renameMutation.isPending}
                className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {renameMutation.isPending ? 'Saving...' : 'Rename'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Share Dialog */}
      {sharingFile && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card rounded-lg border p-6 w-full max-w-sm shadow-xl">
            <h2 className="text-lg font-semibold mb-4">Share &ldquo;{sharingFile.name}&rdquo;</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Share type</label>
                <select
                  value={shareForm.share_type}
                  onChange={(e) => setShareForm({ ...shareForm, share_type: e.target.value as 'link' | 'user' })}
                  className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="link">Link</option>
                  <option value="user">User</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Password (optional)</label>
                <input
                  type="password"
                  placeholder="Optional password"
                  value={shareForm.password}
                  onChange={(e) => setShareForm({ ...shareForm, password: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Expiry days (optional)</label>
                <input
                  type="number"
                  placeholder="Leave empty for no expiry"
                  value={shareForm.expires_days}
                  onChange={(e) => setShareForm({ ...shareForm, expires_days: e.target.value })}
                  className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Permission</label>
                <select
                  value={shareForm.permission}
                  onChange={(e) => setShareForm({ ...shareForm, permission: e.target.value as 'read' | 'write' })}
                  className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="read">Read</option>
                  <option value="write">Write</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => {
                  setSharingFile(null);
                  setShareForm({ share_type: 'link', password: '', expires_days: '', permission: 'read' });
                }}
                className="px-4 py-2 rounded-lg border hover:bg-accent transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  const options: Partial<ShareType> = {
                    share_type: shareForm.share_type,
                    permission: shareForm.permission,
                  };
                  if (shareForm.password) options.password = shareForm.password;
                  if (shareForm.expires_days) options.expires_days = Number(shareForm.expires_days);
                  createShareMutation.mutate({ fileId: sharingFile.id, options });
                }}
                disabled={createShareMutation.isPending}
                className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {createShareMutation.isPending ? 'Creating...' : 'Create Share'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Drop zone overlay */}
      {isDragActive && (
        <div className="absolute inset-0 bg-primary/10 border-2 border-dashed border-primary m-4 rounded-2xl flex items-center justify-center z-50">
          <div className="text-center">
            <Upload className="w-12 h-12 mx-auto mb-4 text-primary" />
            <p className="text-lg font-medium">Drop files here to upload</p>
          </div>
        </div>
      )}

      {/* Upload progress */}
      {uploadProgress && (
        <div className="bg-primary/10 px-6 py-3 flex items-center gap-3">
          <Loader2 className="w-4 h-4 animate-spin shrink-0" />
          <span className="text-sm">
            Uploading {uploadProgress.fileName} ({uploadProgress.fileIndex} of {uploadProgress.totalFiles})
          </span>
          <span className="text-sm font-medium">{uploadProgress.percent}%</span>
          <button
            onClick={handleCancelUpload}
            className="ml-auto px-3 py-1 text-sm rounded-md border border-destructive/50 text-destructive hover:bg-destructive/10 transition-colors"
          >
            Cancel
          </button>
        </div>
      )}

      {/* File list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
          </div>
        ) : displayFiles?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            {searchQuery.length >= 3 ? (
              <>
                <Search className="w-16 h-16 mb-4 opacity-50" />
                <p className="text-lg">No files matching &ldquo;{searchQuery}&rdquo;</p>
              </>
            ) : (
              <>
                <Folder className="w-16 h-16 mb-4 opacity-50" />
                <p className="text-lg">This folder is empty</p>
                <p className="text-sm">Drag and drop files or click Upload</p>
              </>
            )}
          </div>
        ) : (
          <>
            {!searchQuery && files && files.length > 0 && (
              <p className="text-sm text-muted-foreground mb-3">
                Showing {files.length} item{files.length !== 1 ? 's' : ''}
              </p>
            )}
            {viewMode === 'grid' ? (
              <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                {displayFiles?.map((file: FileType) => (
                  <FileGridItem
                    key={file.id}
                    file={file}
                    onDownload={() => handleDownload(file)}
                    onNavigate={() => handleNavigate(file)}
                    onRename={() => startRename(file)}
                    onDelete={() => { if (confirm(`Delete "${file.name}"? It will be moved to trash.`)) deleteMutation.mutate(file.id); }}
                    onPreview={() => setPreviewFile(file)}
                    onShare={() => setSharingFile(file)}
                  />            ))}
              </div>
            ) : (
              <div className="bg-card rounded-lg border">
                <table className="w-full">
                  <thead className="border-b">
                    <tr className="text-left text-sm text-muted-foreground">
                      <th className="px-4 py-3 font-medium">Name</th>
                      <th className="px-4 py-3 font-medium w-32">Size</th>
                      <th className="px-4 py-3 font-medium w-48">Modified</th>
                      <th className="px-4 py-3 font-medium w-24"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {displayFiles?.map((file: FileType) => (
                      <FileListItem
                        key={file.id}
                        file={file}
                        onDownload={() => handleDownload(file)}
                        onNavigate={() => handleNavigate(file)}
                        onRename={() => startRename(file)}
                        onDelete={() => { if (confirm(`Delete "${file.name}"? It will be moved to trash.`)) deleteMutation.mutate(file.id); }}
                        onPreview={() => setPreviewFile(file)}
                        onShare={() => setSharingFile(file)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {!searchQuery && files && files.length === PAGE_SIZE && (
              <div className="flex justify-center mt-6">
                <button
                  onClick={() => setPageOffset(prev => prev + PAGE_SIZE)}
                  className="px-6 py-2 rounded-lg border hover:bg-accent transition-colors text-sm font-medium"
                >
                  Load more
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* File Preview */}
      {previewFile && (
        <FilePreview
          file={previewFile}
          onClose={() => setPreviewFile(null)}
          onRename={() => {
            startRename(previewFile);
            setPreviewFile(null);
          }}
          onDelete={() => {
            if (confirm(`Delete "${previewFile.name}"? It will be moved to trash.`)) {
              deleteMutation.mutate(previewFile.id);
              setPreviewFile(null);
            }
          }}
          onDownload={() => {
            handleDownload(previewFile);
          }}
        />
      )}
    </div>
  );
}

// Action dropdown menu
function ActionMenu({
  isFolder,
  onDownload,
  onRename,
  onDelete,
  onShare,
}: {
  isFolder: boolean;
  onDownload: () => void;
  onRename: () => void;
  onDelete: () => void;
  onShare?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={(e) => {
          e.stopPropagation();
          setOpen(!open);
        }}
        className="p-2 hover:bg-accent rounded-lg transition-colors"
      >
        <MoreHorizontal className="w-4 h-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 w-40 bg-card border rounded-lg shadow-lg z-50 py-1">
          {!isFolder && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDownload();
                setOpen(false);
              }}
              className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
            >
              <Download className="w-4 h-4" />
              Download
            </button>
          )}
          {!isFolder && onShare && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onShare();
                setOpen(false);
              }}
              className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
            >
              <Share className="w-4 h-4" />
              Share
            </button>
          )}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onRename();
              setOpen(false);
            }}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
          >
            <Pencil className="w-4 h-4" />
            Rename
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
              setOpen(false);
            }}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-accent transition-colors"
          >
            <Trash2 className="w-4 h-4" />
            Delete
          </button>
        </div>
      )}
    </div>
  );
}

const IMAGE_MIME_TYPES = new Set([
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
  'image/svg+xml',
]);

// File Grid Item
function FileGridItem({
  file,
  onDownload,
  onNavigate,
  onRename,
  onDelete,
  onPreview,
  onShare,
}: {
  file: FileType;
  onDownload: () => void;
  onNavigate: () => void;
  onRename: () => void;
  onDelete: () => void;
  onPreview: () => void;
  onShare?: () => void;
}) {
  const isFolder = file.type === 'folder';
  const isImage = !isFolder && IMAGE_MIME_TYPES.has(file.mime_type);

  // Thumbnail loading for image files
  const [thumbUrl, setThumbUrl] = useState<string | null>(null);
  const [thumbError, setThumbError] = useState(false);
  const thumbUrlRef = useRef<string | null>(null);

  useEffect(() => {
    if (!isImage) return;

    let cancelled = false;

    getAuthBlobUrl(previewApi.getThumbnailUrl(file.id))
      .then((url) => {
        if (!cancelled) {
          thumbUrlRef.current = url;
          setThumbUrl(url);
        }
      })
      .catch(() => {
        if (!cancelled) setThumbError(true);
      });

    return () => {
      cancelled = true;
      if (thumbUrlRef.current) {
        URL.revokeObjectURL(thumbUrlRef.current);
        thumbUrlRef.current = null;
      }
    };
  }, [file.id, isImage]);

  return (
    <div
      onClick={isFolder ? onNavigate : onPreview}
      className={cn(
        'group relative bg-card border rounded-xl p-4 hover:border-primary transition-colors',
        !isFolder && 'cursor-pointer'
      )}
    >
      <div className="flex flex-col items-center text-center">
        {isFolder ? (
          <Folder className="w-12 h-12 text-primary mb-3" />
        ) : isImage && thumbUrl && !thumbError ? (
          <img
            src={thumbUrl}
            alt={file.name}
            className="w-12 h-12 object-cover rounded mb-3"
            onError={() => {
              setThumbError(true);
              if (thumbUrlRef.current) {
                URL.revokeObjectURL(thumbUrlRef.current);
                thumbUrlRef.current = null;
              }
            }}
          />
        ) : (
          <File className="w-12 h-12 text-muted-foreground mb-3" />
        )}
        <p className="font-medium truncate w-full">{file.name}</p>
        <p className="text-xs text-muted-foreground">
          {isFolder ? 'Folder' : formatBytes(file.size_bytes)}
        </p>
      </div>

      <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-all">
        <ActionMenu
          isFolder={isFolder}
          onDownload={onDownload}
          onRename={onRename}
          onDelete={onDelete}
          onShare={onShare}
        />
      </div>
    </div>
  );
}

// File List Item
function FileListItem({
  file,
  onDownload,
  onNavigate,
  onRename,
  onDelete,
  onPreview,
  onShare,
}: {
  file: FileType;
  onDownload: () => void;
  onNavigate: () => void;
  onRename: () => void;
  onDelete: () => void;
  onPreview: () => void;
  onShare?: () => void;
}) {
  const isFolder = file.type === 'folder';

  return (
    <tr className="group hover:bg-accent/50">
      <td className="px-4 py-3">
        <div
          onClick={isFolder ? onNavigate : onPreview}
          onDoubleClick={!isFolder ? onRename : undefined}
          className={cn('flex items-center gap-3', 'cursor-pointer')}
        >
          {isFolder ? (
            <Folder className="w-5 h-5 text-primary" />
          ) : (
            <File className="w-5 h-5 text-muted-foreground" />
          )}
          <span className="font-medium">{file.name}</span>
        </div>
      </td>
      <td className="px-4 py-3 text-muted-foreground">
        {isFolder ? '--' : formatBytes(file.size_bytes)}
      </td>
      <td className="px-4 py-3 text-muted-foreground">{formatDate(file.updated_at)}</td>
      <td className="px-4 py-3">
        <div className="opacity-0 group-hover:opacity-100 transition-all">
          <ActionMenu
            isFolder={isFolder}
            onDownload={onDownload}
            onRename={onRename}
            onDelete={onDelete}
            onShare={onShare}
          />
        </div>
      </td>
    </tr>
  );
}
