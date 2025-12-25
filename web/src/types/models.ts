// UI-specific type definitions

export type ThemeMode = 'light' | 'dark';

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error';

export interface UIState {
  sidebarOpen: boolean;
  theme: ThemeMode;
  selectedTimeRange: string;
  selectedClients: string[];
  selectedTargets: string[];
}

export interface AuthState {
  user: {
    id: string;
    username: string;
    role: 'admin' | 'viewer' | 'operator';
  } | null;
  isAuthenticated: boolean;
  csrfToken: string | null;
}

export interface WSConnectionState {
  status: ConnectionStatus;
  lastEventId: string | null;
  lastEventTimestamp: string | null;
  reconnectAttempts: number;
  subscriptions: string[];
}

// Chart data types
export interface ChartDataPoint {
  x: number | string;
  y: number;
  label?: string;
}

export interface ChartSeries {
  name: string;
  data: ChartDataPoint[];
  color?: string;
}

// Filter types
export interface ClientFilter {
  search: string;
  status: ('active' | 'inactive' | 'warning')[];
  sortBy: 'latency' | 'name' | 'last_seen';
  sortOrder: 'asc' | 'desc';
}

export interface DiagnosticFilter {
  types: string[];
  timeRange: string;
  severity: ('critical' | 'warning' | 'info')[];
}

// Pagination
export interface PaginationState {
  cursor: string | null;
  hasMore: boolean;
  isLoading: boolean;
}
