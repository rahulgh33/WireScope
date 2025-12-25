// WebSocket Client for real-time updates

import { QueryClient } from '@tanstack/react-query';
import type { WSMessage, WSSubscribeMessage, WSUpdateMessage, WSBatchUpdate, WSError } from '@/types/api';

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1';

interface WebSocketClientConfig {
  queryClient: QueryClient;
  onConnectionChange?: (status: 'connected' | 'connecting' | 'disconnected' | 'error') => void;
  onError?: (error: Error) => void;
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private queryClient: QueryClient;
  private reconnectAttempts = 0;
  private maxBackoff = 30000; // 30 seconds
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private currentSubscriptions: string[] = [];
  private lastProcessedEventId: string | null = null;
  private wsToken: string | null = null;
  private onConnectionChange?: (status: 'connected' | 'connecting' | 'disconnected' | 'error') => void;
  private onError?: (error: Error) => void;
  private pendingInvalidations = new Set<string>();
  private batchInvalidateTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(config: WebSocketClientConfig) {
    this.queryClient = config.queryClient;
    this.onConnectionChange = config.onConnectionChange;
    this.onError = config.onError;
  }

  async connect(token: string) {
    this.wsToken = token;
    this.connectWebSocket();
  }

  private connectWebSocket() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.onConnectionChange?.('connecting');

    const wsUrl = `${WS_BASE_URL}/ws/metrics?token=${this.wsToken}`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('[WebSocket] Connected');
      this.reconnectAttempts = 0;
      this.onConnectionChange?.('connected');

      // Resubscribe to channels
      if (this.currentSubscriptions.length > 0) {
        this.subscribe(this.currentSubscriptions);
      }

      // Resync critical data
      this.queryClient.invalidateQueries({ queryKey: ['metrics', 'dashboard'] });
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error('[WebSocket] Failed to parse message:', error);
      }
    };

    this.ws.onerror = (error) => {
      console.error('[WebSocket] Error:', error);
      this.onConnectionChange?.('error');
      this.onError?.(new Error('WebSocket connection error'));
    };

    this.ws.onclose = () => {
      console.log('[WebSocket] Disconnected');
      this.onConnectionChange?.('disconnected');
      this.ws = null;
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    // Exponential backoff with jitter
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts) + Math.random() * 1000,
      this.maxBackoff
    );
    this.reconnectAttempts++;

    console.log(`[WebSocket] Reconnecting in ${Math.floor(delay / 1000)}s (attempt ${this.reconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      this.connectWebSocket();
    }, delay);
  }

  private handleMessage(message: WSMessage) {
    switch (message.type) {
      case 'update':
        this.handleUpdate(message as WSUpdateMessage);
        break;
      case 'batch_update':
        this.handleBatchUpdate(message as WSBatchUpdate);
        break;
      case 'error':
        this.handleError(message as WSError);
        break;
      case 'ack':
        console.log('[WebSocket] Subscription acknowledged');
        break;
      default:
        console.warn('[WebSocket] Unknown message type:', message.type);
    }
  }

  private handleUpdate(message: WSUpdateMessage) {
    if (message.event_id) {
      this.lastProcessedEventId = message.event_id;
    }

    // "Light" events: direct cache updates
    if (message.channel === 'dashboard') {
      this.queryClient.setQueryData(['metrics', 'dashboard'], (old: any) => ({
        ...old,
        summary: message.data.summary || old?.summary,
        trends: message.data.trends || old?.trends,
      }));
    } else if (message.channel.startsWith('client:')) {
      // "Heavy" events: batch invalidation
      const clientId = message.channel.split(':')[1];
      this.pendingInvalidations.add(clientId);
      this.scheduleBatchInvalidation();
    }
  }

  private handleBatchUpdate(message: WSBatchUpdate) {
    message.updates.forEach((update) => {
      if (update.event_id) {
        this.lastProcessedEventId = update.event_id;
      }
      
      if (update.channel.startsWith('client:')) {
        const clientId = update.channel.split(':')[1];
        this.pendingInvalidations.add(clientId);
      }
    });
    
    this.scheduleBatchInvalidation();
  }

  private scheduleBatchInvalidation() {
    if (this.batchInvalidateTimer) {
      return; // Already scheduled
    }

    this.batchInvalidateTimer = setTimeout(() => {
      if (this.pendingInvalidations.size > 0) {
        console.log(`[WebSocket] Batch invalidating ${this.pendingInvalidations.size} clients`);
        this.queryClient.invalidateQueries({ queryKey: ['clients'] });
        this.pendingInvalidations.clear();
      }
      this.batchInvalidateTimer = null;
    }, 5000); // 5 seconds
  }

  private handleError(message: WSError) {
    console.error('[WebSocket] Server error:', message.error);
    this.onError?.(new Error(message.error.message));
  }

  subscribe(channels: string[]) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('[WebSocket] Cannot subscribe: not connected');
      return;
    }

    this.currentSubscriptions = [...new Set([...this.currentSubscriptions, ...channels])];

    const message: WSSubscribeMessage = {
      schema_version: '1.0',
      type: 'subscribe',
      channels,
      timestamp: new Date().toISOString(),
      last_event_id: this.lastProcessedEventId || undefined,
    };

    this.ws.send(JSON.stringify(message));
    console.log('[WebSocket] Subscribed to:', channels);
  }

  unsubscribe(channels: string[]) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    this.currentSubscriptions = this.currentSubscriptions.filter(
      (ch) => !channels.includes(ch)
    );

    const message: WSMessage = {
      schema_version: '1.0',
      type: 'unsubscribe',
      timestamp: new Date().toISOString(),
      data: { channels },
    };

    this.ws.send(JSON.stringify(message));
    console.log('[WebSocket] Unsubscribed from:', channels);
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.batchInvalidateTimer) {
      clearTimeout(this.batchInvalidateTimer);
      this.batchInvalidateTimer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.currentSubscriptions = [];
    this.lastProcessedEventId = null;
    this.reconnectAttempts = 0;
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  getStatus(): 'connected' | 'connecting' | 'disconnected' | 'error' {
    if (!this.ws) return 'disconnected';
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting';
      case WebSocket.OPEN:
        return 'connected';
      case WebSocket.CLOSING:
      case WebSocket.CLOSED:
        return 'disconnected';
      default:
        return 'error';
    }
  }
}
