import { useState, useEffect } from 'react';
import { useLiveProbes } from '@/hooks/useLiveUpdates';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Badge } from '@/components/ui/badge';
import { Activity, CheckCircle, AlertCircle, Clock, Wifi, WifiOff } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ProbeStatus {
  client_id: string;
  status: 'active' | 'inactive' | 'error' | 'unknown';
  last_seen: Date;
  latency_avg?: number;
  error_rate?: number;
}

interface LiveProbeStatusProps {
  token: string;
  maxProbes?: number;
}

export function LiveProbeStatus({ token, maxProbes = 20 }: LiveProbeStatusProps) {
  const [probes, setProbes] = useState<Map<string, ProbeStatus>>(new Map());
  const { isConnected } = useLiveProbes(token);

  // Update probe status based on last_seen time
  useEffect(() => {
    const interval = setInterval(() => {
      setProbes((prev) => {
        const updated = new Map(prev);
        const now = new Date();
        
        updated.forEach((probe, clientId) => {
          const timeSinceLastSeen = now.getTime() - probe.last_seen.getTime();
          
          if (timeSinceLastSeen > 5 * 60 * 1000) {
            // More than 5 minutes since last seen
            probe.status = 'inactive';
          } else if (timeSinceLastSeen > 2 * 60 * 1000) {
            // More than 2 minutes since last seen
            probe.status = 'unknown';
          }
        });
        
        return updated;
      });
    }, 10000); // Check every 10 seconds

    return () => clearInterval(interval);
  }, []);

  const probesList = Array.from(probes.values())
    .sort((a, b) => b.last_seen.getTime() - a.last_seen.getTime())
    .slice(0, maxProbes);

  const statusCounts = {
    active: probesList.filter((p) => p.status === 'active').length,
    inactive: probesList.filter((p) => p.status === 'inactive').length,
    error: probesList.filter((p) => p.status === 'error').length,
    unknown: probesList.filter((p) => p.status === 'unknown').length,
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Live Probe Status
          </CardTitle>
          <Badge variant={isConnected ? 'default' : 'secondary'}>
            {probesList.length} Probes
          </Badge>
        </div>
        
        {/* Status summary */}
        <div className="flex items-center gap-4 mt-4">
          <StatusBadge count={statusCounts.active} label="Active" variant="success" />
          <StatusBadge count={statusCounts.inactive} label="Inactive" variant="secondary" />
          <StatusBadge count={statusCounts.error} label="Error" variant="destructive" />
          <StatusBadge count={statusCounts.unknown} label="Unknown" variant="warning" />
        </div>
      </CardHeader>
      
      <CardContent>
        <div className="space-y-2">
          {probesList.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-gray-500">
              <div className="text-center">
                <Wifi className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No probes detected</p>
              </div>
            </div>
          ) : (
            probesList.map((probe) => (
              <ProbeCard key={probe.client_id} probe={probe} />
            ))
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface ProbeCardProps {
  probe: ProbeStatus;
}

function ProbeCard({ probe }: ProbeCardProps) {
  const getStatusConfig = () => {
    switch (probe.status) {
      case 'active':
        return {
          icon: CheckCircle,
          color: 'text-green-500',
          bgColor: 'bg-green-100 dark:bg-green-900/20',
          label: 'Active',
        };
      case 'error':
        return {
          icon: AlertCircle,
          color: 'text-red-500',
          bgColor: 'bg-red-100 dark:bg-red-900/20',
          label: 'Error',
        };
      case 'inactive':
        return {
          icon: WifiOff,
          color: 'text-gray-500',
          bgColor: 'bg-gray-100 dark:bg-gray-900/20',
          label: 'Inactive',
        };
      case 'unknown':
      default:
        return {
          icon: Clock,
          color: 'text-yellow-500',
          bgColor: 'bg-yellow-100 dark:bg-yellow-900/20',
          label: 'Unknown',
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;
  const timeSinceLastSeen = Date.now() - probe.last_seen.getTime();
  const lastSeenText = formatTimeSince(timeSinceLastSeen);

  return (
    <div className="flex items-center gap-3 p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors">
      <div className={cn('p-2 rounded-full', config.bgColor)}>
        <Icon className={cn('h-4 w-4', config.color)} />
      </div>
      
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm truncate">{probe.client_id}</span>
          <Badge variant="outline" className="text-xs">
            {config.label}
          </Badge>
        </div>
        <div className="flex items-center gap-4 mt-1 text-xs text-gray-500">
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {lastSeenText}
          </span>
          {probe.latency_avg !== undefined && (
            <span>Latency: {probe.latency_avg.toFixed(0)}ms</span>
          )}
          {probe.error_rate !== undefined && (
            <span className={probe.error_rate > 5 ? 'text-red-500' : ''}>
              Errors: {probe.error_rate.toFixed(1)}%
            </span>
          )}
        </div>
      </div>
      
      {probe.status === 'active' && (
        <div className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-green-500" />
        </div>
      )}
    </div>
  );
}

interface StatusBadgeProps {
  count: number;
  label: string;
  variant: 'success' | 'secondary' | 'destructive' | 'warning';
}

function StatusBadge({ count, label, variant }: StatusBadgeProps) {
  const colors = {
    success: 'bg-green-100 dark:bg-green-900/20 text-green-700 dark:text-green-400',
    secondary: 'bg-gray-100 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400',
    destructive: 'bg-red-100 dark:bg-red-900/20 text-red-700 dark:text-red-400',
    warning: 'bg-yellow-100 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-400',
  };

  return (
    <div className={cn('inline-flex items-center gap-2 rounded-full px-3 py-1', colors[variant])}>
      <span className="font-semibold">{count}</span>
      <span className="text-sm">{label}</span>
    </div>
  );
}

function formatTimeSince(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ago`;
  if (hours > 0) return `${hours}h ago`;
  if (minutes > 0) return `${minutes}m ago`;
  if (seconds > 30) return `${seconds}s ago`;
  return 'just now';
}

// Compact version for dashboard
export function CompactProbeStatus({ token }: { token: string }) {
  const [activeCount, setActiveCount] = useState(0);
  const { isConnected } = useLiveProbes(token);

  return (
    <div className="inline-flex items-center gap-2 rounded-full bg-blue-100 dark:bg-blue-900/20 px-3 py-1">
      <Wifi className={cn('h-4 w-4', isConnected ? 'text-blue-500' : 'text-gray-500')} />
      <span className="text-sm font-medium text-blue-700 dark:text-blue-400">
        {activeCount} Active Probes
      </span>
      {isConnected && (
        <div className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-blue-500" />
        </div>
      )}
    </div>
  );
}
