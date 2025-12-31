import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Skeleton } from '@/components/ui/Skeleton';
import { 
  Settings, 
  Database, 
  Users, 
  Key, 
  Activity, 
  Server,
  AlertCircle,
  CheckCircle,
  Clock,
  Plus,
  Trash2,
  Edit,
  AlertTriangle
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { useDashboardOverview } from '@/hooks/useDashboard';

// Admin Configuration and Monitoring Page
export default function AdminPage() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">System Administration</h1>
          <p className="text-muted-foreground">
            Manage system configuration, users, and monitor health
          </p>
        </div>
        <SystemHealthBadge />
      </div>

      <Tabs defaultValue="health" className="space-y-4">
        <TabsList>
          <TabsTrigger value="health">
            <Activity className="w-4 h-4 mr-2" />
            System Health
          </TabsTrigger>
          <TabsTrigger value="probes">
            <Server className="w-4 h-4 mr-2" />
            Probes
          </TabsTrigger>
          <TabsTrigger value="tokens">
            <Key className="w-4 h-4 mr-2" />
            API Tokens
          </TabsTrigger>
          <TabsTrigger value="users">
            <Users className="w-4 h-4 mr-2" />
            Users
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="w-4 h-4 mr-2" />
            Settings
          </TabsTrigger>
          <TabsTrigger value="database">
            <Database className="w-4 h-4 mr-2" />
            Database
          </TabsTrigger>
        </TabsList>

        <TabsContent value="health">
          <SystemHealthDashboard />
        </TabsContent>

        <TabsContent value="probes">
          <ProbeManagement />
        </TabsContent>

        <TabsContent value="tokens">
          <TokenManagement />
        </TabsContent>

        <TabsContent value="users">
          <UserManagement />
        </TabsContent>

        <TabsContent value="settings">
          <SystemSettings />
        </TabsContent>

        <TabsContent value="database">
          <DatabaseManagement />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// System Health Badge Component
function SystemHealthBadge() {
  const { data: healthData, isLoading } = useQuery({
    queryKey: ['admin', 'health'],
    queryFn: () => apiClient.getHealth(),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return <Skeleton className="h-6 w-20" />;
  }

  const status = healthData?.status || 'unknown';
  const variant = status === 'healthy' ? 'success' : 
                  status === 'degraded' ? 'warning' : 'destructive';

  return (
    <Badge variant={variant as any}>
      {status.toUpperCase()}
    </Badge>
  );
}

// System Health Dashboard
function SystemHealthDashboard() {
  const { data: dashboardData, isLoading: dashboardLoading } = useDashboardOverview();
  const { data: healthData, isLoading: healthLoading, error: healthError } = useQuery({
    queryKey: ['admin', 'health'],
    queryFn: () => apiClient.getHealth(),
    refetchInterval: 30000,
  });

  if (healthLoading || dashboardLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-24 w-full" />
          ))}
        </div>
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (healthError) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Error loading system health data
      </div>
    );
  }

  // Use real data from APIs
  const activeClients = dashboardData?.summary?.active_clients || 0;
  const totalEvents = dashboardData?.summary?.total_measurements || 0;
  const errorRate = dashboardData?.summary?.error_rate || 0;
  const avgLatency = dashboardData?.summary?.avg_latency_p95 || 0;
  const eventsPerSecond = totalEvents > 0 ? (totalEvents / (24 * 3600)).toFixed(1) : '0.0';

  return (
    <div className="space-y-6">
      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Active Clients"
          value={activeClients}
          icon={Users}
        />
        <MetricCard
          title="Total Events"
          value={totalEvents}
          icon={Activity}
        />
        <MetricCard
          title="Events/Sec (Est)"
          value={eventsPerSecond}
          icon={Clock}
        />
        <MetricCard
          title="Error Rate"
          value={`${(errorRate * 100).toFixed(1)}%`}
          icon={AlertTriangle}
          status={errorRate > 0.1 ? 'warning' : 'ok'}
        />
      </div>

      {/* Component Health */}
      <Card>
        <CardHeader>
          <CardTitle>Component Health</CardTitle>
          <CardDescription>Status of system components</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <ComponentStatus
              name="Database"
              status={avgLatency > 0 ? 'healthy' : 'warning'}
              latency={avgLatency}
            />
            <ComponentStatus
              name="API Server"
              status={healthData?.status || 'healthy'}
              latency={12.5}
            />
            <ComponentStatus
              name="Data Pipeline"
              status={errorRate > 0.5 ? 'warning' : 'healthy'}
              errorRate={errorRate}
            />
            <ComponentStatus
              name="Event Processing"
              status="healthy"
              message={`${totalEvents} events processed`}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Probe Management Component
function ProbeManagement() {
  const queryClient = useQueryClient();
  const [showCreateForm, setShowCreateForm] = useState(false);

  const { data: probesData, isLoading, error } = useQuery({
    queryKey: ['admin', 'probes'],
    queryFn: () => fetch('/api/v1/admin/probes', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    }).then(res => res.json()),
    refetchInterval: 30000,
  });

  const createProbeMutation = useMutation({
    mutationFn: (probeData: any) => fetch('/api/v1/admin/probes', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(probeData)
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'probes'] });
      setShowCreateForm(false);
    }
  });

  const deleteProbeFunc = (probeId: string) => {
    fetch(`/api/v1/admin/probes/${probeId}`, {
      method: 'DELETE',
      credentials: 'include'
    }).then(() => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'probes'] });
    });
  };

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const probes = probesData?.probes || [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Probe Management</CardTitle>
        <CardDescription>Configure and monitor telemetry probes</CardDescription>
        <div className="flex justify-end">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Add Probe
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {showCreateForm ? (
          <ProbeCreateForm 
            onSubmit={(data: any) => createProbeMutation.mutate(data)}
            onCancel={() => setShowCreateForm(false)}
          />
        ) : (
          <div className="space-y-4">
            {probes.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No probes configured. Click "Add Probe" to get started.
              </div>
            ) : (
              probes.map((probe: any) => (
                <div key={probe.id} className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <h4 className="font-medium">{probe.name || probe.client_id}</h4>
                    <p className="text-sm text-muted-foreground">
                      Targets: {probe.targets?.join(', ') || 'N/A'}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Status: {probe.enabled ? 'Enabled' : 'Disabled'}
                    </p>
                  </div>
                  <div className="flex space-x-2">
                    <Button variant="outline" size="sm">
                      <Edit className="w-4 h-4" />
                    </Button>
                    <Button 
                      variant="destructive" 
                      size="sm"
                      onClick={() => deleteProbeFunc(probe.id)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// Token Management Component
function TokenManagement() {
  const queryClient = useQueryClient();
  const [showCreateForm, setShowCreateForm] = useState(false);

  const { data: tokensData, isLoading } = useQuery({
    queryKey: ['admin', 'tokens'],
    queryFn: () => fetch('/api/v1/admin/tokens', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    }).then(res => res.json()),
    refetchInterval: 30000,
  });

  const createTokenMutation = useMutation({
    mutationFn: (tokenData: any) => fetch('/api/v1/admin/tokens', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(tokenData)
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'tokens'] });
      setShowCreateForm(false);
    }
  });

  const revokeTokenFunc = (tokenId: string) => {
    fetch(`/api/v1/admin/tokens/${tokenId}`, {
      method: 'DELETE',
      credentials: 'include'
    }).then(() => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'tokens'] });
    });
  };

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const tokens = tokensData?.tokens || [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>API Tokens</CardTitle>
        <CardDescription>Manage ingest API authentication tokens</CardDescription>
        <div className="flex justify-end">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Generate Token
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {showCreateForm ? (
          <TokenCreateForm 
            onSubmit={(data: any) => createTokenMutation.mutate(data)}
            onCancel={() => setShowCreateForm(false)}
          />
        ) : (
          <div className="space-y-4">
            {tokens.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No tokens created. Generate a token to allow API access.
              </div>
            ) : (
              tokens.map((token: any) => (
                <div key={token.id} className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <h4 className="font-medium">{token.name}</h4>
                    <p className="text-sm text-muted-foreground font-mono">{token.token_prefix}</p>
                    <p className="text-xs text-muted-foreground">
                      Created: {new Date(token.created_at).toLocaleDateString()}
                      {token.expires_at && ` • Expires: ${new Date(token.expires_at).toLocaleDateString()}`}
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge variant={token.enabled ? 'success' : 'secondary'}>
                      {token.enabled ? 'Active' : 'Disabled'}
                    </Badge>
                    <Button 
                      variant="destructive" 
                      size="sm"
                      onClick={() => revokeTokenFunc(token.id)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// User Management Component
function UserManagement() {
  const queryClient = useQueryClient();
  const [showCreateForm, setShowCreateForm] = useState(false);

  const { data: usersData, isLoading } = useQuery({
    queryKey: ['admin', 'users'],
    queryFn: () => fetch('/api/v1/admin/users', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    }).then(res => res.json()),
    refetchInterval: 30000,
  });

  const createUserMutation = useMutation({
    mutationFn: (userData: any) => fetch('/api/v1/admin/users', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(userData)
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      setShowCreateForm(false);
    }
  });

  const deleteUserFunc = (username: string) => {
    fetch(`/api/v1/admin/users/${username}`, {
      method: 'DELETE',
      credentials: 'include'
    }).then(() => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
    });
  };

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const users = usersData?.users || [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>User Management</CardTitle>
        <CardDescription>Manage system users and permissions</CardDescription>
        <div className="flex justify-end">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Add User
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {showCreateForm ? (
          <UserCreateForm 
            onSubmit={(data: any) => createUserMutation.mutate(data)}
            onCancel={() => setShowCreateForm(false)}
          />
        ) : (
          <div className="space-y-4">
            {users.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No users found. Add a user to get started.
              </div>
            ) : (
              users.map((user: any) => (
                <div key={user.id} className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <h4 className="font-medium">{user.username}</h4>
                    <p className="text-sm text-muted-foreground">{user.email || 'No email set'}</p>
                    <p className="text-xs text-muted-foreground">
                      Role: {user.role} • Created: {new Date(user.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>
                      {user.role}
                    </Badge>
                    <Button 
                      variant="destructive" 
                      size="sm"
                      onClick={() => deleteUserFunc(user.username)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// System Settings Component
function SystemSettings() {
  const { data: settingsData, isLoading } = useQuery({
    queryKey: ['admin', 'settings'],
    queryFn: () => fetch('/api/v1/admin/settings', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    }).then(res => res.json()),
  });

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>System Settings</CardTitle>
        <CardDescription>Configure system-wide settings and thresholds</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Events Retention (Days)</Label>
              <Input 
                type="number" 
                defaultValue={settingsData?.events_retention_days || 7} 
                placeholder="7"
              />
            </div>
            <div>
              <Label>Aggregates Retention (Days)</Label>
              <Input 
                type="number" 
                defaultValue={settingsData?.aggregates_retention_days || 90} 
                placeholder="90"
              />
            </div>
            <div>
              <Label>Latency Threshold (ms)</Label>
              <Input 
                type="number" 
                defaultValue={settingsData?.latency_threshold_ms || 500} 
                placeholder="500"
              />
            </div>
            <div>
              <Label>Error Rate Threshold</Label>
              <Input 
                type="number" 
                step="0.01"
                defaultValue={settingsData?.error_rate_threshold || 0.05} 
                placeholder="0.05"
              />
            </div>
          </div>
          <div className="flex justify-end">
            <Button>Save Settings</Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Database Management Component
function DatabaseManagement() {
  const queryClient = useQueryClient();
  
  const { data: dbStats, isLoading } = useQuery({
    queryKey: ['admin', 'database', 'stats'],
    queryFn: () => fetch('/api/v1/admin/database/stats', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    }).then(res => res.json()),
    refetchInterval: 60000,
  });

  const runCleanupMutation = useMutation({
    mutationFn: (params: any) => fetch('/api/v1/admin/database/maintenance', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params)
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'database'] });
    }
  });

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Database Statistics</CardTitle>
          <CardDescription>Monitor database performance and storage</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <p className="text-2xl font-bold">{dbStats?.events_row_count?.toLocaleString() || 'N/A'}</p>
              <p className="text-sm text-muted-foreground">Events</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{dbStats?.aggregates_row_count?.toLocaleString() || 'N/A'}</p>
              <p className="text-sm text-muted-foreground">Aggregates</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{dbStats?.avg_query_time_ms?.toFixed(1) || 'N/A'}ms</p>
              <p className="text-sm text-muted-foreground">Avg Query Time</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{Math.round((dbStats?.total_database_size || 0) / (1024*1024))}MB</p>
              <p className="text-sm text-muted-foreground">Database Size</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Database Maintenance</CardTitle>
          <CardDescription>Perform cleanup and maintenance operations</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">Clean Old Events</h4>
                <p className="text-sm text-muted-foreground">Remove events older than 30 days</p>
              </div>
              <Button 
                onClick={() => runCleanupMutation.mutate({ type: 'events', older_than: '30d' })}
                disabled={runCleanupMutation.isPending}
              >
                Run Cleanup
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">Vacuum Database</h4>
                <p className="text-sm text-muted-foreground">Reclaim storage space and optimize</p>
              </div>
              <Button 
                onClick={() => runCleanupMutation.mutate({ type: 'vacuum' })}
                disabled={runCleanupMutation.isPending}
              >
                Run Vacuum
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Helper Components
function MetricCard({ title, value, icon: Icon, status = 'ok' }: any) {
  const statusColors: Record<string, string> = {
    ok: 'text-green-600',
    warning: 'text-yellow-600',
    error: 'text-red-600'
  };

  return (
    <Card>
      <CardContent className="flex items-center p-6">
        <Icon className={`h-8 w-8 ${statusColors[status] || statusColors.ok}`} />
        <div className="ml-4">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="text-2xl font-bold">{value}</p>
        </div>
      </CardContent>
    </Card>
  );
}

function ComponentStatus({ name, status, latency, errorRate, message }: any) {
  const statusIcon = status === 'healthy' ? CheckCircle : 
                    status === 'warning' ? AlertCircle : 
                    AlertTriangle;
  const statusColor = status === 'healthy' ? 'text-green-600' : 
                     status === 'warning' ? 'text-yellow-600' : 
                     'text-red-600';

  const Icon = statusIcon;

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center space-x-3">
        <Icon className={`h-5 w-5 ${statusColor}`} />
        <span className="font-medium">{name}</span>
      </div>
      <div className="text-right text-sm text-muted-foreground">
        {latency && <div>Latency: {latency}ms</div>}
        {errorRate && <div>Error Rate: {(errorRate * 100).toFixed(2)}%</div>}
        {message && <div>{message}</div>}
      </div>
    </div>
  );
}

// Form Components (simplified versions)
function ProbeCreateForm({ onSubmit, onCancel }: any) {
  const [formData, setFormData] = useState({
    client_id: '',
    name: '',
    targets: [''],
    interval: 60,
    enabled: true
  });

  return (
    <div className="space-y-4 border rounded-lg p-4">
      <h3 className="font-medium">Create New Probe</h3>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>Client ID</Label>
          <Input 
            value={formData.client_id}
            onChange={(e) => setFormData({...formData, client_id: e.target.value})}
            placeholder="my-probe"
          />
        </div>
        <div>
          <Label>Name</Label>
          <Input 
            value={formData.name}
            onChange={(e) => setFormData({...formData, name: e.target.value})}
            placeholder="My Probe"
          />
        </div>
      </div>
      <div>
        <Label>Target URL</Label>
        <Input 
          value={formData.targets[0]}
          onChange={(e) => setFormData({...formData, targets: [e.target.value]})}
          placeholder="https://example.com"
        />
      </div>
      <div className="flex space-x-2">
        <Button onClick={() => onSubmit(formData)}>Create</Button>
        <Button variant="outline" onClick={onCancel}>Cancel</Button>
      </div>
    </div>
  );
}

function TokenCreateForm({ onSubmit, onCancel }: any) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    expires_at: null
  });

  return (
    <div className="space-y-4 border rounded-lg p-4">
      <h3 className="font-medium">Generate API Token</h3>
      <div>
        <Label>Token Name</Label>
        <Input 
          value={formData.name}
          onChange={(e) => setFormData({...formData, name: e.target.value})}
          placeholder="Production Ingest Token"
        />
      </div>
      <div>
        <Label>Description</Label>
        <Input 
          value={formData.description}
          onChange={(e) => setFormData({...formData, description: e.target.value})}
          placeholder="Token for production probe authentication"
        />
      </div>
      <div className="flex space-x-2">
        <Button onClick={() => onSubmit(formData)}>Generate</Button>
        <Button variant="outline" onClick={onCancel}>Cancel</Button>
      </div>
    </div>
  );
}

function UserCreateForm({ onSubmit, onCancel }: any) {
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    role: 'user',
    email: ''
  });

  return (
    <div className="space-y-4 border rounded-lg p-4">
      <h3 className="font-medium">Create New User</h3>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>Username</Label>
          <Input 
            value={formData.username}
            onChange={(e) => setFormData({...formData, username: e.target.value})}
            placeholder="johndoe"
          />
        </div>
        <div>
          <Label>Email</Label>
          <Input 
            type="email"
            value={formData.email}
            onChange={(e) => setFormData({...formData, email: e.target.value})}
            placeholder="john@example.com"
          />
        </div>
      </div>
      <div>
        <Label>Password</Label>
        <Input 
          type="password"
          value={formData.password}
          onChange={(e) => setFormData({...formData, password: e.target.value})}
          placeholder="••••••••"
        />
      </div>
      <div>
        <Label>Role</Label>
        <Select value={formData.role} onValueChange={(value: string) => setFormData({...formData, role: value})}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="user">User</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="flex space-x-2">
        <Button onClick={() => onSubmit(formData)}>Create User</Button>
        <Button variant="outline" onClick={onCancel}>Cancel</Button>
      </div>
    </div>
  );
}
