import { useParams, Link } from 'react-router-dom';
import { useClientDetail } from '@/hooks/useClients';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { formatDuration, formatRelativeTime, getStatusColor } from '@/lib/utils';
import { LatencyBreakdownChart } from '@/components/clients/LatencyBreakdownChart';
import { PerTargetPerformance } from '@/components/clients/PerTargetPerformance';
import { RecentDiagnosticsPanel } from '@/components/dashboard/RecentDiagnosticsPanel';

export function ClientDetailPage() {
  const { clientId } = useParams<{ clientId: string }>();
  const { data: client, isLoading, error } = useClientDetail(clientId || '');

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <Link
            to="/clients"
            className="text-sm text-muted-foreground hover:text-foreground inline-flex items-center gap-1 mb-4"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              className="h-4 w-4"
            >
              <polyline points="15 18 9 12 15 6" />
            </svg>
            Back to Clients
          </Link>
          <h1 className="text-3xl font-bold">Client Not Found</h1>
        </div>
        <div className="text-center py-12 text-muted-foreground">
          Unable to load client details. Please try again.
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (!client) return null;

  const statusColor = getStatusColor(client.status);
  const summary = client.performance_summary;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <Link
          to="/clients"
          className="text-sm text-muted-foreground hover:text-foreground inline-flex items-center gap-1 mb-4"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            className="h-4 w-4"
          >
            <polyline points="15 18 9 12 15 6" />
          </svg>
          Back to Clients
        </Link>
        <div className="flex items-center gap-3">
          <h1 className="text-3xl font-bold">{client.client_id}</h1>
          <div className="flex items-center gap-1.5">
            <div className={`w-2.5 h-2.5 rounded-full ${statusColor.bg}`} />
            <span className={`text-sm font-medium ${statusColor.text}`}>
              {client.status}
            </span>
          </div>
        </div>
        <p className="text-muted-foreground mt-1">
          Last seen {formatRelativeTime(client.last_seen)}
        </p>
      </div>

      {/* Performance Summary Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Avg Latency (P95)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatDuration(summary.avg_latency_p95)}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Avg Latency (P50)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatDuration(summary.avg_latency_p50 || 0)}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Error Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className={`text-2xl font-bold ${
                summary.error_rate > 0.05
                  ? 'text-red-600'
                  : summary.error_rate > 0.01
                  ? 'text-yellow-600'
                  : 'text-green-600'
              }`}
            >
              {(summary.error_rate * 100).toFixed(2)}%
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Success Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {((summary.success_rate || 0) * 100).toFixed(2)}%
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Latency Breakdown Chart */}
      <LatencyBreakdownChart clientId={clientId || ''} />

      {/* Per-Target Performance */}
      <PerTargetPerformance clientId={clientId || ''} />

      {/* Recent Diagnostics */}
      <RecentDiagnosticsPanel />
    </div>
  );
}
