import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { formatDuration } from '@/lib/utils';
import type { ClientPerformance } from '@/types/api';

interface PerClientPerformanceProps {
  target: string;
  clients: ClientPerformance[];
}

export function PerClientPerformance({ target, clients }: PerClientPerformanceProps) {
  if (!clients || clients.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Per-Client Performance Comparison</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            No client performance data available
          </div>
        </CardContent>
      </Card>
    );
  }

  // Transform data for chart - take top 10 clients by latency
  const chartData = clients
    .slice(0, 10)
    .map((client) => ({
      name: client.client_id || 'Unknown',
      latency: client.avg_latency_p95 || 0,
      errorRate: (client.error_rate || 0) * 100,
    }))
    .sort((a, b) => b.latency - a.latency);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Per-Client Performance Comparison</CardTitle>
        <p className="text-sm text-muted-foreground mt-1">
          Top 10 clients by latency for {target}
        </p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(156, 163, 175, 0.2)" />
            <XAxis
              dataKey="name"
              stroke="#9ca3af"
              angle={-45}
              textAnchor="end"
              height={80}
            />
            <YAxis stroke="#9ca3af" label={{ value: 'Latency (ms)', angle: -90, position: 'insideLeft' }} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'hsl(var(--color-background))',
                border: '1px solid hsl(var(--color-border))',
                borderRadius: '0.5rem',
              }}
              formatter={(value: number, name: string) => {
                if (name === 'latency') return [formatDuration(value), 'Latency (P95)'];
                if (name === 'errorRate') return [`${value.toFixed(2)}%`, 'Error Rate'];
                return [value, name];
              }}
            />
            <Legend />
            <Bar dataKey="latency" fill="#3b82f6" name="Latency (P95)" />
            <Bar dataKey="errorRate" fill="#ef4444" name="Error Rate (%)" />
          </BarChart>
        </ResponsiveContainer>

        {/* Detailed Table */}
        <div className="mt-6 overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                  Client
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Latency (P95)
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Error Rate
                </th>
                <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                  Measurements
                </th>
              </tr>
            </thead>
            <tbody>
              {clients.map((client, index) => (
                <tr
                  key={index}
                  className="border-b border-border last:border-0 hover:bg-muted/50 transition-colors"
                >
                  <td className="py-3 px-4">
                    <span className="font-medium">{client.client_id || 'Unknown'}</span>
                  </td>
                  <td className="text-right py-3 px-4">
                    <span
                      className={
                        (client.avg_latency_p95 || 0) > 500
                          ? 'text-red-600 font-medium'
                          : (client.avg_latency_p95 || 0) > 200
                          ? 'text-yellow-600 font-medium'
                          : 'text-green-600 font-medium'
                      }
                    >
                      {formatDuration(client.avg_latency_p95 || 0)}
                    </span>
                  </td>
                  <td className="text-right py-3 px-4">
                    <span
                      className={
                        (client.error_rate || 0) > 0.05
                          ? 'text-red-600 font-medium'
                          : (client.error_rate || 0) > 0.01
                          ? 'text-yellow-600 font-medium'
                          : 'text-green-600 font-medium'
                      }
                    >
                      {((client.error_rate || 0) * 100).toFixed(2)}%
                    </span>
                  </td>
                  <td className="text-right py-3 px-4">
                    {(client.total_measurements || 0).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}
