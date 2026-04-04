import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Users,
  HardDrive,
  Activity,
  Settings,
  Plus,
  Trash2,
  Edit2,
  Shield,
  User,
  Loader2,
  Search,
} from 'lucide-react';
import { toast } from 'sonner';
import { adminApi } from '@/lib/api';
import { formatBytes, formatDate, cn } from '@/lib/utils';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export function AdminPage() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [newUser, setNewUser] = useState({
    username: '',
    email: '',
    password: '',
    display_name: '',
    role: 'user',
    quota_bytes: 10 * 1024 * 1024 * 1024, // 10GB default
  });
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<{
    id: string;
    email: string;
    role: string;
    status: string;
    quota_bytes: number;
    reset_password: string;
  } | null>(null);

  // Fetch users
  const { data: users, isLoading: isLoadingUsers } = useQuery({
    queryKey: ['admin', 'users'],
    queryFn: adminApi.getUsers,
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: adminApi.getStats,
  });

  // Fetch system health
  const { data: health } = useQuery({
    queryKey: ['admin', 'health'],
    queryFn: adminApi.getSystemHealth,
    refetchInterval: 30000,
  });

  // Create user mutation
  const createUserMutation = useMutation({
    mutationFn: adminApi.createUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      toast.success('User created successfully');
      setIsCreateDialogOpen(false);
      setNewUser({
        username: '',
        email: '',
        password: '',
        display_name: '',
        role: 'user',
        quota_bytes: 10 * 1024 * 1024 * 1024,
      });
    },
    onError: () => toast.error('Failed to create user'),
  });

  // Delete user mutation
  const deleteUserMutation = useMutation({
    mutationFn: adminApi.deleteUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      toast.success('User deleted');
    },
    onError: () => toast.error('Failed to delete user'),
  });

  // Update user mutation
  const updateUserMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) =>
      adminApi.updateUser(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      toast.success('User updated');
      setIsEditDialogOpen(false);
      setEditingUser(null);
    },
    onError: () => toast.error('Failed to update user'),
  });

  const openEditDialog = (user: { id: string; email: string; role: string; status: string; quota_bytes: number }) => {
    setEditingUser({
      id: user.id,
      email: user.email,
      role: user.role,
      status: user.status,
      quota_bytes: user.quota_bytes,
      reset_password: '',
    });
    setIsEditDialogOpen(true);
  };

  const handleUpdateUser = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingUser) return;
    const data: Record<string, unknown> = {};
    if (editingUser.email) data.email = editingUser.email;
    if (editingUser.role) data.role = editingUser.role;
    if (editingUser.status) data.status = editingUser.status;
    if (editingUser.quota_bytes > 0) data.quota_bytes = editingUser.quota_bytes;
    if (editingUser.reset_password) data.reset_password = editingUser.reset_password;
    updateUserMutation.mutate({ id: editingUser.id, data });
  };

  const handleDeleteUser = (user: { username: string; id: string }) => {
    if (!confirm(`Are you sure you want to delete user "${user.username}"? This action cannot be undone.`)) return;
    deleteUserMutation.mutate(user.id);
  };

  const filteredUsers = users?.filter(
    (user) =>
      user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      user.email.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleCreateUser = (e: React.FormEvent) => {
    e.preventDefault();
    createUserMutation.mutate(newUser);
  };

  return (
    <div className="h-full flex flex-col overflow-auto">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card shrink-0">
        <h1 className="text-lg font-semibold">Admin Panel</h1>
        <div className="flex items-center gap-2">
          <Shield className="w-4 h-4 text-primary" />
          <span className="text-sm text-muted-foreground">Administrator</span>
        </div>
      </header>

      {/* Content */}
      <div className="p-6 space-y-6">
        <Tabs defaultValue="overview" className="space-y-6">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="users">Users</TabsTrigger>
            <TabsTrigger value="system">System</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-6">
            {/* Stats Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Total Users</CardTitle>
                  <Users className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats?.users?.total || 0}</div>
                  <p className="text-xs text-muted-foreground">
                    {stats?.users?.active || 0} active
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Total Files</CardTitle>
                  <HardDrive className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats?.storage?.total_files || 0}</div>
                  <p className="text-xs text-muted-foreground">
                    {formatBytes(stats?.storage?.total_bytes || 0)} stored
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Active Shares</CardTitle>
                  <Activity className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats?.storage?.active_shares || 0}</div>
                  <p className="text-xs text-muted-foreground">
                    {stats?.storage?.total_chunks || 0} chunks
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">System Load</CardTitle>
                  <Settings className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {health?.system?.goroutines || 0}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {health?.system?.memory_alloc_mb || 0} MB memory
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Storage Usage */}
            <Card>
              <CardHeader>
                <CardTitle>Storage Overview</CardTitle>
                <CardDescription>
                  Total storage usage across all users
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Used Storage</span>
                    <span className="font-medium">
                      {formatBytes(stats?.storage?.total_bytes || 0)} /{' '}
                      {formatBytes(Number(stats?.storage?.storage_backend) || 0)}
                    </span>
                  </div>
                  <div className="h-2 bg-muted rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary transition-all rounded-full"
                      style={{
                        width: `${Math.min(
                          ((stats?.storage?.total_bytes || 0) / (Number(stats?.storage?.storage_backend) || 1)) * 100,
                          100
                        )}%`,
                      }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Users Tab */}
          <TabsContent value="users" className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="relative max-w-sm flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  type="text"
                  placeholder="Search users..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>

              <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                <DialogTrigger asChild>
                  <Button>
                    <Plus className="w-4 h-4 mr-2" />
                    Add User
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Create New User</DialogTitle>
                    <DialogDescription>
                      Add a new user to the system
                    </DialogDescription>
                  </DialogHeader>
                  <form onSubmit={handleCreateUser} className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label htmlFor="username">Username</Label>
                        <Input
                          id="username"
                          value={newUser.username}
                          onChange={(e) =>
                            setNewUser({ ...newUser, username: e.target.value })
                          }
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="display_name">Display Name</Label>
                        <Input
                          id="display_name"
                          value={newUser.display_name}
                          onChange={(e) =>
                            setNewUser({ ...newUser, display_name: e.target.value })
                          }
                        />
                      </div>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="email">Email</Label>
                      <Input
                        id="email"
                        type="email"
                        value={newUser.email}
                        onChange={(e) =>
                          setNewUser({ ...newUser, email: e.target.value })
                        }
                        required
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="password">Password</Label>
                      <Input
                        id="password"
                        type="password"
                        value={newUser.password}
                        onChange={(e) =>
                          setNewUser({ ...newUser, password: e.target.value })
                        }
                        required
                      />
                    </div>
                    <DialogFooter>
                      <Button
                        type="submit"
                        disabled={createUserMutation.isPending}
                      >
                        {createUserMutation.isPending && (
                          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        )}
                        Create User
                      </Button>
                    </DialogFooter>
                  </form>
                </DialogContent>
              </Dialog>
            </div>

            <Card>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Storage</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-24">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoadingUsers ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8">
                        <Loader2 className="w-6 h-6 animate-spin mx-auto" />
                      </TableCell>
                    </TableRow>
                  ) : filteredUsers?.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                        No users found
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredUsers?.map((user) => (
                      <TableRow key={user.id}>
                        <TableCell>
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                              <User className="w-4 h-4 text-primary" />
                            </div>
                            <div>
                              <p className="font-medium">{user.display_name || user.username}</p>
                              <p className="text-sm text-muted-foreground">{user.email}</p>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span
                            className={cn(
                              'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                              user.role === 'admin'
                                ? 'bg-primary/10 text-primary'
                                : 'bg-muted text-muted-foreground'
                            )}
                          >
                            {user.role}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm">
                            {formatBytes(user.used_bytes)} /{' '}
                            {formatBytes(user.quota_bytes)}
                          </div>
                          <div className="w-24 h-1.5 bg-muted rounded-full mt-1 overflow-hidden">
                            <div
                              className="h-full bg-primary rounded-full"
                              style={{
                                width: `${Math.min(
                                  (user.used_bytes / user.quota_bytes) * 100,
                                  100
                                )}%`,
                              }}
                            />
                          </div>
                        </TableCell>
                        <TableCell>
                          <span
                            className={cn(
                              'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                              user.status === 'active'
                                ? 'bg-primary/10 text-primary'
                                : 'bg-destructive/10 text-destructive'
                            )}
                          >
                            {user.status}
                          </span>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(user.created_at)}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() => openEditDialog(user)}
                            >
                              <Edit2 className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8 text-destructive hover:text-destructive"
                              onClick={() => handleDeleteUser(user)}
                              disabled={deleteUserMutation.isPending}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </Card>

            {/* Edit User Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Edit User</DialogTitle>
                  <DialogDescription>
                    Update user details. Leave password blank to keep unchanged.
                  </DialogDescription>
                </DialogHeader>
                {editingUser && (
                  <form onSubmit={handleUpdateUser} className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="edit-email">Email</Label>
                      <Input
                        id="edit-email"
                        type="email"
                        value={editingUser.email}
                        onChange={(e) =>
                          setEditingUser({ ...editingUser, email: e.target.value })
                        }
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label htmlFor="edit-role">Role</Label>
                        <select
                          id="edit-role"
                          value={editingUser.role}
                          onChange={(e) =>
                            setEditingUser({ ...editingUser, role: e.target.value })
                          }
                          className="w-full h-10 rounded-md border bg-background px-3 text-sm"
                        >
                          <option value="user">User</option>
                          <option value="admin">Admin</option>
                        </select>
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="edit-status">Status</Label>
                        <select
                          id="edit-status"
                          value={editingUser.status}
                          onChange={(e) =>
                            setEditingUser({ ...editingUser, status: e.target.value })
                          }
                          className="w-full h-10 rounded-md border bg-background px-3 text-sm"
                        >
                          <option value="active">Active</option>
                          <option value="suspended">Suspended</option>
                          <option value="disabled">Disabled</option>
                        </select>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-quota">Quota (bytes)</Label>
                      <Input
                        id="edit-quota"
                        type="number"
                        value={editingUser.quota_bytes}
                        onChange={(e) =>
                          setEditingUser({ ...editingUser, quota_bytes: parseInt(e.target.value) || 0 })
                        }
                      />
                      <p className="text-xs text-muted-foreground">
                        Current: {formatBytes(editingUser.quota_bytes)}
                      </p>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-password">Reset Password</Label>
                      <Input
                        id="edit-password"
                        type="password"
                        placeholder="Leave blank to keep current"
                        value={editingUser.reset_password}
                        onChange={(e) =>
                          setEditingUser({ ...editingUser, reset_password: e.target.value })
                        }
                      />
                    </div>
                    <DialogFooter>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setIsEditDialogOpen(false)}
                      >
                        Cancel
                      </Button>
                      <Button type="submit" disabled={updateUserMutation.isPending}>
                        {updateUserMutation.isPending && (
                          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        )}
                        Save Changes
                      </Button>
                    </DialogFooter>
                  </form>
                )}
              </DialogContent>
            </Dialog>
          </TabsContent>

          {/* System Tab */}
          <TabsContent value="system" className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle>System Health</CardTitle>
                  <CardDescription>
                    Overall status: {health?.status || 'unknown'}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted/50 rounded-lg">
                      <p className="text-sm text-muted-foreground">Goroutines</p>
                      <p className="text-2xl font-bold">{health?.system?.goroutines || 0}</p>
                    </div>
                    <div className="p-3 bg-muted/50 rounded-lg">
                      <p className="text-sm text-muted-foreground">CPU Cores</p>
                      <p className="text-2xl font-bold">{health?.system?.cpu_cores || 0}</p>
                    </div>
                    <div className="p-3 bg-muted/50 rounded-lg">
                      <p className="text-sm text-muted-foreground">Memory Alloc</p>
                      <p className="text-2xl font-bold">{health?.system?.memory_alloc_mb || 0} MB</p>
                    </div>
                    <div className="p-3 bg-muted/50 rounded-lg">
                      <p className="text-sm text-muted-foreground">GC Count</p>
                      <p className="text-2xl font-bold">{health?.system?.gc_count || 0}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Services</CardTitle>
                  <CardDescription>Dependency health</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {['database', 'storage'].map((service) => (
                      <div
                        key={service}
                        className="flex items-center justify-between p-3 bg-muted/50 rounded-lg"
                      >
                        <div className="flex items-center gap-3">
                          <div
                            className={cn(
                              'w-2 h-2 rounded-full',
                              health?.[service] === 'healthy'
                                ? 'bg-primary'
                                : 'bg-destructive'
                            )}
                          />
                          <span className="font-medium capitalize">{service}</span>
                        </div>
                        <span className={cn(
                          'text-xs font-medium px-2 py-1 rounded-full',
                          health?.[service] === 'healthy'
                            ? 'bg-primary/10 text-primary'
                            : 'bg-destructive/10 text-destructive'
                        )}>
                          {String(health?.[service] || 'unknown')}
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
