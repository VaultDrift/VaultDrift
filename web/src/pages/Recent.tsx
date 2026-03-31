import { useQuery } from '@tanstack/react-query';
import { Clock, File, Folder, ArrowRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { filesApi } from '@/lib/api';
import { File as FileType } from '@/types';
import { formatBytes, formatDate } from '@/lib/utils';

export function RecentPage() {
  const navigate = useNavigate();

  const { data: recentFiles, isLoading } = useQuery({
    queryKey: ['recent'],
    queryFn: filesApi.recent,
  });

  const handleFileClick = (file: FileType) => {
    if (file.type === 'folder') {
      navigate(`/files/${file.id}`);
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card">
        <h1 className="text-lg font-semibold">Recent Files</h1>
      </header>

      {/* Recent files list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        ) : recentFiles?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Clock className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg">No recent files</p>
            <p className="text-sm">Files you access will appear here</p>
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
                {recentFiles?.map((file) => (
                  <tr
                    key={file.id}
                    className="group hover:bg-accent/50 cursor-pointer"
                    onClick={() => handleFileClick(file)}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        {file.type === 'folder' ? (
                          <Folder className="w-5 h-5 text-primary" />
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
                      {formatDate(file.updated_at)}
                    </td>
                    <td className="px-4 py-3">
                      <ArrowRight className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" />
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
