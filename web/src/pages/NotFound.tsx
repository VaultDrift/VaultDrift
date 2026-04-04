import { Link } from 'react-router-dom';
import { FileQuestion, FolderOpen } from 'lucide-react';

export function NotFoundPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-muted-foreground">
      <FileQuestion className="w-20 h-20 mb-6 opacity-50" />
      <h1 className="text-4xl font-bold text-foreground mb-2">404</h1>
      <p className="text-lg mb-8">Page not found</p>
      <p className="text-sm text-muted-foreground mb-6">
        The page you are looking for does not exist or has been moved.
      </p>
      <Link
        to="/files"
        className="flex items-center gap-2 px-6 py-3 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
      >
        <FolderOpen className="w-4 h-4" />
        <span>Go to Files</span>
      </Link>
    </div>
  );
}
