import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { useUIStore } from '@/stores/uiStore';

export function useDashboardOverview() {
  const { selectedTimeRange, selectedClients, selectedTargets } = useUIStore();

  return useQuery({
    queryKey: ['metrics', 'overview', selectedTimeRange, selectedClients, selectedTargets],
    queryFn: () =>
      apiClient.getDashboardOverview({
        time_range: selectedTimeRange,
        clients: selectedClients.length > 0 ? selectedClients.join(',') : undefined,
        targets: selectedTargets.length > 0 ? selectedTargets.join(',') : undefined,
      }),
    staleTime: 10000, // 10 seconds
    refetchInterval: 30000, // Refetch every 30 seconds
  });
}

export function useLatencyTimeSeries() {
  const { selectedTimeRange, selectedClients, selectedTargets } = useUIStore();

  return useQuery({
    queryKey: ['metrics', 'timeseries', 'latency_p95', selectedTimeRange, selectedClients, selectedTargets],
    queryFn: async () => {
      const timeRange = parseTimeRange(selectedTimeRange);
      return apiClient.getTimeSeries({
        metric: 'latency_p95',
        start: timeRange.start,
        end: timeRange.end,
        interval: getOptimalInterval(selectedTimeRange),
        client_id: selectedClients[0], // TODO: Support multiple clients
        target: selectedTargets[0], // TODO: Support multiple targets
      });
    },
    staleTime: 10000,
    refetchInterval: 30000,
  });
}

function parseTimeRange(range: string): { start: string; end: string } {
  const end = new Date();
  const start = new Date();

  switch (range) {
    case '1h':
      start.setHours(start.getHours() - 1);
      break;
    case '24h':
      start.setHours(start.getHours() - 24);
      break;
    case '7d':
      start.setDate(start.getDate() - 7);
      break;
    case '30d':
      start.setDate(start.getDate() - 30);
      break;
    default:
      start.setHours(start.getHours() - 24);
  }

  return {
    start: start.toISOString(),
    end: end.toISOString(),
  };
}

function getOptimalInterval(timeRange: string): string {
  switch (timeRange) {
    case '1h':
      return '1m';
    case '24h':
      return '5m';
    case '7d':
      return '1h';
    case '30d':
      return '6h';
    default:
      return '5m';
  }
}
