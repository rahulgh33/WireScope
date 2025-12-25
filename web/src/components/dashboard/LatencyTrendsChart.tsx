import { useMemo } from 'react';
import type { AlignedData } from 'uplot';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { UPlotChart } from '@/components/charts/UPlotChart';
import { Skeleton } from '@/components/ui/Skeleton';
import { useLatencyTimeSeries } from '@/hooks/useDashboard';

export function LatencyTrendsChart() {
  const { data, isLoading, error } = useLatencyTimeSeries();

  const chartData: AlignedData | null = useMemo(() => {
    if (!data?.data || data.data.length === 0) return null;

    const timestamps = data.data.map((point) => new Date(point.timestamp).getTime() / 1000);
    const values = data.data.map((point) => point.value);

    return [timestamps, values];
  }, [data]);

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Latency Trends (P95)</CardTitle>
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
          <CardTitle>Latency Trends (P95)</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (!chartData || chartData[0].length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Latency Trends (P95)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            No data available
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Latency Trends (P95)</CardTitle>
      </CardHeader>
      <CardContent>
        <UPlotChart
          data={chartData}
          options={{
            width: 800,
            height: 300,
            series: [
              {},
              {
                label: 'Latency (ms)',
                stroke: 'rgb(59, 130, 246)',
                width: 2,
                fill: 'rgba(59, 130, 246, 0.1)',
              },
            ],
            axes: [
              {
                grid: { show: true, stroke: 'rgba(156, 163, 175, 0.2)' },
              },
              {
                grid: { show: true, stroke: 'rgba(156, 163, 175, 0.2)' },
                label: 'Latency (ms)',
              },
            ],
          }}
        />
      </CardContent>
    </Card>
  );
}
