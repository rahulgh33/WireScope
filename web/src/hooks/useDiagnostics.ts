// Hook for fetching diagnostics data

import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api';
import type { DiagnosticsResponse, DiagnosticsParams } from '@/types/api';

export function useDiagnostics(params?: DiagnosticsParams) {
  return useQuery<DiagnosticsResponse>({
    queryKey: ['diagnostics', params],
    queryFn: () => apiClient.getDiagnostics(params),
    refetchInterval: 30000, // Refetch every 30 seconds
  });
}

export function useDiagnosticsTrends(timeRange?: string) {
  return useQuery({
    queryKey: ['diagnostics-trends', timeRange],
    queryFn: () => apiClient.getDiagnosticsTrends(timeRange),
    refetchInterval: 60000, // Refetch every minute
  });
}
