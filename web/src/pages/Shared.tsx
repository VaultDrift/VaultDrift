import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, Copy, Trash2, Lock, Globe, Users } from 'lucide-react';
import { toast } from 'sonner';
import { sharesApi } from '@/lib/api';
import { Share } from '@/types';
import { formatDate } from '@/lib/utils';

export function SharedPage() {
  const queryClient = useQueryClient();
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const { data: shares, isLoading } = useQuery<Share[]>({
    queryKey: ['shares'],
    queryFn: () => sharesApi.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: sharesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] });
      toast.success('Share removed');
    },
    onError: () => toast.error('Failed to remove share'),
  });

  const copyLink = (share: Share) => {
    const baseUrl = window.location.origin;
    const url = `${baseUrl}/s/${share.token}`;
    navigator.clipboard.writeText(url);
    setCopiedId(share.id);
    toast.success('Link copied to clipboard');
    setTimeout(() => setCopiedId(null), 2000);
  };

  const getShareIcon = (share: Share) => {
    if (share.share_type === 'link') {
      return share.is_active ? <Globe className="w-5 h-5" /> : <Lock className="w-5 h-5" />;
    }
    return <Users className="w-5 h-5" />;
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card">
        <h1 className="text-lg font-semibold">Shared Files</h1>
      </header>

      {/* Share list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        ) : shares?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Link className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg">No shared files</p>
            <p className="text-sm">Share files from your file manager</p>
          </div>
        ) : (
          <div className="bg-card rounded-lg border">
            <table className="w-full">
              <thead className="border-b">
                <tr className="text-left text-sm text-muted-foreground">
                  <th className="px-4 py-3 font-medium">File</th>
                  <th className="px-4 py-3 font-medium">Type</th>
                  <th className="px-4 py-3 font-medium">Permissions</th>
                  <th className="px-4 py-3 font-medium">Downloads</th>
                  <th className="px-4 py-3 font-medium">Created</th>
                  <th className="px-4 py-3 font-medium w-32">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {shares?.map((share) => (
                  <tr key={share.id} className="group hover:bg-accent/50">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        {getShareIcon(share)}
                        <span className="font-medium">{share.file_id}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground capitalize">
                      {share.share_type}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {share.permission}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {share.download_count}
                      {share.max_downloads && ` / ${share.max_downloads}`}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {formatDate(share.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        {share.share_type === 'link' && share.token && (
                          <button
                            onClick={() => copyLink(share)}
                            className="p-2 hover:bg-accent rounded-lg transition-colors"
                            title="Copy link"
                          >
                            {copiedId === share.id ? (
                              <Copy className="w-4 h-4 text-green-500" />
                            ) : (
                              <Copy className="w-4 h-4" />
                            )}
                          </button>
                        )}
                        <button
                          onClick={() => deleteMutation.mutate(share.id)}
                          className="p-2 hover:bg-accent rounded-lg transition-colors text-destructive"
                          title="Remove share"
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
        )}
      </div>
    </div>
  );
}
