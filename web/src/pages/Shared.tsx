import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, Copy, Trash2, Lock, Globe, Users, File, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { sharesApi } from '@/lib/api';
import { Share, ReceivedShare } from '@/types';
import { formatDate } from '@/lib/utils';
import { cn } from '@/lib/utils';

type Tab = 'mine' | 'received';

export function SharedPage() {
  const queryClient = useQueryClient();
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('mine');

  const { data: shares, isLoading: sharesLoading } = useQuery<Share[]>({
    queryKey: ['shares'],
    queryFn: () => sharesApi.list(),
  });

  const { data: receivedShares, isLoading: receivedLoading } = useQuery<ReceivedShare[]>({
    queryKey: ['shares', 'received'],
    queryFn: () => sharesApi.getReceived(),
  });

  const deleteMutation = useMutation({
    mutationFn: sharesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] });
      toast.success('Share removed');
    },
    onError: () => toast.error('Failed to remove share'),
  });

  const copyLink = async (share: Share) => {
    const baseUrl = window.location.origin;
    const url = `${baseUrl}/s/${share.token}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopiedId(share.id);
      toast.success('Link copied to clipboard');
    } catch {
      toast.error('Failed to copy link');
    }
    setTimeout(() => setCopiedId(null), 2000);
  };

  const getShareIcon = (share: Share) => {
    if (share.share_type === 'link') {
      return share.is_active ? <Globe className="w-5 h-5" /> : <Lock className="w-5 h-5" />;
    }
    return <Users className="w-5 h-5" />;
  };

  const isLoading = activeTab === 'mine' ? sharesLoading : receivedLoading;

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card">
        <h1 className="text-lg font-semibold">Shared Files</h1>
      </header>

      {/* Tabs */}
      <div className="px-6 pt-4 flex gap-1">
        <button
          onClick={() => setActiveTab('mine')}
          className={cn(
            'px-4 py-2 text-sm font-medium rounded-t-lg transition-colors',
            activeTab === 'mine'
              ? 'bg-card border border-b-0 text-foreground'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          My Shares
        </button>
        <button
          onClick={() => setActiveTab('received')}
          className={cn(
            'px-4 py-2 text-sm font-medium rounded-t-lg transition-colors',
            activeTab === 'received'
              ? 'bg-card border border-b-0 text-foreground'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Shared with Me
        </button>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-auto px-6 pb-6">
        {activeTab === 'mine' ? (
          isLoading ? (
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
                          <span className="font-medium">{share.token || 'Share'}</span>
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
                            onClick={() => { if (confirm('Remove this share? Anyone with the link will lose access.')) deleteMutation.mutate(share.id); }}
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
          )
        ) : (
          // Shared with Me tab
          isLoading ? (
            <div className="flex items-center justify-center h-full">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          ) : receivedShares?.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <Users className="w-16 h-16 mb-4 opacity-50" />
              <p className="text-lg">No files shared with you</p>
              <p className="text-sm">When others share files with you, they will appear here</p>
            </div>
          ) : (
            <div className="bg-card rounded-lg border">
              <table className="w-full">
                <thead className="border-b">
                  <tr className="text-left text-sm text-muted-foreground">
                    <th className="px-4 py-3 font-medium">File</th>
                    <th className="px-4 py-3 font-medium">Shared by</th>
                    <th className="px-4 py-3 font-medium">Permission</th>
                    <th className="px-4 py-3 font-medium">Type</th>
                    <th className="px-4 py-3 font-medium">Shared</th>
                    <th className="px-4 py-3 font-medium">Expires</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {receivedShares?.map((share) => (
                    <tr key={share.id} className="group hover:bg-accent/50">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <File className="w-5 h-5 text-muted-foreground" />
                          <span className="font-medium">{share.file_name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {share.shared_by}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground capitalize">
                        {share.permission}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground capitalize">
                        {share.share_type}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {formatDate(share.created_at)}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {share.expires_at ? formatDate(share.expires_at) : (
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            Never
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
      </div>
    </div>
  );
}
