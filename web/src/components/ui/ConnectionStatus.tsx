import type { ConnectionStatus } from '@/hooks/useLiveUpdates';
import { Activity, AlertCircle, WifiOff } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  error?: Error | null;
  className?: string;
  showLabel?: boolean;
}

export function ConnectionStatusIndicator({
  status,
  error,
  className,
  showLabel = true,
}: ConnectionStatusIndicatorProps) {
  const getStatusConfig = () => {
    switch (status) {
      case 'connected':
        return {
          icon: Activity,
          label: 'Live',
          color: 'text-green-500',
          bgColor: 'bg-green-100 dark:bg-green-900/20',
          pulseColor: 'bg-green-500',
        };
      case 'connecting':
        return {
          icon: Activity,
          label: 'Connecting',
          color: 'text-yellow-500',
          bgColor: 'bg-yellow-100 dark:bg-yellow-900/20',
          pulseColor: 'bg-yellow-500',
        };
      case 'error':
        return {
          icon: AlertCircle,
          label: error?.message || 'Error',
          color: 'text-red-500',
          bgColor: 'bg-red-100 dark:bg-red-900/20',
          pulseColor: 'bg-red-500',
        };
      case 'disconnected':
      default:
        return {
          icon: WifiOff,
          label: 'Offline',
          color: 'text-gray-500',
          bgColor: 'bg-gray-100 dark:bg-gray-900/20',
          pulseColor: 'bg-gray-500',
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;

  return (
    <div
      className={cn(
        'inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium',
        config.bgColor,
        className
      )}
      title={status === 'error' && error ? error.message : `Connection: ${status}`}
    >
      <div className="relative flex h-2 w-2 items-center justify-center">
        {status === 'connected' && (
          <span
            className={cn(
              'absolute inline-flex h-full w-full animate-ping rounded-full opacity-75',
              config.pulseColor
            )}
          />
        )}
        <span className={cn('relative inline-flex h-2 w-2 rounded-full', config.pulseColor)} />
      </div>
      {showLabel && <span className={config.color}>{config.label}</span>}
      <Icon className={cn('h-4 w-4', config.color)} />
    </div>
  );
}

// Compact version for use in headers or tight spaces
export function CompactConnectionStatus({
  status,
  error,
}: {
  status: ConnectionStatus;
  error?: Error | null;
}) {
  return (
    <ConnectionStatusIndicator
      status={status}
      error={error}
      showLabel={false}
      className="px-2 py-1"
    />
  );
}

// Large version for prominent display (e.g., settings page)
export function LargeConnectionStatus({
  status,
  error,
  onRetry,
}: {
  status: ConnectionStatus;
  error?: Error | null;
  onRetry?: () => void;
}) {
  const config = (() => {
    switch (status) {
      case 'connected':
        return {
          title: 'Connected',
          description: 'Real-time updates are active',
          color: 'text-green-600 dark:text-green-400',
        };
      case 'connecting':
        return {
          title: 'Connecting',
          description: 'Establishing connection...',
          color: 'text-yellow-600 dark:text-yellow-400',
        };
      case 'error':
        return {
          title: 'Connection Error',
          description: error?.message || 'Failed to connect to server',
          color: 'text-red-600 dark:text-red-400',
        };
      case 'disconnected':
      default:
        return {
          title: 'Disconnected',
          description: 'Real-time updates are unavailable',
          color: 'text-gray-600 dark:text-gray-400',
        };
    }
  })();

  return (
    <div className="flex items-start gap-4 rounded-lg border p-4">
      <ConnectionStatusIndicator status={status} error={error} showLabel={false} />
      <div className="flex-1">
        <h3 className={cn('font-semibold', config.color)}>{config.title}</h3>
        <p className="text-sm text-gray-600 dark:text-gray-400">{config.description}</p>
        {status === 'error' && onRetry && (
          <button
            onClick={onRetry}
            className="mt-2 text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
          >
            Retry Connection
          </button>
        )}
      </div>
    </div>
  );
}
