import { Link } from 'react-router-dom';
import { Card, CardContent } from '@/components/ui/Card';
import { formatDuration, formatRelativeTime, getStatusColor } from '@/lib/utils';
import type { Client } from '@/types/api';

interface ClientCardProps {
  client: Client;
}

export function ClientCard({ client }: ClientCardProps) {
  const statusColor = getStatusColor(client.status);

  return (
    <Link to={`/clients/${client.id}`}>
      <Card className="transition-shadow hover:shadow-md">
        <CardContent className="p-4">
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2">
                <h3 className="font-semibold text-lg truncate">{client.name}</h3>
                <div className="flex items-center gap-1.5">
                  <div className={`w-2 h-2 rounded-full ${statusColor.bg}`} />
                  <span className={`text-xs font-medium ${statusColor.text}`}>
                    {client.status}
                  </span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Avg Latency:</span>
                  <span className="ml-2 font-medium">
                    {formatDuration(client.avg_latency_ms)}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Error Count:</span>
                  <span className="ml-2 font-medium text-red-600">
                    {client.error_count}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Total Requests:</span>
                  <span className="ml-2 font-medium">
                    {client.total_requests.toLocaleString()}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Last Seen:</span>
                  <span className="ml-2 font-medium">
                    {formatRelativeTime(client.last_seen)}
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
