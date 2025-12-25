import { Link } from 'react-router-dom';
import { useState } from 'react';
import { Card, CardContent } from '@/components/ui/Card';
import { formatDuration, formatRelativeTime, getStatusColor } from '@/lib/utils';
import type { Client } from '@/types/api';

interface ClientCardProps {
  client: Client;
  onDelete?: (clientId: string) => void;
}

export function ClientCard({ client, onDelete }: ClientCardProps) {
  const [showConfirm, setShowConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const statusColor = getStatusColor(client.status);

  const handleDelete = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    if (!showConfirm) {
      setShowConfirm(true);
      return;
    }

    setIsDeleting(true);
    try {
      const response = await fetch(`/api/v1/clients/${client.id}`, {
        method: 'DELETE',
      });
      
      if (response.ok) {
        onDelete?.(client.id);
      } else {
        console.error('Failed to delete client');
      }
    } catch (error) {
      console.error('Error deleting client:', error);
    } finally {
      setIsDeleting(false);
      setShowConfirm(false);
    }
  };

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

            <div className="flex items-center gap-2 ml-4">
              {onDelete && (
                <button
                  onClick={handleDelete}
                  disabled={isDeleting}
                  className={`px-2 py-1 text-xs rounded transition-colors ${
                    showConfirm
                      ? 'bg-red-600 text-white hover:bg-red-700'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-red-100 dark:hover:bg-red-900'
                  } disabled:opacity-50`}
                  title={showConfirm ? 'Click again to confirm' : 'Delete client'}
                >
                  {isDeleting ? '...' : showConfirm ? 'Confirm' : 'Delete'}
                </button>
              )}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                className="h-5 w-5 text-muted-foreground flex-shrink-0"
              >
                <polyline points="9 18 15 12 9 6" />
              </svg>
            </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}