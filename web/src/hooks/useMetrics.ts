import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';

export function useMetrics(params?: {
  timeRange?: string;
  clients?: string[];
  targets?: string[];
}) {
  return useQuery({
    queryKey: ['metrics', params],
    queryFn: () =>
      apiClient.getDashboardOverview({
        time_range: params?.timeRange,
        clients: params?.clients?.join(','),
        targets: params?.targets?.join(','),
      }),
    staleTime: 10000,
  });
}

export function useTimeSeries(params: {
  metric: string;
  start: string;
  end: string;
  interval?: string;
  clientId?: string;
  target?: string;
}) {
  return useQuery({
    queryKey: ['timeseries', params],
    queryFn: () => apiClient.getTimeSeries({
      metric: params.metric,
      start: params.start,
      end: params.end,
      interval: params.interval || '5m',
      client_id: params.clientId,
      target: params.target,
    }),
    staleTime: 10000,
    enabled: !!(params.start && params.end),
  });
}
