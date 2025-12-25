// Utility functions

export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(' ');
}

export function formatNumber(num: number, decimals: number = 2): string {
  return num.toFixed(decimals);
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
}

export function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diff = now - then;

  if (diff < 60000) return 'just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

export function parseTimeRange(range: string): { start: string; end: string } {
  const end = new Date();
  const start = new Date(end);

  switch (range) {
    case '1h':
      start.setHours(end.getHours() - 1);
      break;
    case '24h':
      start.setHours(end.getHours() - 24);
      break;
    case '7d':
      start.setDate(end.getDate() - 7);
      break;
    case '30d':
      start.setDate(end.getDate() - 30);
      break;
    default:
      start.setHours(end.getHours() - 24);
  }

  return {
    start: start.toISOString(),
    end: end.toISOString(),
  };
}

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout>;
  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean;
  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

export function getStatusColor(status: string): { bg: string; text: string } {
  switch (status.toLowerCase()) {
    case 'active':
    case 'healthy':
    case 'good':
      return { bg: 'bg-green-500', text: 'text-green-600' };
    case 'warning':
    case 'slow':
      return { bg: 'bg-yellow-500', text: 'text-yellow-600' };
    case 'critical':
    case 'error':
    case 'unhealthy':
      return { bg: 'bg-red-500', text: 'text-red-600' };
    case 'inactive':
      return { bg: 'bg-gray-500', text: 'text-gray-600' };
    default:
      return { bg: 'bg-gray-500', text: 'text-gray-600' };
  }
}

export function getStatusBgColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'active':
    case 'healthy':
    case 'good':
      return 'bg-green-100';
    case 'warning':
    case 'slow':
      return 'bg-yellow-100';
    case 'critical':
    case 'error':
    case 'unhealthy':
      return 'bg-red-100';
    case 'inactive':
      return 'bg-gray-100';
    default:
      return 'bg-gray-100';
  }
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function range(start: number, end: number, step: number = 1): number[] {
  const result: number[] = [];
  for (let i = start; i < end; i += step) {
    result.push(i);
  }
  return result;
}
