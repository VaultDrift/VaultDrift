import { WebSocketEvent } from '@/types';

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private listeners: Map<string, Set<(event: WebSocketEvent) => void>> = new Map();
  private onConnectCallbacks: Set<() => void> = new Set();
  private onDisconnectCallbacks: Set<() => void> = new Set();

  constructor(token: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.url = `${protocol}//${window.location.host}/ws?token=${token}`;
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
      this.onConnectCallbacks.forEach(cb => cb());
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.handleMessage(data);
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.onDisconnectCallbacks.forEach(cb => cb());
      this.attemptReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    setTimeout(() => this.connect(), delay);
  }

  private handleMessage(data: WebSocketEvent): void {
    const { type } = data;
    const listeners = this.listeners.get(type);
    if (listeners) {
      listeners.forEach(callback => callback(data));
    }

    // Also notify wildcard listeners
    const wildcards = this.listeners.get('*');
    if (wildcards) {
      wildcards.forEach(callback => callback(data));
    }
  }

  subscribe(eventType: string, callback: (event: WebSocketEvent) => void): () => void {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, new Set());
    }
    this.listeners.get(eventType)!.add(callback);

    // Return unsubscribe function
    return () => {
      this.listeners.get(eventType)?.delete(callback);
    };
  }

  onConnect(callback: () => void): () => void {
    this.onConnectCallbacks.add(callback);
    return () => this.onConnectCallbacks.delete(callback);
  }

  onDisconnect(callback: () => void): () => void {
    this.onDisconnectCallbacks.add(callback);
    return () => this.onDisconnectCallbacks.delete(callback);
  }

  send(data: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  subscribeFolder(folderId: string): void {
    this.send({
      type: 'subscribe',
      payload: { folder_id: folderId },
    });
  }

  unsubscribeFolder(folderId: string): void {
    this.send({
      type: 'unsubscribe',
      payload: { folder_id: folderId },
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

let wsClient: WebSocketClient | null = null;

export function getWebSocketClient(_token?: string): WebSocketClient | null {
  if (!wsClient && _token) {
    wsClient = new WebSocketClient(_token);
  }
  return wsClient;
}

export function disconnectWebSocket(): void {
  wsClient?.disconnect();
  wsClient = null;
}
