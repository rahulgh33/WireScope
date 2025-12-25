import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';

export interface UseTargetsParams {
  sort?: string;
  health?: 'all' | 'healthy' | 'warning' | 'critical';
}

export function useTargets(params?: UseTargetsParams) {
  return useQuery({
    queryKey: ['targets', params],
    queryFn: () =>
      apiClient.getTargets(
        params?.sort,
        params?.health && params.health !== 'all' ? params.health : undefined
      ),
    staleTime: 10000,
    refetchInterval: 30000,
  });
}

export function useTargetDetail(target: string) {
  return useQuery({
    queryKey: ['targets', target],
    queryFn: () => apiClient.getTargetDetail(target),
    staleTime: 10000,
    refetchInterval: 30000,
    enabled: !!target,
  });
}
