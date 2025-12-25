import { Link } from 'react-router-dom';
import { Card, CardContent } from '@/components/ui/Card';
import { formatDuration } from '@/lib/utils';
import type { Target } from '@/types/api';

interface TargetCardProps {
  target: Target;
}

export function TargetCard({ target }: TargetCardProps) {
  const healthColors = {
    healthy: 'bg-green-500 text-green-600',
    warning: 'bg-yellow-500 text-yellow-600',
    critical: 'bg-red-500 text-red-600',
    degraded: 'bg-orange-500 text-orange-600',
  };

  const healthColor =
    healthColors[target.status as keyof typeof healthColors] ||
    'bg-gray-500 text-gray-600';

  return (
    <Link to={`/targets/${encodeURIComponent(target.target)}`}>
      <Card className="transition-shadow hover:shadow-md">
        <CardContent className="p-4">
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2">
                <h3 className="font-semibold text-lg truncate">{target.target}</h3>
                <div className="flex items-center gap-1.5">
                  <div className={`w-2 h-2 rounded-full ${healthColor.split(' ')[0]}`} />
                  <span className={`text-xs font-medium ${healthColor.split(' ')[1]}`}>
                    {target.status}
                  </span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Active Clients:</span>
                  <span className="ml-2 font-medium">{target.active_clients}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Avg Latency:</span>
                  <span
                    className={`ml-2 font-medium ${
                      target.avg_latency_ms > 500
                        ? 'text-red-600'
                        : target.avg_latency_ms > 200
                        ? 'text-yellow-600'
                        : 'text-green-600'
                    }`}
                  >
                    {formatDuration(target.avg_latency_ms)}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Error Count:</span>
                  <span className="ml-2 font-medium text-red-600">
                    {target.error_count}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Total Requests:</span>
                  <span className="ml-2 font-medium">
                    {target.request_count.toLocaleString()}
                  </span>
                </div>
              </div>
            </div>

            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              className="h-5 w-5 text-muted-foreground flex-shrink-0 ml-4"
            >
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
