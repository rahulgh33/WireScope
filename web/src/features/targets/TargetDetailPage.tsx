import { useParams, Link } from 'react-router-dom';
import { useTargetDetail } from '@/hooks/useTargets';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { formatDuration } from '@/lib/utils';
import { PerClientPerformance } from '@/components/targets/PerClientPerformance';
import { CommonIssuesPanel } from '@/components/targets/CommonIssuesPanel';

export function TargetDetailPage() {
  const { target: targetParam } = useParams<{ target: string }>();
  const target = targetParam ? decodeURIComponent(targetParam) : '';
  const { data, isLoading, error } = useTargetDetail(target);

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <Link
            to="/targets"
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
            Back to Targets
          </Link>
          <h1 className="text-3xl font-bold">Target Not Found</h1>
        </div>
        <div className="text-center py-12 text-muted-foreground">
          Unable to load target details. Please try again.
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (!data) return null;

  const summary = data.summary;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <Link
          to="/targets"
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
          Back to Targets
        </Link>
        <div className="flex items-center gap-3">
          <h1 className="text-3xl font-bold">{data.target}</h1>
          <div className="flex items-center gap-1.5">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              className="h-5 w-5 text-muted-foreground"
            >
              <circle cx="12" cy="12" r="10" />
              <circle cx="12" cy="12" r="6" />
              <circle cx="12" cy="12" r="2" />
            </svg>
          </div>
        </div>
        <p className="text-muted-foreground mt-1">Target endpoint performance overview</p>
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
            <div
              className={`text-2xl font-bold ${
                summary.avg_latency_p95 > 500
                  ? 'text-red-600'
                  : summary.avg_latency_p95 > 200
                  ? 'text-yellow-600'
                  : 'text-green-600'
              }`}
            >
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
              Total Measurements
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {summary.total_measurements.toLocaleString()}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Per-Client Performance */}
      <PerClientPerformance target={data.target} clients={data.per_client_performance} />

      {/* Common Issues */}
      <CommonIssuesPanel issues={data.common_issues} />
    </div>
  );
}
