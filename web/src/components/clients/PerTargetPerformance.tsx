import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { useClientPerformance } from '@/hooks/useClients';
import { formatDuration, getStatusColor } from '@/lib/utils';

interface PerTargetPerformanceProps {
  clientId: string;
}

export function PerTargetPerformance({ clientId }: PerTargetPerformanceProps) {
  const { data, isLoading, error } = useClientPerformance({
    clientId,
  });

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Per-Target Performance</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            Error loading performance data
          </div>
        </CardContent>
      </Card>
    );
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Per-Target Performance</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  const targets = data?.targets || [];

  if (targets.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Per-Target Performance</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            No target data available
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Per-Target Performance</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                  Target
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Latency (P95)
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Latency (P50)
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Error Rate
                </th>
                <th className="text-center py-3 px-4 text-sm font-medium text-muted-foreground">
                  Status
                </th>
              </tr>
            </thead>
            <tbody>
              {targets.map((target, index) => {
                const statusColor = getStatusColor(target.status);
                return (
                  <tr
                    key={index}
                    className="border-b border-border last:border-0 hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4">
                      <span className="font-medium">{target.target}</span>
                    </td>
                    <td className="text-right py-3 px-4">
                      <span
                        className={
                          target.avg_latency_p95 > 500
                            ? 'text-red-600 font-medium'
                            : target.avg_latency_p95 > 200
                            ? 'text-yellow-600 font-medium'
                            : 'text-green-600 font-medium'
                        }
                      >
                        {formatDuration(target.avg_latency_p95)}
                      </span>
                    </td>
                    <td className="text-right py-3 px-4">
                      {formatDuration(target.avg_latency_p50)}
                    </td>
                    <td className="text-right py-3 px-4">
                      <span
                        className={
                          target.error_rate > 0.05
                            ? 'text-red-600 font-medium'
                            : target.error_rate > 0.01
                            ? 'text-yellow-600 font-medium'
                            : 'text-green-600 font-medium'
                        }
                      >
                        {(target.error_rate * 100).toFixed(2)}%
                      </span>
                    </td>
                    <td className="text-center py-3 px-4">
                      <div className="inline-flex items-center gap-1.5">
                        <div className={`w-2 h-2 rounded-full ${statusColor.bg}`} />
                        <span className={`text-xs font-medium ${statusColor.text}`}>
                          {target.status}
                        </span>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}
