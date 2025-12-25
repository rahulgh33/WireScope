// API Client for Network QoE Telemetry Platform

import type {
  DashboardOverview,
  DashboardParams,
  TimeSeriesResponse,
  TimeSeriesParams,
  ClientsResponse,
  ClientsParams,
  ClientDetail,
  ClientPerformance,
  TargetsResponse,
  TargetDetail,
  DiagnosticsResponse,
  DiagnosticsParams,
  DiagnosticsTrends,
  HealthResponse,
  AdminStats,
  CleanupResponse,
  LoginRequest,
  LoginResponse,
  User,
  RefreshResponse,
  APIError,
} from '@/types/api';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

class APIClient {
  private csrfToken: string | null = null;

  setCSRFToken(token: string) {
    this.csrfToken = token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    };

    // Add CSRF token for unsafe methods
    if (this.csrfToken && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method || 'GET')) {
      headers['X-CSRF-Token'] = this.csrfToken;
    }

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
      credentials: 'include', // Include cookies
    });

    // Store request ID for error tracking
    const requestId = response.headers.get('X-Request-ID');

    if (!response.ok) {
      const errorData: APIError = await response.json().catch(() => ({
        error: {
          code: 'UNKNOWN_ERROR',
          message: 'An unexpected error occurred',
          request_id: requestId || 'unknown',
          timestamp: new Date().toISOString(),
        },
      }));

      throw new Error(JSON.stringify(errorData));
    }

    return response.json();
  }

  // Dashboard & Overview
  async getDashboardOverview(params?: DashboardParams): Promise<DashboardOverview> {
    const query = new URLSearchParams(params as Record<string, string>);
    return this.request(`/dashboard/overview?${query}`);
  }

  async getTimeSeries(params: TimeSeriesParams): Promise<TimeSeriesResponse> {
    const query = new URLSearchParams(params as unknown as Record<string, string>);
    return this.request(`/dashboard/timeseries?${query}`);
  }

  // Client Management
  async getClients(params?: ClientsParams): Promise<ClientsResponse> {
    const query = new URLSearchParams(params as Record<string, string>);
    return this.request(`/clients?${query}`);
  }

  async getClientDetail(clientId: string): Promise<ClientDetail> {
    return this.request(`/clients/${encodeURIComponent(clientId)}`);
  }

  async getClientPerformance(
    clientId: string,
    start: string,
    end: string,
    targets?: string
  ): Promise<ClientPerformance> {
    const query = new URLSearchParams({ start, end });
    if (targets) query.set('targets', targets);
    return this.request(`/clients/${encodeURIComponent(clientId)}/performance?${query}`);
  }

  // Target Management
  async getTargets(sort?: string, health?: string): Promise<TargetsResponse> {
    const query = new URLSearchParams();
    if (sort) query.set('sort', sort);
    if (health) query.set('health', health);
    return this.request(`/targets?${query}`);
  }

  async getTargetDetail(target: string): Promise<TargetDetail> {
    return this.request(`/targets/${encodeURIComponent(target)}`);
  }

  // Diagnostics
  async getDiagnostics(params?: DiagnosticsParams): Promise<DiagnosticsResponse> {
    const query = new URLSearchParams(params as Record<string, string>);
    return this.request(`/diagnostics?${query}`);
  }

  async getDiagnosticsTrends(timeRange?: string): Promise<DiagnosticsTrends> {
    const query = new URLSearchParams();
    if (timeRange) query.set('time_range', timeRange);
    return this.request(`/diagnostics/trends?${query}`);
  }

  // Admin
  async getHealth(): Promise<HealthResponse> {
    return this.request('/admin/health');
  }

  async getAdminStats(): Promise<AdminStats> {
    return this.request('/admin/stats');
  }

  async runCleanup(olderThan: string): Promise<CleanupResponse> {
    return this.request('/admin/maintenance/cleanup', {
      method: 'POST',
      body: JSON.stringify({ older_than: olderThan }),
    });
  }

  // Authentication
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
    this.csrfToken = response.csrf_token;
    return response;
  }

  async logout(): Promise<void> {
    await this.request('/auth/logout', {
      method: 'POST',
    });
    this.csrfToken = null;
  }

  async getCurrentUser(): Promise<User> {
    return this.request<{ user: User }>('/auth/me').then(res => res.user);
  }

  async refreshToken(): Promise<void> {
    const response = await this.request<RefreshResponse>('/auth/refresh', {
      method: 'POST',
    });
    this.csrfToken = response.csrf_token;
  }

  // WebSocket token
  async getWSToken(): Promise<{ token: string; expires_in: number }> {
    return this.request('/auth/ws-token');
  }
}

export const apiClient = new APIClient();
export default apiClient;
