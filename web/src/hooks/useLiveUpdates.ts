import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { WebSocketClient } from '@/lib/websocket';

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error';

interface UseLiveUpdatesOptions {
  enabled?: boolean;
  channels?: string[];
  token: string;
}

/**
 * Hook for managing WebSocket connection and live updates
 * Integrates with React Query to update cached data in real-time
 */
export function useLiveUpdates(options: UseLiveUpdatesOptions) {
  const { enabled = true, channels = ['dashboard'], token } = options;
  const queryClient = useQueryClient();
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected');
  const [error, setError] = useState<Error | null>(null);
  const wsClientRef = useRef<WebSocketClient | null>(null);

  useEffect(() => {
    if (!enabled || !token) {
      return;
    }

    // Create WebSocket client
    const wsClient = new WebSocketClient({
      queryClient,
      onConnectionChange: (status) => {
        setConnectionStatus(status);
        if (status === 'connected') {
          setError(null);
        }
      },
      onError: (err) => {
        setError(err);
      },
    });

    wsClientRef.current = wsClient;

    // Connect to WebSocket
    wsClient.connect(token).then(() => {
      // Subscribe to channels
      if (channels.length > 0) {
        wsClient.subscribe(channels);
      }
    });

    // Cleanup on unmount
    return () => {
      wsClient.disconnect();
      wsClientRef.current = null;
    };
  }, [enabled, token, queryClient, channels.join(',')]);

  // Function to subscribe to additional channels
  const subscribe = (newChannels: string[]) => {
    if (wsClientRef.current && wsClientRef.current.isConnected()) {
      wsClientRef.current.subscribe(newChannels);
    }
  };

  // Function to unsubscribe from channels
  const unsubscribe = (channelsToRemove: string[]) => {
    if (wsClientRef.current) {
      wsClientRef.current.unsubscribe(channelsToRemove);
    }
  };

  return {
    connectionStatus,
    error,
    subscribe,
    unsubscribe,
    isConnected: connectionStatus === 'connected',
  };
}

/**
 * Hook for live dashboard updates
 * Subscribes to the 'dashboard' channel for real-time metrics
 */
export function useLiveDashboard(token: string, enabled = true) {
  return useLiveUpdates({
    enabled,
    channels: ['dashboard'],
    token,
  });
}

/**
 * Hook for live client updates
 * Subscribes to a specific client's channel for real-time updates
 */
export function useLiveClient(clientId: string | undefined, token: string, enabled = true) {
  const channels = clientId ? [`client:${clientId}`] : [];
  
  return useLiveUpdates({
    enabled: enabled && !!clientId,
    channels,
    token,
  });
}

/**
 * Hook for live diagnostics updates
 * Subscribes to the 'diagnostics' channel for real-time diagnosis events
 */
export function useLiveDiagnostics(token: string, enabled = true) {
  return useLiveUpdates({
    enabled,
    channels: ['diagnostics'],
    token,
  });
}

/**
 * Hook for live probe status updates
 * Subscribes to the 'probes' channel for real-time probe status
 */
export function useLiveProbes(token: string, enabled = true) {
  return useLiveUpdates({
    enabled,
    channels: ['probes'],
    token,
  });
}

/**
 * Hook for managing multiple channel subscriptions dynamically
 * Useful for components that need to subscribe to different channels based on user interaction
 */
export function useDynamicChannels(token: string, enabled = true) {
  const [channels, setChannels] = useState<string[]>([]);
  const liveUpdates = useLiveUpdates({
    enabled,
    channels,
    token,
  });

  const addChannel = (channel: string) => {
    setChannels((prev) => {
      if (prev.includes(channel)) return prev;
      return [...prev, channel];
    });
  };

  const removeChannel = (channel: string) => {
    setChannels((prev) => prev.filter((ch) => ch !== channel));
  };

  const replaceChannels = (newChannels: string[]) => {
    setChannels(newChannels);
  };

  return {
    ...liveUpdates,
    channels,
    addChannel,
    removeChannel,
    replaceChannels,
  };
}
