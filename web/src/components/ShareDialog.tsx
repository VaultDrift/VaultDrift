import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { X, Link, Copy, Clock, Eye, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { sharesApi } from '@/lib/api';
import { cn, formatDate } from '@/lib/utils';

interface ShareDialogProps {
  fileId: string;
  fileName: string;
  isOpen: boolean;
  onClose: () => void;
}

export function ShareDialog({ fileId, fileName, isOpen, onClose }: ShareDialogProps) {
  const [password, setPassword] = useState('');
  const [expiresIn, setExpiresIn] = useState('7');
  const [previewOnly, setPreviewOnly] = useState(false);
  const [allowUpload, setAllowUpload] = useState(false);
  const queryClient = useQueryClient();

  // Fetch existing shares
  const { data: shares } = useQuery({
    queryKey: ['shares', fileId],
    queryFn: () => sharesApi.list(fileId),
    enabled: isOpen,
  });

  // Create share mutation
  const createMutation = useMutation({
    mutationFn: () =>
      sharesApi.create(fileId, {
        password: password || undefined,
        expires_in_days: parseInt(expiresIn),
        preview_only: previewOnly,
        allow_upload: allowUpload,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares', fileId] });
      toast.success('Share link created');
      setPassword('');
    },
    onError: () => toast.error('Failed to create share'),
  });

  // Delete share mutation
  const deleteMutation = useMutation({
    mutationFn: sharesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares', fileId] });
      toast.success('Share revoked');
    },
    onError: () => toast.error('Failed to revoke share'),
  });

  const copyLink = (token: string) => {
    const url = `${window.location.origin}/s/${token}`;
    navigator.clipboard.writeText(url);
    toast.success('Link copied to clipboard');
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card border rounded-xl shadow-lg w-full max-w-lg max-h-[90vh] overflow-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div>
            <h2 className="text-xl font-semibold">Share</h2>
            <p className="text-sm text-muted-foreground truncate max-w-[300px]">
              {fileName}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-accent rounded-lg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* New Share Form */}
        <div className="p-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium mb-2 block">
                Expires in
              </label>
              <select
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border bg-background"
              >
                <option value="1">1 day</option>
                <option value="7">7 days</option>
                <option value="30">30 days</option>
                <option value="90">90 days</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">
                Password (optional)
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="No password"
                className="w-full px-3 py-2 rounded-lg border bg-background"
              />
            </div>
          </div>

          <div className="flex gap-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={previewOnly}
                onChange={(e) => setPreviewOnly(e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">Preview only</span>
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={allowUpload}
                onChange={(e) => setAllowUpload(e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">Allow upload</span>
            </label>
          </div>

          <button
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
            className={cn(
              'w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
              createMutation.isPending && 'opacity-50 cursor-not-allowed'
            )}
          >
            <Link className="w-4 h-4" />
            {createMutation.isPending ? 'Creating...' : 'Create Share Link'}
          </button>
        </div>

        {/* Existing Shares */}
        {shares && shares.length > 0 && (
          <div className="border-t">
            <div className="p-4 bg-muted/50">
              <h3 className="font-medium">Active Shares</h3>
            </div>
            <div className="divide-y">
              {shares.map((share) => (
                <div
                  key={share.id}
                  className="p-4 flex items-center justify-between hover:bg-accent/50"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex items-center gap-1 text-sm text-muted-foreground">
                      <Clock className="w-4 h-4" />
                      {share.expires_at ? formatDate(share.expires_at) : 'No expiry'}
                    </div>
                    {share.has_password && (
                      <span className="text-xs bg-amber-100 text-amber-800 px-2 py-1 rounded">
                        Password
                      </span>
                    )}
                    {share.preview_only && (
                      <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded flex items-center gap-1">
                        <Eye className="w-3 h-3" />
                        Preview
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-1">
                    {share.token && (
                      <button
                        onClick={() => copyLink(share.token!)}
                        className="p-2 hover:bg-accent rounded-lg transition-colors"
                        title="Copy link"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      onClick={() => deleteMutation.mutate(share.id)}
                      disabled={deleteMutation.isPending}
                      className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                      title="Revoke"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
