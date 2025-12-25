# UI Architecture & Technology Stack

## Overview

This document defines the architecture, technology stack, and design decisions for the Network QoE Telemetry Platform web interface.

## Technology Stack Selection

### Frontend Framework: **React 18+ with TypeScript**

**Rationale:**
- **Largest ecosystem**: Best selection of charting, data visualization, and real-time libraries
- **Performance**: Virtual DOM and React 18 concurrent features ideal for live data updates
- **TypeScript**: Type safety for complex data structures and API contracts
- **Community**: Extensive documentation, tools, and third-party libraries
- **Hiring**: Easier to find React developers vs Vue/Svelte

**Alternatives Considered:**
- Vue 3: Good, but smaller ecosystem for data visualization
- Svelte: Excellent performance, but less mature ecosystem for enterprise dashboards

### Build Tool: **Vite**

**Rationale:**
- Fast HMR (Hot Module Replacement) for development
- Modern build optimizations
- Native TypeScript support
- Better than CRA (Create React App) in 2025

### UI Framework: **Tailwind CSS + shadcn/ui**

**Rationale:**
- **Tailwind CSS**: Utility-first, highly customizable, no CSS file management
- **shadcn/ui**: Modern, accessible component library built on Radix UI
- **Flexibility**: Easy to customize and extend components
- **Performance**: Only includes CSS that's actually used

**Alternatives Considered:**
- Material-UI: Heavy bundle size, opinionated design
- Ant Design: Good but less modern than shadcn/ui
- Chakra UI: Similar to shadcn/ui but larger bundle

### Charting Library: **uPlot** (primary) + **Recharts** (simple charts)

**Rationale:**
- **uPlot**: Extremely fast canvas-based library for time-series with 2k-100k+ points
- **Recharts**: React-friendly for simple charts (pie, bar, small datasets)
- **Performance**: QoE telemetry can generate thousands of points per chart
- **Pragmatic Split**: Use uPlot for main time-series dashboards, Recharts for summary views

**Performance Warning:**
- Recharts is SVG-based and will lag with >2-5k points per series
- **Backend must enforce**: server-side downsampling, max points per series, interval aggregation
- For charts with >5k points, uPlot or ECharts (canvas) are mandatory

**Alternatives:**
- **ECharts**: Canvas-based, great zoom/brush, heavier bundle (~500KB)
- Chart.js: Good but less performant than uPlot for high-frequency data
- D3.js: Too low-level for most use cases

### State Management: **Zustand** (UI state only)

**Rationale:**
- Minimal boilerplate compared to Redux
- TypeScript-first design
- Excellent performance
- Easy to understand and maintain

**CRITICAL: Clear separation of concerns**
- **React Query**: Owns ALL server state (metrics, lists, client details, diagnostics)
- **Zustand**: Owns ONLY UI state (sidebar open/closed, selected filters, time range, theme, active client)
- **WebSocket**: Patches React Query caches via `queryClient.setQueryData()`, NOT a separate store

**Anti-pattern to avoid:** Three competing stores (Zustand + React Query + WebSocket store) leads to sync bugs.

**Alternatives:**
- Redux Toolkit: More boilerplate, overkill for this project
- Jotai/Recoil: Good but less mature

### Data Fetching: **TanStack Query (React Query)**

**Rationale:**
- Built-in caching, invalidation, and background refetching
- Optimistic updates support
- Excellent for real-time data scenarios
- Reduces boilerplate for API calls

### Real-time Communication: **WebSocket API + EventSource**

**Rationale:**
- WebSocket for bidirectional real-time updates
- EventSource (SSE) as fallback for simpler one-way streams
- Native browser APIs, no heavy libraries needed

### Routing: **React Router v6**

**Rationale:**
- Industry standard for React routing
- Nested routes support
- Data loading integration

## Project Structure

```
web/
├── public/                    # Static assets
│   ├── index.html
│   └── favicon.ico
├── src/
│   ├── components/            # Reusable UI components
│   │   ├── ui/               # shadcn/ui components
│   │   ├── charts/           # Chart wrappers and configs
│   │   ├── layout/           # Layout components (Header, Sidebar, etc.)
│   │   └── shared/           # Shared business components
│   ├── features/              # Feature-based modules
│   │   ├── dashboard/        # Dashboard feature
│   │   ├── clients/          # Client management
│   │   ├── targets/          # Target management
│   │   ├── diagnostics/      # Diagnostics views
│   │   ├── admin/            # Admin panel
│   │   └── auth/             # Authentication
│   ├── lib/                   # Utilities and helpers
│   │   ├── api.ts            # API client
│   │   ├── websocket.ts      # WebSocket client
│   │   └── utils.ts          # Helper functions
│   ├── hooks/                 # Custom React hooks
│   │   ├── useMetrics.ts
│   │   ├── useRealtime.ts
│   │   └── useAuth.ts
│   ├── stores/                # Zustand stores
│   │   ├── authStore.ts
│   │   └── uiStore.ts
│   ├── types/                 # TypeScript type definitions
│   │   ├── api.ts
│   │   └── models.ts
│   ├── App.tsx               # Root component
│   ├── main.tsx              # Entry point
│   └── index.css             # Global styles
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

## REST API Design

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
```
Authorization: Bearer <token>
```

### Time Range Convention

**Consistent approach:** Server accepts both presets AND explicit ranges:
- `time_range=24h|7d|30d` (server expands to start/end)
- `start=ISO8601&end=ISO8601` (explicit timestamps)
- If both provided, explicit timestamps take precedence
- UI presets ("Last 24h", "Last 7d") are convenience—always send one or the other

### Core Endpoints

#### Dashboard & Overview
```
GET /api/v1/metrics/overview
  ?time_range=24h
  &clients=client1,client2
  &targets=api.example.com
Response: {
  time_range: { start, end },
  summary: {
    total_clients: number,
    active_clients: number,
    total_targets: number,
    avg_latency_p95: number,
    avg_throughput_p50: number,
    total_measurements: number,
    success_rate: number,
    error_rate: number
  },
  trends: {
    latency: TimeSeriesPoint[],
    throughput: TimeSeriesPoint[],
    error_rate: TimeSeriesPoint[]
  }
}

GET /api/v1/metrics/timeseries
  ?metric=latency_p95
  &client_id=client1
  &target=api.example.com
  &start=2025-12-24T00:00:00Z
  &end=2025-12-24T23:59:59Z
  &interval=1m
Response: {
  data: [
    { timestamp, value, breakdown: { dns, tcp, tls, ttfb } }
  ]
}
```

#### Client Management
```
GET /api/v1/clients
  ?search=client
  &sort=latency_desc|latency_asc|last_seen_desc|name_asc
  &limit=50
  &cursor=opaque_cursor_string
Response: {
  clients: [
    {
      client_id: string,
      last_seen: timestamp,
      metrics: {
        avg_latency_p95: number,
        avg_throughput_p50: number,
        error_rate: number,
        total_measurements: number
      },
      status: "active" | "inactive" | "warning"
    }
  ],
  total: number,
  next_cursor: string | null,
  has_more: boolean
}

**Note:** Cursor-based pagination (not offset/limit) for stable ordering under churn.
Sort fields are explicitly enumerated (no arbitrary SQL-ish sorts).

GET /api/v1/clients/:client_id
Response: {
  client_id: string,
  created_at: timestamp,
  last_seen: timestamp,
  performance_summary: {...},
  recent_diagnostics: [...],
  performance_history: [...]
}

GET /api/v1/clients/:client_id/performance
  ?start=timestamp
  &end=timestamp
  &targets=target1,target2
Response: {
  timeseries: [...],
  targets: [...]
}
```

#### Target Management
```
GET /api/v1/targets
  ?sort=clients_desc
  &health=all|healthy|warning|critical
Response: {
  targets: [
    {
      target: string,
      client_count: number,
      avg_latency_p95: number,
      error_rate: number,
      health_status: string,
      issues: string[]
    }
  ]
}

GET /api/v1/targets/:target
Response: {
  target: string,
  summary: {...},
  per_client_performance: [...],
  common_issues: [...]
}
```

#### Diagnostics
```
GET /api/v1/diagnostics
  ?client_id=client1
  &target=api.example.com
  &label=DNS-bound
  &start=timestamp
  &end=timestamp
  &limit=100
Response: {
  diagnostics: [
    {
      id: string,
      timestamp: timestamp,
      client_id: string,
      target: string,
      diagnosis_label: string,
      metrics: {...},
      root_cause: string,
      recommendations: string[]
    }
  ],
  summary: {
    by_label: { "DNS-bound": 45, "Server-bound": 23 },
    by_severity: { critical: 5, warning: 63 }
  }
}

GET /api/v1/diagnostics/trends
  ?time_range=7d
Response: {
  trends: [
    { date, label, count }
  ]
}
```

#### Real-time & WebSocket
```
WS /api/v1/ws/metrics
  - Subscribe to real-time metric updates
  - Send: { type: "subscribe", channels: ["dashboard", "client:123"] }
  - Receive: { type: "update", channel, data }

GET /api/v1/events/stream (Server-Sent Events)
  - Stream of system events
  - Format: event: metric_update\ndata: {...}\n\n
```

#### Admin & Configuration
```
GET /api/v1/admin/health
Response: {
  status: "healthy",
  components: {
    database: "healthy",
    nats: "healthy",
    ai_agent: "healthy"
  },
  uptime: number
}

GET /api/v1/admin/stats
Response: {
  events_processed: number,
  aggregates_created: number,
  active_connections: number,
  queue_depth: number
}

POST /api/v1/admin/maintenance/cleanup
  body: { older_than: "30d" }
Response: {
  deleted_records: number,
  space_freed_mb: number
}
```

#### Authentication
```
POST /api/v1/auth/login
  body: { username, password }
  Sets cookies: refresh_token, access_token
Response: {
  user: { id, username, role },
  csrf_token: string
}

POST /api/v1/auth/logout
  Clears cookies
Response: { success: true }

GET /api/v1/auth/me
Response: {
  user: { id, username, role }
}

POST /api/v1/auth/refresh
  Uses refresh_token cookie
  Rotates tokens
Response: {
  csrf_token: string
}
```

**CSRF Protection:**
- All POST/PUT/DELETE requests must include `X-CSRF-Token` header
- Token returned from login/refresh endpoints
- Server validates token matches cookie-based session

## Page Layouts & Wireframes

### 1. Dashboard View (`/`)

```
┌─────────────────────────────────────────────────────────────┐
│ Header: Logo | Search | Time Range Selector | User Menu     │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                    │
│ Sidebar  │  Dashboard Overview                               │
│          │                                                    │
│ • Home   │  ┌─────────┬─────────┬─────────┬─────────┐      │
│ • Clients│  │ Clients │ Targets │ Latency │ Errors  │      │
│ • Targets│  │   145   │   23    │ 245 ms  │  1.2%   │      │
│ • Diag.  │  └─────────┴─────────┴─────────┴─────────┘      │
│ • Admin  │                                                    │
│          │  Latency Trends (P95)                             │
│          │  ┌───────────────────────────────────────┐       │
│          │  │     📊 Line Chart                     │       │
│          │  │  300ms─┐              ╱─╲             │       │
│          │  │  200ms─┤    ╱╲      ╱   ╲            │       │
│          │  │  100ms─┼───╱──╲────╱     ╲───        │       │
│          │  │    0ms─┴─────────────────────────     │       │
│          │  │        12:00   18:00   00:00   06:00  │       │
│          │  └───────────────────────────────────────┘       │
│          │                                                    │
│          │  ┌─────────────────┬─────────────────────┐       │
│          │  │ Top Issues      │ Recent Diagnostics  │       │
│          │  │ • DNS-bound 45% │ • client-001: High  │       │
│          │  │ • Server 30%    │   DNS latency       │       │
│          │  └─────────────────┴─────────────────────┘       │
└──────────┴──────────────────────────────────────────────────┘
```

**Components:**
- `<Header />` - App header with time range selector
- `<Sidebar />` - Navigation menu
- `<MetricCard />` - Summary stat cards
- `<LatencyChart />` - Time-series line chart
- `<IssuesPanel />` - Top issues list
- `<DiagnosticsFeed />` - Recent diagnostics stream

### 2. Clients View (`/clients`)

```
┌─────────────────────────────────────────────────────────────┐
│ Header: Clients | Search: [________] | Sort: [Latency ▼]   │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                    │
│ Sidebar  │  Filters:                                         │
│          │  ☑ Active  ☐ Inactive  ☐ Warning                 │
│          │                                                    │
│          │  ┌────────────────────────────────────────────┐  │
│          │  │ client-001         Status: ● Active        │  │
│          │  │ Latency: 450ms P95  Throughput: 8.5 MB/s  │  │
│          │  │ Error Rate: 2.5%    Last Seen: 2m ago     │  │
│          │  │ [View Details →]                           │  │
│          │  └────────────────────────────────────────────┘  │
│          │                                                    │
│          │  ┌────────────────────────────────────────────┐  │
│          │  │ client-002         Status: ⚠ Warning      │  │
│          │  │ Latency: 380ms P95  Throughput: 7.2 MB/s  │  │
│          │  │ Error Rate: 1.8%    Last Seen: 5m ago     │  │
│          │  │ [View Details →]                           │  │
│          │  └────────────────────────────────────────────┘  │
│          │                                                    │
│          │  [Load More]                                      │
└──────────┴──────────────────────────────────────────────────┘
```

**Components:**
- `<ClientList />` - Virtualized list of clients
- `<ClientCard />` - Individual client summary
- `<FilterPanel />` - Status and performance filters
- `<SearchBar />` - Client search

### 3. Client Detail View (`/clients/:id`)

```
┌─────────────────────────────────────────────────────────────┐
│ ← Back to Clients | client-001                              │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                    │
│ Sidebar  │  Performance Summary (Last 24h)                   │
│          │  ┌──────────┬──────────┬──────────┬─────────┐    │
│          │  │ Avg P95  │ Avg P50  │ Error %  │ Success │    │
│          │  │  450ms   │  280ms   │   2.5%   │  97.5%  │    │
│          │  └──────────┴──────────┴──────────┴─────────┘    │
│          │                                                    │
│          │  Latency Breakdown                                │
│          │  ┌─────────────────────────────────────────┐     │
│          │  │ 📊 Stacked Area Chart                   │     │
│          │  │   DNS (40%) TCP (20%) TLS (15%) TTFB   │     │
│          │  └─────────────────────────────────────────┘     │
│          │                                                    │
│          │  Per-Target Performance                           │
│          │  ┌──────────────────────────────────────┐        │
│          │  │ Target           Latency   Status    │        │
│          │  │ api.example.com  450ms     ⚠ Slow    │        │
│          │  │ cdn.example.com  180ms     ✓ Good    │        │
│          │  └──────────────────────────────────────┘        │
│          │                                                    │
│          │  Recent Diagnostics                               │
│          │  [Diagnostic cards...]                            │
└──────────┴──────────────────────────────────────────────────┘
```

### 4. Diagnostics View (`/diagnostics`)

```
┌─────────────────────────────────────────────────────────────┐
│ Diagnostics | Filters: [All Types ▼] [Last 7 days ▼]       │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                    │
│ Sidebar  │  Issue Distribution                               │
│          │  ┌─────────────────────────────────────┐         │
│          │  │ 📊 Pie Chart                        │         │
│          │  │  DNS-bound: 45%                     │         │
│          │  │  Server-bound: 30%                  │         │
│          │  │  Throughput: 15%                    │         │
│          │  │  Handshake: 10%                     │         │
│          │  └─────────────────────────────────────┘         │
│          │                                                    │
│          │  Timeline                                         │
│          │  ┌────────────────────────────────────────────┐  │
│          │  │ 🔴 12:45 - client-001 @ api.example.com   │  │
│          │  │    DNS-bound: High DNS latency (450ms)    │  │
│          │  │    → Investigate DNS resolver             │  │
│          │  │    [View Details]                         │  │
│          │  ├────────────────────────────────────────────┤  │
│          │  │ 🟡 12:30 - client-002 @ cdn.example.com   │  │
│          │  │    Server-bound: Slow TTFB (280ms)        │  │
│          │  │    [View Details]                         │  │
│          │  └────────────────────────────────────────────┘  │
└──────────┴──────────────────────────────────────────────────┘
```

### 5. Admin Panel (`/admin`)

```
┌─────────────────────────────────────────────────────────────┐
│ Admin Panel                                                  │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                    │
│ Sidebar  │  System Health                                    │
│          │  ┌─────────────────────────────────────┐         │
│ • Health │  │ Component        Status             │         │
│ • Stats  │  │ PostgreSQL       ● Healthy          │         │
│ • Users  │  │ NATS JetStream   ● Healthy          │         │
│ • Config │  │ AI Agent         ● Healthy          │         │
│ • Maint. │  │ Aggregator       ● Healthy (3 ins.) │         │
│          │  └─────────────────────────────────────┘         │
│          │                                                    │
│          │  System Statistics                                │
│          │  ┌──────────────┬──────────────┬─────────┐       │
│          │  │ Events/sec   │ Queue Depth  │ Uptime  │       │
│          │  │    1,234     │      45      │  5d 3h  │       │
│          │  └──────────────┴──────────────┴─────────┘       │
│          │                                                    │
│          │  Database Maintenance                             │
│          │  Last Cleanup: 2 days ago                         │
│          │  Total Records: 125M                              │
│          │  Disk Usage: 45 GB                                │
│          │  [Run Cleanup Now]                                │
└──────────┴──────────────────────────────────────────────────┘
```

## Responsive Design Strategy

### Breakpoints
```css
/* Tailwind CSS default breakpoints */
sm: 640px   /* Small devices */
md: 768px   /* Medium devices */
lg: 1024px  /* Large devices */
xl: 1280px  /* Extra large */
2xl: 1536px /* 2x extra large */
```

**Note:** If you need custom breakpoints (e.g., md: 1024px), explicitly configure them in `tailwind.config.js` to avoid confusion.

### Mobile Adaptations
- **Sidebar**: Collapsible hamburger menu
- **Charts**: Stack vertically, reduce height
- **Tables**: Horizontal scroll or card layout
- **Metric Cards**: Stack in single column
- **Time Range Selector**: Simplified presets

## Authentication & Authorization

### Strategy: Cookie-based Session with JWT

**Security-first approach:**
- **httpOnly, Secure cookies** for refresh/session tokens (mitigates XSS token theft)
- **SameSite=Lax** or **Strict** for CSRF protection
- Short-lived access tokens (15 min), longer refresh tokens (7 days)
- **NEVER store tokens in localStorage** (vulnerable to XSS)

**Flow:**
1. User submits credentials to `POST /api/v1/auth/login`
2. Server validates and sets httpOnly cookies:
   - `refresh_token` (httpOnly, Secure, SameSite=Strict, 7d)
   - `access_token` (httpOnly, Secure, SameSite=Lax, 15m)
3. Client includes cookies automatically on all requests
4. Server validates cookie on each request
5. Token refresh: automatic via `/api/v1/auth/refresh` endpoint
   - On 401 response, call refresh endpoint
   - If refresh fails, redirect to login
6. CSRF protection: CSRF token in response header, include in POST/PUT/DELETE requests

**Token Rotation:**
- Access token expires after 15 minutes
- Refresh token rotates on each use (sliding window)
- Auto-logout after refresh token expires

### User Roles
- **Admin**: Full access to all features
- **Viewer**: Read-only access to dashboards and metrics
- **Operator**: Can view and trigger maintenance tasks

### Protected Routes
```tsx
<Route element={<ProtectedRoute requiredRole="admin" />}>
  <Route path="/admin" element={<AdminPanel />} />
</Route>
```

## Real-time Updates Strategy

### Architecture: WebSocket patches React Query caches

**Golden rule:** WebSocket events update React Query, NOT a separate store.

```typescript
// "Light" events: direct cache updates
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'metric_update') {
    queryClient.setQueryData(['metrics', 'dashboard'], (old) => ({
      ...old,
      summary: msg.data.summary
    }));
  }
};

// "Heavy" events: batch + invalidate periodically
let pendingInvalidations = new Set();
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'client_performance_update') {
    pendingInvalidations.add(msg.client_id);
  }
};

// Every 5-10 seconds, invalidate in batch
setInterval(() => {
  if (pendingInvalidations.size > 0) {
    queryClient.invalidateQueries({ queryKey: ['clients'] });
    pendingInvalidations.clear();
  }
}, 5000);
```

### WebSocket Connection Contract

#### Authentication
```typescript
// Option 1: Short-lived WS token (preferred)
const wsToken = await fetch('/api/v1/auth/ws-token').then(r => r.json());
const ws = new WebSocket(`ws://localhost:8080/api/v1/ws/metrics?token=${wsToken.token}`);

// Option 2: Send auth after connect
ws.onopen = () => {
  ws.send(JSON.stringify({ type: 'auth', token: wsToken.token }));
};
```

**Security:** Never use main JWT in WS query string (logs leak). Use short-lived WS-specific token (5 min TTL).

#### Message Schema (versioned)
```typescript
interface WSMessage {
  schema_version: "1.0";
  type: "subscribe" | "unsubscribe" | "update" | "batch_update" | "error" | "ack";
  event_id?: string;  // For replay/deduplication
  timestamp: string;  // ISO8601
  data: any;
}

// Subscription
{
  schema_version: "1.0",
  type: "subscribe",
  channels: ["dashboard", "client:client-001"],
  last_event_id: "evt_12345"  // For reconnect resync
}

// Single update
{
  schema_version: "1.0",
  type: "update",
  event_id: "evt_12346",
  timestamp: "2025-12-24T12:00:00Z",
  channel: "dashboard",
  data: { summary: { ... } }
}

// Batch update (server throttling)
{
  schema_version: "1.0",
  type: "batch_update",
  timestamp: "2025-12-24T12:00:00Z",
  updates: [
    { event_id: "evt_12347", channel: "client:001", data: {...} },
    { event_id: "evt_12348", channel: "client:002", data: {...} }
  ]
}

// Error
{
  schema_version: "1.0",
  type: "error",
  error: {
    code: "SUBSCRIPTION_FAILED",
    message: "Invalid channel: xyz"
  }
}
```

#### Reconnection Strategy
```typescript
let reconnectAttempts = 0;
const maxBackoff = 30000; // 30 seconds

function connectWebSocket() {
  const ws = new WebSocket(wsUrl);
  
  ws.onopen = () => {
    reconnectAttempts = 0;
    // Resubscribe with last_event_id for replay
    ws.send(JSON.stringify({
      type: 'subscribe',
      channels: currentSubscriptions,
      last_event_id: lastProcessedEventId
    }));
    
    // One-shot REST fetch to resync critical data
    queryClient.invalidateQueries({ queryKey: ['metrics', 'dashboard'] });
  };
  
  ws.onclose = () => {
    // Exponential backoff with jitter
    const delay = Math.min(
      1000 * Math.pow(2, reconnectAttempts) + Math.random() * 1000,
      maxBackoff
    );
    reconnectAttempts++;
    setTimeout(connectWebSocket, delay);
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
    ws.close();
  };
}
```

#### Server-side Backpressure
```
Server rules:
- Max 1 update/second per channel (throttle)
- Batch updates if >10 events/second
- Drop old events if client can't keep up (send dropped count)
```

#### React Query Integration
```typescript
// Background polling as fallback (WebSocket is additive)
const { data } = useQuery({
  queryKey: ['metrics', 'dashboard'],
  queryFn: fetchDashboardMetrics,
  refetchInterval: 30000,  // Poll every 30s as safety net
  staleTime: 10000         // Data fresh for 10s
});

// WebSocket updates mark data as fresh
ws.onmessage = (event) => {
  queryClient.setQueryData(['metrics', 'dashboard'], newData);
};
```

### Server-Sent Events (Fallback)
```typescript
const eventSource = new EventSource('/api/v1/events/stream');

eventSource.addEventListener('metric_update', (event) => {
  const data = JSON.parse(event.data);
  queryClient.setQueryData(['metrics', 'dashboard'], data);
});

eventSource.onerror = () => {
  // Automatic reconnect with Last-Event-ID header
};
```

**Recommendation:** Start with SSE (simpler, auto-reconnect), upgrade to WS only if you need bidirectional.

## Performance Considerations

### Code Splitting
```typescript
const Dashboard = lazy(() => import('./features/dashboard'));
const Clients = lazy(() => import('./features/clients'));
```

### Virtualization
- Use `react-virtual` for large lists (thousands of clients)
- Render only visible items

### Chart Optimization
- Downsample data points for long time ranges
- Use canvas instead of SVG for large datasets
- Implement data aggregation on backend

### Bundle Size
- Target < 200KB initial JS bundle
- Lazy load charts and heavy components
- Tree-shake unused libraries

## API Standards

### Error Response Format
```typescript
interface APIError {
  error: {
    code: string;           // Machine-readable (e.g., "CLIENT_NOT_FOUND")
    message: string;        // Human-readable
    details?: any;          // Additional context
    request_id: string;     // For support/debugging
    timestamp: string;      // ISO8601
  }
}

// Example
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid time range: start must be before end",
    "details": { "field": "start", "value": "2025-12-25T00:00:00Z" },
    "request_id": "req_abc123",
    "timestamp": "2025-12-24T12:00:00Z"
  }
}
```

### HTTP Status Codes
- `200 OK`: Success
- `400 Bad Request`: Validation error
- `401 Unauthorized`: Missing/invalid auth
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource doesn't exist
- `429 Too Many Requests`: Rate limit
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Maintenance/overload

### OpenAPI Integration
```bash
# Generate TypeScript types from OpenAPI spec
npx openapi-typescript http://localhost:8080/api/v1/openapi.json -o src/types/api.ts

# Use generated types
import type { paths } from './types/api';

type DashboardResponse = paths['/api/v1/metrics/overview']['get']['responses']['200']['content']['application/json'];
```

**Benefits:** Zero API drift, type-safe requests, auto-complete for API calls.

## Development Workflow

### Local Development
```bash
cd web
npm install
npm run dev  # Vite dev server on port 3000
```

### Backend Proxy (vite.config.ts)
```typescript
export default {
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
}
```

### Environment Variables
```
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

## Testing Strategy

### Unit Tests
- Jest + React Testing Library
- Test components in isolation
- Mock API calls

### Integration Tests
- Test user workflows
- Mock backend with MSW (Mock Service Worker)

### E2E Tests (Optional)
- Playwright or Cypress
- Test critical paths

## Accessibility (WCAG 2.1 AA)

- **Keyboard Navigation**: All interactive elements accessible via keyboard
- **Screen Reader Support**: ARIA labels and roles
- **Color Contrast**: Minimum 4.5:1 ratio
- **Focus Indicators**: Visible focus states
- **Error Messages**: Clear and descriptive

## Dark Mode Support

- CSS variables for theming
- Toggle in user menu
- Persist preference in localStorage

```css
:root {
  --bg-primary: #ffffff;
  --text-primary: #000000;
}

[data-theme="dark"] {
  --bg-primary: #1a1a1a;
  --text-primary: #ffffff;
}
```

## Production Readiness Checklist

### Error Boundaries
```tsx
// Global error boundary for catastrophic failures
<ErrorBoundary 
  fallback={<ErrorScreen />}
  onError={(error, errorInfo) => {
    // Send to Sentry/similar
    errorReporter.capture(error, errorInfo);
  }}
>
  <App />
</ErrorBoundary>

// Feature-level boundaries
<ErrorBoundary fallback={<ChartError />}>
  <LatencyChart />
</ErrorBoundary>
```

### Observability

**Request IDs:**
- Backend includes `X-Request-ID` header in all responses
- UI displays request ID in error messages
- Logs include request ID for correlation

**Performance Monitoring:**
```typescript
// Track slow API calls
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      onSuccess: (data, query) => {
        if (query.meta.duration > 2000) {
          analytics.track('slow_query', {
            queryKey: query.queryKey,
            duration: query.meta.duration
          });
        }
      }
    }
  }
});
```

**WebSocket Connection Status:**
```tsx
// UI indicator for connection health
<ConnectionStatus 
  status={wsConnected ? 'connected' : 'disconnected'}
  lastUpdate={lastEventTimestamp}
/>
```

### RBAC in UI

**Route Protection:**
```tsx
<Route element={<RequireRole role="admin" />}>
  <Route path="/admin" element={<AdminPanel />} />
</Route>
```

**Conditional UI Elements:**
```tsx
// Hide controls based on role
{user.role === 'admin' && (
  <button onClick={runCleanup}>Run Cleanup</button>
)}

// Disable instead of hide (better UX)
<button 
  disabled={!user.permissions.includes('maintenance:write')}
  title={!user.permissions.includes('maintenance:write') ? 'Requires admin role' : ''}
>
  Run Cleanup
</button>
```

### Design Tokens (Theming)

**CSS Variables Strategy:**
```css
/* Define in globals.css */
:root {
  /* Colors */
  --color-primary: 220 90% 56%;
  --color-background: 0 0% 100%;
  --color-foreground: 222.2 47.4% 11.2%;
  
  /* Spacing */
  --spacing-unit: 4px;
  
  /* Border radius */
  --radius: 0.5rem;
  
  /* Transitions */
  --transition-fast: 150ms;
}

[data-theme="dark"] {
  --color-background: 224 71% 4%;
  --color-foreground: 213 31% 91%;
}
```

**Tailwind Integration:**
```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        background: 'hsl(var(--color-background))',
        foreground: 'hsl(var(--color-foreground))'
      }
    }
  }
}
```

### Analytics & Telemetry
```typescript
// Track user interactions
analytics.track('filter_applied', {
  filter_type: 'client',
  value: selectedClient
});

// Track feature usage
analytics.track('feature_used', {
  feature: 'ai_query',
  context: 'dashboard'
});
```

### Loading States & Skeletons
```tsx
// Better than spinners
{isLoading ? (
  <Skeleton className="h-64 w-full" />
) : (
  <Chart data={data} />
)}
```

## Architecture Decision Summary

### State Management (The "Tight" Architecture)
```
┌─────────────────────────────────────────────┐
│           UI Components                     │
│  (React + TypeScript + Tailwind)            │
└──────────┬─────────────────┬────────────────┘
           │                 │
           │                 │
    ┌──────▼──────┐   ┌──────▼──────────┐
    │  Zustand    │   │  React Query    │
    │  (UI state) │   │ (Server state)  │
    └─────────────┘   └─────────┬───────┘
                                 │
                      ┌──────────▼──────────┐
                      │   WebSocket         │
                      │   (patches cache)   │
                      └─────────────────────┘
```

**Golden Rules:**
1. React Query = ALL server state (metrics, lists, diagnostics)
2. Zustand = ONLY UI state (filters, sidebar, theme, selected items)
3. WebSocket events => `queryClient.setQueryData()` or `invalidateQueries()`
4. Backend guarantees: interval aggregation, max points, throttling
5. Auth: httpOnly cookies + CSRF tokens (NEVER localStorage)
6. Charts: uPlot for time-series, Recharts for simple/summary

## Summary

This architecture provides:
- ✅ **Modern stack**: React 18, TypeScript, Vite, Tailwind, shadcn/ui
- ✅ **Performance**: uPlot for large time-series, code splitting, virtualization
- ✅ **Real-time**: WebSocket with reconnection, backpressure, event replay
- ✅ **Security**: httpOnly cookies, CSRF protection, token rotation
- ✅ **State clarity**: React Query (server), Zustand (UI), WebSocket (patches)
- ✅ **Developer Experience**: OpenAPI types, fast builds, TypeScript safety
- ✅ **Production Ready**: Error boundaries, observability, RBAC, design tokens
- ✅ **Scalable**: Feature-based structure, cursor pagination, explicit API contracts

## Next Steps

### Phase 1: Foundation (Task 31)
1. Set up Vite + React + TypeScript project
2. Install core dependencies (React Query, Zustand, uPlot, shadcn/ui)
3. Configure Tailwind with design tokens
4. Create basic layout (Header, Sidebar, routing)
5. Implement authentication (login, httpOnly cookies, CSRF)
6. Set up API client with OpenAPI types

### Phase 2: Core Features (Tasks 32-36)
7. Dashboard view with real-time metrics
8. Client list and detail views
9. Target management
10. Diagnostics views
11. WebSocket integration
12. Admin panel

### Phase 3: Polish (Tasks 37-38)
13. Error boundaries and loading states
14. Responsive design and mobile optimization
15. Accessibility audit (WCAG 2.1 AA)
16. Dark mode implementation
17. Performance testing (Lighthouse, bundle analysis)
18. E2E tests for critical paths
19. Comprehensive demo
