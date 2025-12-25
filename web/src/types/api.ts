// API Type Definitions for Network QoE Telemetry Platform

export interface TimeRange {
  start: string; // ISO8601
  end: string;   // ISO8601
}

export interface TimeSeriesPoint {
  timestamp: string;
  value: number;
  breakdown?: {
    dns?: number;
    tcp?: number;
    tls?: number;
    ttfb?: number;
  };
}

// Dashboard & Overview
export interface DashboardOverview {
  time_range: TimeRange;
  summary: {
    total_clients: number;
    active_clients: number;
    total_targets: number;
    avg_latency_p95: number;
    avg_throughput_p50: number;
    total_measurements: number;
    success_rate: number;
    error_rate: number;
  };
  trends: {
    latency: TimeSeriesPoint[];
    throughput: TimeSeriesPoint[];
    error_rate: TimeSeriesPoint[];
  };
}

export interface TimeSeriesResponse {
  data: TimeSeriesPoint[];
}

// Client Management
export interface ClientMetrics {
  avg_latency_p95: number;
  avg_latency_p50?: number;
  avg_throughput_p50: number;
  error_rate: number;
  success_rate?: number;
  total_measurements: number;
}

export interface Client {
  id: string;
  client_id: string; // Add client_id for compatibility
  name: string;
  status: 'active' | 'inactive' | 'warning';
  last_seen: string;
  avg_latency_ms: number;
  error_count: number;
  total_requests: number;
  active_targets: number;
}

export interface ClientsResponse {
  clients: Client[];
  total: number;
  next_cursor: string | null;
  has_more: boolean;
}

export interface ClientDetail {
  client_id: string;
  created_at: string;
  last_seen: string;
  status: 'active' | 'inactive' | 'warning';
  performance_summary: ClientMetrics;
  recent_diagnostics: Diagnostic[];
  performance_history: TimeSeriesPoint[];
}

export interface ClientPerformance {
  client_id?: string;
  avg_latency_p95?: number;
  error_rate?: number;
  total_measurements?: number;
  timeseries?: TimeSeriesPoint[];
  targets?: TargetPerformance[];
}

// Target Management
export interface Target {
  target: string;
  active_clients: number;
  avg_latency_ms: number;
  error_count: number;
  request_count: number;
  status: string;
  last_checked: string;
}

export interface TargetsResponse {
  targets: Target[];
}

export interface TargetDetail {
  target: string;
  summary: ClientMetrics;
  per_client_performance: ClientPerformance[];
  common_issues: string[];
}

export interface TargetPerformance {
  target: string;
  latency: number;
  avg_latency_p95: number;
  avg_latency_p50: number;
  error_rate: number;
  status: string;
}

// Diagnostics
export interface Diagnostic {
  id: string;
  timestamp: string;
  client_id: string;
  target: string;
  diagnosis_label: string;
  metrics: Record<string, number>;
  root_cause: string;
  recommendations: string[];
}

export interface DiagnosticsResponse {
  diagnostics: Diagnostic[];
  summary: {
    by_label: Record<string, number>;
    by_severity: Record<string, number>;
  };
}

export interface DiagnosticsTrends {
  trends: Array<{
    date: string;
    label: string;
    count: number;
  }>;
}

// Admin
export interface HealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  components: Record<string, string>;
  uptime: number;
}

export interface AdminStats {
  events_processed: number;
  aggregates_created: number;
  active_connections: number;
  queue_depth: number;
}

export interface CleanupResponse {
  deleted_records: number;
  space_freed_mb: number;
}

// Authentication
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  csrf_token: string;
}

export interface User {
  id: string;
  username: string;
  role: 'admin' | 'viewer' | 'operator';
}

export interface RefreshResponse {
  csrf_token: string;
}

// WebSocket Messages
export interface WSMessage {
  schema_version: '1.0';
  type: 'subscribe' | 'unsubscribe' | 'update' | 'batch_update' | 'error' | 'ack';
  event_id?: string;
  timestamp: string;
  data?: any;
}

export interface WSSubscribeMessage extends WSMessage {
  type: 'subscribe';
  channels: string[];
  last_event_id?: string;
}

export interface WSUpdateMessage extends WSMessage {
  type: 'update';
  event_id: string;
  channel: string;
  data: any;
}

export interface WSBatchUpdate extends WSMessage {
  type: 'batch_update';
  updates: Array<{
    event_id: string;
    channel: string;
    data: any;
  }>;
}

export interface WSError extends WSMessage {
  type: 'error';
  error: {
    code: string;
    message: string;
  };
}

// API Error Response
export interface APIError {
  error: {
    code: string;
    message: string;
    details?: any;
    request_id: string;
    timestamp: string;
  };
}

// Query Parameters
export interface DashboardParams {
  time_range?: string;
  clients?: string;
  targets?: string;
}

export interface TimeSeriesParams {
  metric: string;
  client_id?: string;
  target?: string;
  start: string;
  end: string;
  interval: string;
}

export interface ClientsParams {
  search?: string;
  sort?: 'latency_desc' | 'latency_asc' | 'last_seen_desc' | 'name_asc';
  limit?: number;
  cursor?: string;
}

export interface DiagnosticsParams {
  client_id?: string;
  target?: string;
  label?: string;
  start?: string;
  end?: string;
  limit?: number;
}
