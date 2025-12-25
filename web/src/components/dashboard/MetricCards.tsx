import { useDashboardOverview } from '@/hooks/useDashboard';
import { MetricCard } from './MetricCard';
import { formatDuration } from '@/lib/utils';

export function MetricCards() {
  const { data, isLoading, error } = useDashboardOverview();

  if (error) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <MetricCard key={i} title="Error loading metrics" value="—" loading={false} />
        ))}
      </div>
    );
  }

  const summary = data?.summary;

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <MetricCard
        title="Active Clients"
        value={summary?.active_clients ?? 0}
        loading={isLoading}
        icon={
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
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
        }
      />

      <MetricCard
        title="Total Targets"
        value={summary?.total_targets ?? 0}
        loading={isLoading}
        icon={
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
            <circle cx="12" cy="12" r="10" />
            <circle cx="12" cy="12" r="6" />
            <circle cx="12" cy="12" r="2" />
          </svg>
        }
      />

      <MetricCard
        title="Avg Latency (P95)"
        value={summary ? formatDuration(summary.avg_latency_p95) : '—'}
        loading={isLoading}
        status={
          summary
            ? summary.avg_latency_p95 < 200
              ? 'success'
              : summary.avg_latency_p95 < 500
              ? 'warning'
              : 'danger'
            : 'default'
        }
        icon={
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
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
          </svg>
        }
      />

      <MetricCard
        title="Error Rate"
        value={summary ? `${(summary.error_rate * 100).toFixed(2)}%` : '—'}
        loading={isLoading}
        status={
          summary
            ? summary.error_rate < 0.01
              ? 'success'
              : summary.error_rate < 0.05
              ? 'warning'
              : 'danger'
            : 'default'
        }
        icon={
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
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
        }
      />
    </div>
  );
}
