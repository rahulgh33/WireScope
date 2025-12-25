import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';

export interface UseClientsParams {
  search?: string;
  sort?: 'latency_desc' | 'latency_asc' | 'last_seen_desc' | 'name_asc';
  limit?: number;
  cursor?: string;
}

export function useClients(params?: UseClientsParams) {
  return useQuery({
    queryKey: ['clients', params],
    queryFn: () => apiClient.getClients(params),
    staleTime: 10000,
    refetchInterval: 30000,
  });
}

export function useClientDetail(clientId: string) {
  return useQuery({
    queryKey: ['clients', clientId],
    queryFn: () => apiClient.getClientDetail(clientId),
    staleTime: 10000,
    refetchInterval: 30000,
    enabled: !!clientId,
  });
}

export function useClientPerformance(params: {
  clientId: string;
  start?: string;
  end?: string;
  targets?: string[];
}) {
  const now = new Date().toISOString();
  const dayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
  
  return useQuery({
    queryKey: ['clients', params.clientId, 'performance', params],
    queryFn: () =>
      apiClient.getClientPerformance(
        params.clientId,
        params.start || dayAgo,
        params.end || now,
        params.targets?.join(',')
      ),
    staleTime: 10000,
    enabled: !!params.clientId,
  });
}
