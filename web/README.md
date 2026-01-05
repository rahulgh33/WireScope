# WireScope - Web UI

Modern React-based web interface for WireScope.

## Tech Stack

- **React 18** + **TypeScript** + **Vite**
- **Tailwind CSS** with design tokens
- **TanStack Query** (server state) + **Zustand** (UI state)
- **React Router v6** (routing)
- **uPlot** (time-series charts) + **Recharts** (summaries)

## Quick Start

```bash
npm install
npm run dev    # Dev server on http://localhost:3000
npm run build  # Production build
```

## Architecture

- **State**: React Query = server, Zustand = UI only, WebSocket patches caches
- **Auth**: Cookie-based with CSRF (no localStorage)
- **Real-time**: WebSocket with reconnection, throttling, batch invalidation

## Environment

Create `.env`:
```
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/api/v1
```

See [docs/UI_ARCHITECTURE.md](../docs/UI_ARCHITECTURE.md) for full details.
