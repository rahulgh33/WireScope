import { useState, useEffect } from 'react';
import { useDashboardOverview } from '@/hooks/useDashboard';
import { useLiveDashboard } from '@/hooks/useLiveUpdates';
import { ConnectionStatusIndicator } from '@/components/ui/ConnectionStatus';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Activity, TrendingUp, TrendingDown, AlertCircle, CheckCircle } from 'lucide-react';
import { cn } from '@/lib/utils';

interface LiveDashboardProps {
  token: string;
  enableLiveUpdates?: boolean;
}

export function LiveDashboard({ token, enableLiveUpdates = true }: LiveDashboardProps) {
  const { data: overview, isLoading, error } = useDashboardOverview();
  const { connectionStatus, error: wsError, isConnected } = useLiveDashboard(token, enableLiveUpdates);
  
  const [lastUpdateTime, setLastUpdateTime] = useState<Date>(new Date());
  
  // Update last update time when data changes
  useEffect(() => {
    if (overview) {
      setLastUpdateTime(new Date());
    }
  }, [overview]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Activity className="h-8 w-8 animate-spin text-blue-500" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <p className="text-gray-600 dark:text-gray-400">Failed to load dashboard</p>
        </div>
      </div>
    );
  }

  const summary = overview?.summary;

  return (
    <div className="space-y-6">
      {/* Header with connection status */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Live Dashboard</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Last updated: {lastUpdateTime.toLocaleTimeString()}
          </p>
        </div>
        <ConnectionStatusIndicator status={connectionStatus} error={wsError} />
      </div>

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Active Clients"
          value={summary?.active_clients || 0}
          icon={Activity}
          isLive={isConnected}
        />
        
        <MetricCard
          title="Average Latency (P95)"
          value={`${summary?.avg_latency_p95?.toFixed(1) || 0}ms`}
          icon={TrendingUp}
          isLive={isConnected}
        />
        
        <MetricCard
          title="Success Rate"
          value={`${summary?.success_rate?.toFixed(1) || 0}%`}
          icon={CheckCircle}
          isLive={isConnected}
        />
        
        <MetricCard
          title="Error Rate"
          value={`${summary?.error_rate?.toFixed(2) || 0}%`}
          icon={AlertCircle}
          isLive={isConnected}
        />
      </div>

      {/* Live indicator badge */}
      {isConnected && (
        <div className="inline-flex items-center gap-2 rounded-full bg-green-100 dark:bg-green-900/20 px-3 py-1 text-sm">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-green-500" />
          </span>
          <span className="text-green-600 dark:text-green-400 font-medium">Real-time updates active</span>
        </div>
      )}
    </div>
  );
}

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: React.ElementType;
  trend?: number;
  isLive?: boolean;
  trendInverse?: boolean;
}

function MetricCard({ title, value, icon: Icon, trend, isLive, trendInverse }: MetricCardProps) {
  const getTrendColor = () => {
    if (trend === undefined || trend === 0) return 'text-gray-500';
    
    const isPositive = trend > 0;
    const isGood = trendInverse ? !isPositive : isPositive;
    
    return isGood ? 'text-green-500' : 'text-red-500';
  };

  const TrendIcon = trend && trend > 0 ? TrendingUp : TrendingDown;

  return (
    <Card className={cn('relative overflow-hidden', isLive && 'ring-2 ring-blue-500/20')}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {trend !== undefined && trend !== 0 && (
          <div className={cn('flex items-center text-xs', getTrendColor())}>
            <TrendIcon className="h-3 w-3 mr-1" />
            <span>{Math.abs(trend).toFixed(1)}%</span>
          </div>
        )}
      </CardContent>
      {isLive && (
        <div className="absolute top-2 right-2">
          <div className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-blue-500" />
          </div>
        </div>
      )}
    </Card>
  );
}
