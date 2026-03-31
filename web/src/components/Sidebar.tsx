import { NavLink } from 'react-router-dom';
import { FolderOpen, Users, Clock, Trash2, Settings, LogOut, Cloud } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { cn } from '@/lib/utils';

const navItems = [
  { path: '/files', icon: FolderOpen, label: 'My Files' },
  { path: '/shared', icon: Users, label: 'Shared' },
  { path: '/recent', icon: Clock, label: 'Recent' },
  { path: '/trash', icon: Trash2, label: 'Trash' },
];

export function Sidebar() {
  const { user, logout } = useAuthStore();

  const usedGB = (user?.used_bytes || 0) / (1024 * 1024 * 1024);
  const quotaGB = (user?.quota_bytes || 10 * 1024 * 1024 * 1024) / (1024 * 1024 * 1024);
  const usagePercent = quotaGB > 0 ? (usedGB / quotaGB) * 100 : 0;

  return (
    <aside className="w-64 bg-card border-r flex flex-col">
      {/* Header */}
      <div className="p-6 border-b">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center">
            <Cloud className="w-6 h-6 text-primary-foreground" />
          </div>
          <span className="text-xl font-bold">VaultDrift</span>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-4 space-y-1">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-4 py-3 rounded-lg transition-colors',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground'
              )
            }
          >
            <item.icon className="w-5 h-5" />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Storage Info */}
      <div className="p-4 border-t">
        <div className="mb-2">
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted-foreground">Storage</span>
            <span className="font-medium">{usedGB.toFixed(1)} / {quotaGB.toFixed(0)} GB</span>
          </div>
          <div className="h-2 bg-muted rounded-full overflow-hidden">
            <div
              className="h-full bg-primary transition-all"
              style={{ width: `${Math.min(usagePercent, 100)}%` }}
            />
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="p-4 border-t space-y-1">
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            cn(
              'flex items-center gap-3 px-4 py-3 rounded-lg transition-colors',
              isActive
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            )
          }
        >
          <Settings className="w-5 h-5" />
          <span>Settings</span>
        </NavLink>
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-4 py-3 rounded-lg text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          <LogOut className="w-5 h-5" />
          <span>Logout</span>
        </button>
      </div>
    </aside>
  );
}
