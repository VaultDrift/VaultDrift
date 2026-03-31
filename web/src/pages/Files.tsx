import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { useDropzone } from 'react-dropzone';
import {
  Folder,
  File,
  MoreVertical,
  Upload,
  FolderPlus,
  Search,
  Grid,
  List,
  Loader2,
} from 'lucide-react';
import { toast } from 'sonner';
import { filesApi, uploadApi } from '@/lib/api';
import { File as FileType, ViewMode } from '@/types';
import { cn, formatBytes, formatDate } from '@/lib/utils';

export function FilesPage() {
  const { folderId } = useParams();
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [searchQuery, setSearchQuery] = useState('');
  const [isUploading, setIsUploading] = useState(false);
  const queryClient = useQueryClient();

  // Fetch files
  const { data: files, isLoading } = useQuery({
    queryKey: ['files', folderId],
    queryFn: () => filesApi.list(folderId),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: filesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files', folderId] });
      toast.success('File deleted');
    },
    onError: () => toast.error('Failed to delete file'),
  });

  // Upload handling
  const onDrop = useCallback(async (acceptedFiles: globalThis.File[]) => {
    setIsUploading(true);
    try {
      for (const file of acceptedFiles) {
        // Initialize upload
        const init = await uploadApi.init(file.name, file.size, folderId);

        // Upload in chunks (simplified - 1 chunk for small files)
        await uploadApi.uploadChunk(init.upload_id, 0, file);

        // Complete upload
        await uploadApi.complete(init.upload_id);
      }
      queryClient.invalidateQueries({ queryKey: ['files', folderId] });
      toast.success('Upload complete');
    } catch (error) {
      toast.error('Upload failed');
    } finally {
      setIsUploading(false);
    }
  }, [folderId, queryClient]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    noClick: true,
  });

  // Filter files by search
  const filteredFiles = files?.filter((file) =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div
      {...getRootProps()}
      className={cn(
        'h-full flex flex-col',
        isDragActive && 'bg-primary/5'
      )}
    >
      <input {...getInputProps()} />

      {/* Toolbar */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card">
        <div className="flex items-center gap-4 flex-1">
          <h1 className="text-lg font-semibold">
            {folderId ? 'Folder' : 'My Files'}
          </h1>

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

        <div className="flex items-center gap-2">
          {/* View toggle */}
          <div className="flex items-center bg-muted rounded-lg p-1">
            <button
              onClick={() => setViewMode('grid')}
              className={cn(
                'p-2 rounded-md transition-colors',
                viewMode === 'grid' ? 'bg-background shadow-sm' : 'hover:bg-background/50'
              )}
            >
              <Grid className="w-4 h-4" />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={cn(
                'p-2 rounded-md transition-colors',
                viewMode === 'list' ? 'bg-background shadow-sm' : 'hover:bg-background/50'
              )}
            >
              <List className="w-4 h-4" />
            </button>
          </div>

          <button
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
      {isUploading && (
        <div className="bg-primary/10 px-6 py-3 flex items-center gap-3">
          <Loader2 className="w-4 h-4 animate-spin" />
          <span className="text-sm">Uploading files...</span>
        </div>
      )}

      {/* File list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
          </div>
        ) : filteredFiles?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Folder className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg">This folder is empty</p>
            <p className="text-sm">Drag and drop files or click Upload</p>
          </div>
        ) : viewMode === 'grid' ? (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {filteredFiles?.map((file) => (
              <FileGridItem
                key={file.id}
                file={file}
                onDelete={() => deleteMutation.mutate(file.id)}
              />
            ))}
          </div>
        ) : (
          <div className="bg-card rounded-lg border">
            <table className="w-full">
              <thead className="border-b">
                <tr className="text-left text-sm text-muted-foreground">
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium w-32">Size</th>
                  <th className="px-4 py-3 font-medium w-48">Modified</th>
                  <th className="px-4 py-3 font-medium w-16"></th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {filteredFiles?.map((file) => (
                  <FileListItem
                    key={file.id}
                    file={file}
                    onDelete={() => deleteMutation.mutate(file.id)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

// File Grid Item
function FileGridItem({ file, onDelete }: { file: FileType; onDelete: () => void }) {
  const isFolder = file.type === 'folder';

  return (
    <div className="group relative bg-card border rounded-xl p-4 hover:border-primary transition-colors">
      <div className="flex flex-col items-center text-center">
        {isFolder ? (
          <Folder className="w-12 h-12 text-primary mb-3" />
        ) : (
          <File className="w-12 h-12 text-muted-foreground mb-3" />
        )}
        <p className="font-medium truncate w-full">{file.name}</p>
        <p className="text-xs text-muted-foreground">
          {isFolder ? 'Folder' : formatBytes(file.size_bytes)}
        </p>
      </div>

      <button
        onClick={onDelete}
        className="absolute top-2 right-2 p-2 opacity-0 group-hover:opacity-100 hover:bg-accent rounded-lg transition-all"
      >
        <MoreVertical className="w-4 h-4" />
      </button>
    </div>
  );
}

// File List Item
function FileListItem({ file, onDelete }: { file: FileType; onDelete: () => void }) {
  const isFolder = file.type === 'folder';

  return (
    <tr className="group hover:bg-accent/50">
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
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
      <td className="px-4 py-3 text-muted-foreground">
        {formatDate(file.updated_at)}
      </td>
      <td className="px-4 py-3">
        <button
          onClick={onDelete}
          className="p-2 opacity-0 group-hover:opacity-100 hover:bg-accent rounded-lg transition-all"
        >
          <MoreVertical className="w-4 h-4" />
        </button>
      </td>
    </tr>
  );
}
