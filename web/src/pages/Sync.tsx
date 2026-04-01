import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Smartphone,
  Laptop,
  Tablet,
  RefreshCw,
  Check,
  AlertCircle,
  X,
  Clock,
  Loader2,
  Unlink,
  HardDrive,
  Activity,
} from 'lucide-react';
import { toast } from 'sonner';
import { syncApi } from '@/lib/api';
import { cn, formatDate, formatBytes } from '@/lib/utils';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

interface Device {
  id: string;
  name: string;
  type: 'desktop' | 'mobile' | 'tablet' | 'web';
  platform: string;
  last_sync: string;
  status: 'online' | 'offline' | 'syncing';
  sync_count: number;
}

interface SyncSession {
  id: string;
  device_name: string;
  started_at: string;
  completed_at?: string;
  status: 'pending' | 'syncing' | 'completed' | 'failed';
  files_synced: number;
  bytes_transferred: number;
}

export function SyncPage() {
  const queryClient = useQueryClient();
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [isRevokeDialogOpen, setIsRevokeDialogOpen] = useState(false);

  // Fetch devices
  const { data: devices, isLoading: isLoadingDevices } = useQuery({
    queryKey: ['sync', 'devices'],
    queryFn: syncApi.getDevices,
  });

  // Fetch sessions
  const { data: sessions, isLoading: isLoadingSessions } = useQuery({
    queryKey: ['sync', 'sessions'],
    queryFn: syncApi.getSessions,
  });

  // Fetch sync status
  const { data: syncStatus } = useQuery({
    queryKey: ['sync', 'status'],
    queryFn: syncApi.getSyncStatus,
    refetchInterval: 5000,
  });

  // Revoke device mutation
  const revokeDeviceMutation = useMutation({
    mutationFn: syncApi.revokeDevice,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sync', 'devices'] });
      toast.success('Device revoked');
      setIsRevokeDialogOpen(false);
      setSelectedDevice(null);
    },
    onError: () => toast.error('Failed to revoke device'),
  });

  const getDeviceIcon = (type: string) => {
    switch (type) {
      case 'mobile':
        return <Smartphone className="w-5 h-5" />;
      case 'tablet':
        return <Tablet className="w-5 h-5" />;
      case 'web':
        return <RefreshCw className="w-5 h-5" />;
      default:
        return <Laptop className="w-5 h-5" />;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
      case 'completed':
        return <Check className="w-4 h-4 text-green-500" />;
      case 'syncing':
      case 'pending':
        return <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />;
      case 'failed':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      default:
        return <X className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const activeDevices = devices?.filter((d: Device) => d.status === 'online').length || 0;
  const totalSyncs = devices?.reduce((sum: number, d: Device) => sum + d.sync_count, 0) || 0;

  return (
    <div className="h-full flex flex-col overflow-auto">
      {/* Header */}
      <header className="h-16 border-b px-6 flex items-center justify-between bg-card shrink-0">
        <h1 className="text-lg font-semibold">Sync Dashboard</h1>
        <div className="flex items-center gap-2">
          <div
            className={cn(
              'w-2 h-2 rounded-full',
              syncStatus?.status === 'connected'
                ? 'bg-green-500'
                : syncStatus?.status === 'syncing'
                ? 'bg-blue-500 animate-pulse'
                : 'bg-red-500'
            )}
          />
          <span className="text-sm text-muted-foreground capitalize">
            {syncStatus?.status || 'Disconnected'}
          </span>
        </div>
      </header>

      {/* Content */}
      <div className="p-6 space-y-6">
        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Connected Devices</CardTitle>
              <Laptop className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{devices?.length || 0}</div>
              <p className="text-xs text-muted-foreground">{activeDevices} online</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Syncs</CardTitle>
              <RefreshCw className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalSyncs}</div>
              <p className="text-xs text-muted-foreground">All time</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Synced Data</CardTitle>
              <HardDrive className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatBytes(syncStatus?.total_bytes_synced || 0)}
              </div>
              <p className="text-xs text-muted-foreground">This month</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Sync Queue</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {syncStatus?.pending_files || 0}
              </div>
              <p className="text-xs text-muted-foreground">Files pending</p>
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="devices" className="space-y-4">
          <TabsList>
            <TabsTrigger value="devices">Devices</TabsTrigger>
            <TabsTrigger value="history">Sync History</TabsTrigger>
          </TabsList>

          {/* Devices Tab */}
          <TabsContent value="devices">
            <Card>
              <CardHeader>
                <CardTitle>Connected Devices</CardTitle>
                <CardDescription>
                  Manage devices connected to your account
                </CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingDevices ? (
                  <div className="flex items-center justify-center py-8">
                    <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
                  </div>
                ) : devices?.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <Laptop className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>No devices connected</p>
                    <p className="text-sm">Install VaultDrift on your devices to sync files</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {devices?.map((device: Device) => (
                      <div
                        key={device.id}
                        className="p-4 border rounded-lg hover:border-primary transition-colors"
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                              {getDeviceIcon(device.type)}
                            </div>
                            <div>
                              <p className="font-medium">{device.name}</p>
                              <p className="text-sm text-muted-foreground">
                                {device.platform}
                              </p>
                            </div>
                          </div>
                          <div
                            className={cn(
                              'w-2 h-2 rounded-full',
                              device.status === 'online'
                                ? 'bg-green-500'
                                : device.status === 'syncing'
                                ? 'bg-blue-500 animate-pulse'
                                : 'bg-muted'
                            )}
                          />
                        </div>

                        <div className="mt-4 space-y-2">
                          <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Clock className="w-4 h-4" />
                            <span>Last sync: {formatDate(device.last_sync)}</span>
                          </div>
                          <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <RefreshCw className="w-4 h-4" />
                            <span>{device.sync_count} syncs</span>
                          </div>
                        </div>

                        <div className="mt-4 flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            className="flex-1"
                            onClick={() => {
                              setSelectedDevice(device);
                              setIsRevokeDialogOpen(true);
                            }}
                          >
                            <Unlink className="w-4 h-4 mr-2" />
                            Revoke
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* History Tab */}
          <TabsContent value="history">
            <Card>
              <CardHeader>
                <CardTitle>Recent Sync Activity</CardTitle>
                <CardDescription>View your recent sync sessions</CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Device</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Files</TableHead>
                      <TableHead>Data</TableHead>
                      <TableHead>Started</TableHead>
                      <TableHead>Completed</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {isLoadingSessions ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-8">
                          <Loader2 className="w-6 h-6 animate-spin mx-auto" />
                        </TableCell>
                      </TableRow>
                    ) : sessions?.length === 0 ? (
                      <TableRow>
                        <TableCell
                          colSpan={6}
                          className="text-center py-8 text-muted-foreground"
                        >
                          No sync history
                        </TableCell>
                      </TableRow>
                    ) : (
                      sessions?.map((session: SyncSession) => (
                        <TableRow key={session.id}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <Laptop className="w-4 h-4 text-muted-foreground" />
                              <span>{session.device_name}</span>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              {getStatusIcon(session.status)}
                              <span className="capitalize">{session.status}</span>
                            </div>
                          </TableCell>
                          <TableCell>{session.files_synced}</TableCell>
                          <TableCell>
                            {formatBytes(session.bytes_transferred)}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {formatDate(session.started_at)}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {session.completed_at
                              ? formatDate(session.completed_at)
                              : '-'}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      {/* Revoke Device Dialog */}
      <Dialog open={isRevokeDialogOpen} onOpenChange={setIsRevokeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke Device Access</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke access for {selectedDevice?.name}? This
              device will no longer be able to sync files.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsRevokeDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                selectedDevice && revokeDeviceMutation.mutate(selectedDevice.id)
              }
              disabled={revokeDeviceMutation.isPending}
            >
              {revokeDeviceMutation.isPending && (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              )}
              Revoke Access
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
