import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/Button';
import { 
  Settings, 
  Database, 
  Users, 
  Key, 
  Activity, 
  Server,
  AlertCircle,
  CheckCircle,
  Clock
} from 'lucide-react';

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
  const health = { status: 'healthy' }; // TODO: Fetch from API

  const statusConfig = {
    healthy: { icon: CheckCircle, color: 'bg-green-500', text: 'Healthy' },
    degraded: { icon: AlertCircle, color: 'bg-yellow-500', text: 'Degraded' },
    unhealthy: { icon: AlertCircle, color: 'bg-red-500', text: 'Unhealthy' }
  };

  const config = statusConfig[health.status as keyof typeof statusConfig];
  const Icon = config.icon;

  return (
    <Badge className={`${config.color} text-white`}>
      <Icon className="w-4 h-4 mr-1" />
      System {config.text}
    </Badge>
  );
}

// System Health Dashboard
function SystemHealthDashboard() {
  // Mock data - would be fetched from /api/v1/admin/health
  const health = {
    status: 'healthy',
    activeProbes: 47,
    activeClients: 23,
    eventsPerSecond: 156.8,
    queueLag: 125,
    errorRate: 0.015,
    avgProcessingTime: 45.2,
    database: { status: 'healthy', latency: 15.3 },
    nats: { status: 'healthy', latency: 2.1 },
    aggregator: { status: 'healthy', latency: 45.2, errorRate: 0.001 },
    ingestAPI: { status: 'healthy', latency: 12.5, errorRate: 0.002 },
    websocket: { status: 'healthy', message: '125 active connections' }
  };

  return (
    <div className="space-y-6">
      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Active Probes"
          value={health.activeProbes}
          icon={Server}
        />
        <MetricCard
          title="Active Clients"
          value={health.activeClients}
          icon={Users}
        />
        <MetricCard
          title="Events/Second"
          value={health.eventsPerSecond.toFixed(1)}
          icon={Activity}
        />
        <MetricCard
          title="Queue Lag"
          value={health.queueLag}
          icon={Clock}
          status={health.queueLag > 1000 ? 'warning' : 'ok'}
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
              status={health.database.status}
              latency={health.database.latency}
            />
            <ComponentStatus
              name="NATS JetStream"
              status={health.nats.status}
              latency={health.nats.latency}
            />
            <ComponentStatus
              name="Aggregator"
              status={health.aggregator.status}
              latency={health.aggregator.latency}
              errorRate={health.aggregator.errorRate}
            />
            <ComponentStatus
              name="Ingest API"
              status={health.ingestAPI.status}
              latency={health.ingestAPI.latency}
              errorRate={health.ingestAPI.errorRate}
            />
            <ComponentStatus
              name="WebSocket"
              status={health.websocket.status}
              message={health.websocket.message}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Probe Management Component
function ProbeManagement() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Probe Configuration</CardTitle>
        <CardDescription>Manage probe agents and their targets</CardDescription>
        <div className="flex justify-end">
          <Button>Add Probe</Button>
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">
          Probe management UI - Create, update, and monitor probe configurations.
          Would include a table with probe list, status, targets, and actions.
        </p>
      </CardContent>
    </Card>
  );
}

// Token Management Component
function TokenManagement() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>API Tokens</CardTitle>
        <CardDescription>Manage ingest API authentication tokens</CardDescription>
        <div className="flex justify-end">
          <Button>Generate Token</Button>
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">
          Token management UI - Create, revoke, and monitor API tokens.
          Would show token list with usage stats and expiration dates.
        </p>
      </CardContent>
    </Card>
  );
}

// User Management Component
function UserManagement() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>User Management</CardTitle>
        <CardDescription>Manage system users and permissions</CardDescription>
        <div className="flex justify-end">
          <Button>Add User</Button>
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">
          User management UI - Create, update, and manage user accounts with role-based access control.
        </p>
      </CardContent>
    </Card>
  );
}

// System Settings Component
function SystemSettings() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>System Settings</CardTitle>
        <CardDescription>Configure system parameters and thresholds</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <h3 className="font-semibold mb-2">Data Retention</h3>
            <p className="text-sm text-muted-foreground">
              Events Retention: 7 days
              <br />
              Aggregates Retention: 90 days
            </p>
          </div>
          <div>
            <h3 className="font-semibold mb-2">Performance Thresholds</h3>
            <p className="text-sm text-muted-foreground">
              Latency Threshold: 500ms
              <br />
              Throughput Threshold: 10 Mbps
              <br />
              Error Rate Threshold: 5%
            </p>
          </div>
        </div>
        <Button className="mt-4">Update Settings</Button>
      </CardContent>
    </Card>
  );
}

// Database Management Component
function DatabaseManagement() {
  const stats = {
    activeConnections: 5,
    eventsTableSize: '100 MB',
    aggregatesTableSize: '500 MB',
    eventsRowCount: 150000,
    aggregatesRowCount: 50000
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Database Statistics</CardTitle>
          <CardDescription>Current database metrics</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <p className="text-sm font-medium">Active Connections</p>
              <p className="text-2xl font-bold">{stats.activeConnections}</p>
            </div>
            <div>
              <p className="text-sm font-medium">Events Table</p>
              <p className="text-2xl font-bold">{stats.eventsTableSize}</p>
              <p className="text-xs text-muted-foreground">
                {stats.eventsRowCount.toLocaleString()} rows
              </p>
            </div>
            <div>
              <p className="text-sm font-medium">Aggregates Table</p>
              <p className="text-2xl font-bold">{stats.aggregatesTableSize}</p>
              <p className="text-xs text-muted-foreground">
                {stats.aggregatesRowCount.toLocaleString()} rows
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Maintenance Operations</CardTitle>
          <CardDescription>Run database maintenance tasks</CardDescription>
        </CardHeader>
        <CardContent className="space-x-2">
          <Button variant="outline">Run Vacuum</Button>
          <Button variant="outline">Run Analyze</Button>
          <Button variant="outline">Cleanup Old Data</Button>
        </CardContent>
      </Card>
    </div>
  );
}

// Helper Components
function MetricCard({ 
  title, 
  value, 
  icon: Icon, 
  status = 'ok' 
}: { 
  title: string; 
  value: number | string; 
  icon: React.ElementType; 
  status?: 'ok' | 'warning' | 'error';
}) {
  const statusColors = {
    ok: 'text-green-600',
    warning: 'text-yellow-600',
    error: 'text-red-600'
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className={`h-4 w-4 ${statusColors[status]}`} />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
      </CardContent>
    </Card>
  );
}

function ComponentStatus({
  name,
  status,
  latency,
  errorRate,
  message
}: {
  name: string;
  status: string;
  latency?: number;
  errorRate?: number;
  message?: string;
}) {
  const statusConfig = {
    healthy: { icon: CheckCircle, color: 'text-green-600', bg: 'bg-green-50' },
    degraded: { icon: AlertCircle, color: 'text-yellow-600', bg: 'bg-yellow-50' },
    unhealthy: { icon: AlertCircle, color: 'text-red-600', bg: 'bg-red-50' },
    unknown: { icon: AlertCircle, color: 'text-gray-400', bg: 'bg-gray-50' }
  };

  const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.unknown;
  const Icon = config.icon;

  return (
    <div className={`flex items-center justify-between p-3 rounded-lg ${config.bg}`}>
      <div className="flex items-center space-x-3">
        <Icon className={`h-5 w-5 ${config.color}`} />
        <span className="font-medium">{name}</span>
      </div>
      <div className="flex items-center space-x-4 text-sm text-muted-foreground">
        {latency && <span>{latency.toFixed(1)}ms</span>}
        {errorRate !== undefined && <span>{(errorRate * 100).toFixed(2)}% errors</span>}
        {message && <span>{message}</span>}
      </div>
    </div>
  );
}
