import { useState, useEffect, useRef } from 'react';
import {
  X,
  Download,
  Trash2,
  Pencil,
  FileText,
  Image as ImageIcon,
  Film,
  Music,
  File,
  Loader2,
} from 'lucide-react';
import { File as FileType } from '@/types';
import api, { previewApi, getAuthBlobUrl } from '@/lib/api';
import { formatBytes, formatDate } from '@/lib/utils';

interface FilePreviewProps {
  file: FileType;
  onClose: () => void;
  onRename: () => void;
  onDelete: () => void;
  onDownload: () => void;
}

function getFileCategory(mime: string): 'image' | 'video' | 'audio' | 'pdf' | 'text' | 'other' {
  if (mime.startsWith('image/')) return 'image';
  if (mime.startsWith('video/')) return 'video';
  if (mime.startsWith('audio/')) return 'audio';
  if (mime === 'application/pdf') return 'pdf';
  if (mime.startsWith('text/') || isCodeMime(mime)) return 'text';
  return 'other';
}

function isCodeMime(mime: string): boolean {
  return (
    mime.includes('json') ||
    mime.includes('javascript') ||
    mime.includes('typescript') ||
    mime.includes('xml') ||
    mime.includes('yaml') ||
    mime.includes('markdown') ||
    mime.includes('svg')
  );
}

function getFileIcon(category: string) {
  switch (category) {
    case 'image':
      return ImageIcon;
    case 'video':
      return Film;
    case 'audio':
      return Music;
    case 'pdf':
    case 'text':
      return FileText;
    default:
      return File;
  }
}

export function FilePreview({ file, onClose, onRename, onDelete, onDownload }: FilePreviewProps) {
  const category = getFileCategory(file.mime_type || '');
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const blobUrlRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    // Sync ref with state so cleanup can revoke the latest URL
    const syncRef = (url: string | null) => {
      blobUrlRef.current = url;
      setBlobUrl(url);
    };

    if (category === 'image' || category === 'pdf') {
      setLoading(true);
      getAuthBlobUrl(previewApi.getStreamUrl(file.id))
        .then((url) => {
          if (!cancelled) syncRef(url);
        })
        .catch(() => { if (!cancelled) setError('Failed to load preview'); })
        .finally(() => { if (!cancelled) setLoading(false); });
    } else if (category === 'video' || category === 'audio') {
      // NOTE: The backend supports HLS streaming (GET /api/v1/media/:id/playlist.m3u8)
      // via previewApi.getHlsPlaylistUrl(), which is more efficient for large video files.
      // However, HLS requires auth headers that <video> / <audio> elements cannot send.
      // To use HLS, we would need either:
      //   1. A token-passing mechanism (e.g., query param token or cookie-based auth)
      //   2. hls.js library for browsers without native HLS support
      // For now, we download the full blob which works correctly but is suboptimal for large files.
      setLoading(true);
      getAuthBlobUrl(previewApi.getStreamUrl(file.id))
        .then((url) => {
          if (!cancelled) syncRef(url);
        })
        .catch(() => { if (!cancelled) setError('Failed to load media'); })
        .finally(() => { if (!cancelled) setLoading(false); });
    } else if (category === 'text') {
      setLoading(true);
      (async () => {
        try {
          const response = await api.get(`/files/${file.id}/stream`, {
            responseType: 'text',
          });
          if (!cancelled) setTextContent(String(response.data).slice(0, 100000));
        } catch {
          if (!cancelled) setError('Failed to load text content');
        } finally {
          if (!cancelled) setLoading(false);
        }
      })();
    }

    return () => {
      cancelled = true;
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current);
        blobUrlRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [file.id, category]);

  const Icon = getFileIcon(category);

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card rounded-xl border shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            <Icon className="w-5 h-5 text-primary shrink-0" />
            <h2 className="text-lg font-semibold truncate">{file.name}</h2>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={onDownload}
              className="p-2 hover:bg-accent rounded-lg transition-colors"
              title="Download"
            >
              <Download className="w-4 h-4" />
            </button>
            <button
              onClick={onRename}
              className="p-2 hover:bg-accent rounded-lg transition-colors"
              title="Rename"
            >
              <Pencil className="w-4 h-4" />
            </button>
            <button
              onClick={onDelete}
              className="p-2 hover:bg-accent rounded-lg transition-colors text-destructive"
              title="Delete"
            >
              <Trash2 className="w-4 h-4" />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-accent rounded-lg transition-colors ml-2"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Preview content */}
        <div className="flex-1 overflow-auto p-6 flex items-center justify-center min-h-[300px]">
          {loading ? (
            <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
          ) : error ? (
            <div className="text-center text-muted-foreground">
              <File className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>{error}</p>
            </div>
          ) : category === 'image' && blobUrl ? (
            <img
              src={blobUrl}
              alt={file.name}
              className="max-w-full max-h-[70vh] object-contain rounded-lg"
            />
          ) : category === 'video' && blobUrl ? (
            <video
              src={blobUrl}
              controls
              className="max-w-full max-h-[70vh] rounded-lg"
            >
              Your browser does not support video playback.
            </video>
          ) : category === 'audio' && blobUrl ? (
            <div className="w-full max-w-md text-center">
              <Music className="w-16 h-16 mx-auto mb-4 text-primary" />
              <p className="font-medium mb-4">{file.name}</p>
              <audio
                src={blobUrl}
                controls
                className="w-full"
              >
                Your browser does not support audio playback.
              </audio>
            </div>
          ) : category === 'pdf' && blobUrl ? (
            <iframe
              src={blobUrl}
              className="w-full h-[70vh] rounded-lg border"
              title={file.name}
            />
          ) : category === 'text' && textContent !== null ? (
            <pre className="w-full bg-muted rounded-lg p-4 text-sm font-mono overflow-auto max-h-[70vh] whitespace-pre-wrap break-words">
              <code>{textContent}</code>
            </pre>
          ) : (
            <div className="text-center text-muted-foreground">
              <Icon className="w-16 h-16 mx-auto mb-4 opacity-50" />
              <p className="text-lg font-medium mb-1">{file.name}</p>
              <p className="text-sm">Preview not available for this file type</p>
            </div>
          )}
        </div>

        {/* File info footer */}
        <div className="px-6 py-3 border-t bg-muted/30 text-sm text-muted-foreground shrink-0">
          <div className="flex items-center gap-6">
            <span>{formatBytes(file.size_bytes)}</span>
            <span>{file.mime_type || 'Unknown type'}</span>
            <span>Modified: {formatDate(file.updated_at)}</span>
            {file.version > 1 && <span>Version: {file.version}</span>}
          </div>
        </div>
      </div>
    </div>
  );
}
