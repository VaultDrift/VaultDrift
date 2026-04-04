import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Trash2, RotateCcw, Folder, File, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { trashApi } from '@/lib/api';
import { File as FileType } from '@/types';
import { formatBytes, formatDate } from '@/lib/utils';

const PAGE_SIZE = 50;

export function TrashPage() {
  const queryClient = useQueryClient();
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [pageOffset, setPageOffset] = useState(0);

  const { data: files, isLoading } = useQuery({
    queryKey: ['trash', pageOffset],
    queryFn: () => trashApi.list({ limit: PAGE_SIZE, offset: pageOffset }),
  });

  const restoreMutation = useMutation({
    mutationFn: trashApi.restore,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['trash'] });
      queryClient.invalidateQueries({ queryKey: ['files'] });
      toast.success('File restored');
    },
    onError: () => toast.error('Failed to restore file'),
  });

  const deleteMutation = useMutation({
    mutationFn: trashApi.permanentDelete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['trash'] });
      toast.success('File permanently deleted');
    },
    onError: () => toast.error('Failed to delete file'),
  });

  const emptyTrashMutation = useMutation({
    mutationFn: trashApi.emptyTrash,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['trash'] });
      toast.success('Trash emptied');
    },
    onError: () => toast.error('Failed to empty trash'),
  });

  const toggleSelection = (id: string) => {
    const newSet = new Set(selectedFiles);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedFiles(newSet);
  };

  const restoreSelected = () => {
    selectedFiles.forEach((id) => restoreMutation.mutate(id));
    setSelectedFiles(new Set());
  };

  const deleteSelected = () => {
    if (confirm('Permanently delete selected files? This cannot be undone.')) {
      selectedFiles.forEach((id) => deleteMutation.mutate(id));
      setSelectedFiles(new Set());
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card">
        <h1 className="text-lg font-semibold">Trash</h1>
        {selectedFiles.size > 0 && (
          <div className="flex items-center gap-2">
            <button
              onClick={restoreSelected}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border hover:bg-accent transition-colors"
            >
              <RotateCcw className="w-4 h-4" />
              <span>Restore ({selectedFiles.size})</span>
            </button>
            <button
              onClick={deleteSelected}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              <span>Delete ({selectedFiles.size})</span>
            </button>
          </div>
        )}
      </header>

      {/* Info banner */}
      <div className="bg-muted px-6 py-3 flex items-center gap-3 text-sm text-muted-foreground">
        <AlertCircle className="w-4 h-4" />
        <span>Files in trash are automatically deleted after 30 days</span>
        <div className="flex-1" />
        <button
          onClick={() => { if (confirm('Permanently delete all trashed files?')) emptyTrashMutation.mutate(); }}
          disabled={emptyTrashMutation.isPending}
          className="px-4 py-2 rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
        >
          {emptyTrashMutation.isPending ? 'Emptying...' : 'Empty Trash'}
        </button>
      </div>

      {/* Trash list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        ) : files?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Trash2 className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg">Trash is empty</p>
            <p className="text-sm">Deleted files will appear here</p>
          </div>
        ) : (
          <>
            {files && files.length > 0 && (
              <p className="text-sm text-muted-foreground mb-3">
                Showing {files.length} item{files.length !== 1 ? 's' : ''}
              </p>
            )}
            <div className="bg-card rounded-lg border">
              <table className="w-full">
                <thead className="border-b">
                  <tr className="text-left text-sm text-muted-foreground">
                    <th className="px-4 py-3 font-medium w-12">
                      <input
                        type="checkbox"
                        checked={!!files && files.length > 0 && selectedFiles.size === files.length}
                        onChange={(e) => {
                          if (e.target.checked && files) {
                            setSelectedFiles(new Set(files.map((f: FileType) => f.id)));
                          } else {
                            setSelectedFiles(new Set());
                          }
                        }}
                        className="rounded border-input"
                      />
                    </th>
                    <th className="px-4 py-3 font-medium">Name</th>
                    <th className="px-4 py-3 font-medium w-32">Size</th>
                    <th className="px-4 py-3 font-medium w-48">Deleted</th>
                    <th className="px-4 py-3 font-medium w-32">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {files?.map((file) => (
                    <tr key={file.id} className="group hover:bg-accent/50">
                      <td className="px-4 py-3">
                        <input
                          type="checkbox"
                          checked={selectedFiles.has(file.id)}
                          onChange={() => toggleSelection(file.id)}
                          className="rounded border-input"
                        />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          {file.type === 'folder' ? (
                            <Folder className="w-5 h-5 text-muted-foreground" />
                          ) : (
                            <File className="w-5 h-5 text-muted-foreground" />
                          )}
                          <span className="font-medium">{file.name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {file.type === 'folder' ? '--' : formatBytes(file.size_bytes)}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {file.trashed_at ? formatDate(file.trashed_at) : '--'}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            onClick={() => restoreMutation.mutate(file.id)}
                            className="p-2 hover:bg-accent rounded-lg transition-colors"
                            title="Restore"
                          >
                            <RotateCcw className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => {
                              if (confirm('Permanently delete this file?')) {
                                deleteMutation.mutate(file.id);
                              }
                            }}
                            className="p-2 hover:bg-accent rounded-lg transition-colors text-destructive"
                            title="Delete permanently"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {files && files.length === PAGE_SIZE && (
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
    </div>
  );
}
