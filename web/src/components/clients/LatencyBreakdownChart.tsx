import { useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { useClientPerformance } from '@/hooks/useClients';

interface LatencyBreakdownChartProps {
  clientId: string;
}

export function LatencyBreakdownChart({ clientId }: LatencyBreakdownChartProps) {
  const { data, isLoading, error } = useClientPerformance({
    clientId,
  });

  const chartData = useMemo(() => {
    if (!data?.timeseries) return [];

    return data.timeseries.map((point) => ({
      timestamp: new Date(point.timestamp).getTime(),
      dns: point.breakdown?.dns || 0,
      tcp: point.breakdown?.tcp || 0,
      tls: point.breakdown?.tls || 0,
      ttfb: point.breakdown?.ttfb || 0,
    }));
  }, [data]);

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Latency Breakdown</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            Error loading chart data
          </div>
        </CardContent>
      </Card>
    );
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Latency Breakdown</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Latency Breakdown</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            No data available for the selected time range
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Latency Breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(156, 163, 175, 0.2)" />
            <XAxis
              dataKey="timestamp"
              type="number"
              domain={['dataMin', 'dataMax']}
              tickFormatter={(value) => {
                const date = new Date(value);
                return date.toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                });
              }}
              stroke="#9ca3af"
            />
            <YAxis stroke="#9ca3af" label={{ value: 'Latency (ms)', angle: -90, position: 'insideLeft' }} />
            <Tooltip
              labelFormatter={(value) => new Date(value).toLocaleString()}
              contentStyle={{
                backgroundColor: 'hsl(var(--color-background))',
                border: '1px solid hsl(var(--color-border))',
                borderRadius: '0.5rem',
              }}
            />
            <Legend />
            <Area
              type="monotone"
              dataKey="dns"
              stackId="1"
              stroke="#ef4444"
              fill="#ef4444"
              name="DNS"
            />
            <Area
              type="monotone"
              dataKey="tcp"
              stackId="1"
              stroke="#f97316"
              fill="#f97316"
              name="TCP"
            />
            <Area
              type="monotone"
              dataKey="tls"
              stackId="1"
              stroke="#eab308"
              fill="#eab308"
              name="TLS"
            />
            <Area
              type="monotone"
              dataKey="ttfb"
              stackId="1"
              stroke="#3b82f6"
              fill="#3b82f6"
              name="TTFB"
            />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
