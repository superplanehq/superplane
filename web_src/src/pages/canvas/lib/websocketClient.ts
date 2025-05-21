// src/lib/websocketClient.ts
import { WebsocketBuilder, Websocket, ConstantBackoff } from 'websocket-ts';
import { ServerEvent, EventMap } from '@/canvas/types/events';

type Handler<K extends keyof EventMap> = (payload: EventMap[K]) => void;

class WebSocketClient {
  private socket: Websocket | null = null;
  private handlers: {
    [K in keyof EventMap]?: Set<Handler<K>>;
  } = {};

  connect(url: string) {
    if (this.socket) return; // Singleton

    this.socket = new WebsocketBuilder(url)
      .withBackoff(new ConstantBackoff(1000))
      .onOpen(() => {
        console.log('[WS] Connected');
      })
      .onClose(() => {
        console.log('[WS] Disconnected');
        this.socket = null;
      })
      .onMessage((_, event) => {
        try {
          const data = JSON.parse(event.data) as ServerEvent;
          const { event: eventType, payload } = data;

          console.log("[WS] Received event: ", eventType, " payload: ", payload);

          const callbacks = this.handlers[eventType];
          if (callbacks) {
            callbacks.forEach((cb) => (cb as Handler<typeof eventType>)(payload));
          }
        } catch (err) {
          console.error('[WS] Invalid message', err);
        }
      })
      .build();

  }

  register<K extends keyof EventMap>(event: K, cb: Handler<K>) {
    if (!this.handlers[event]) {
      this.handlers[event] = new Set<Handler<keyof EventMap>>;
    }
    (this.handlers[event] as Set<Handler<K>>).add(cb);
    console.log("Registered handler for event ", event, " number of handlers: ", this.handlers[event].size);
  }

  unregister<K extends keyof EventMap>(event: K, cb: Handler<K>) {
    this.handlers[event]?.delete(cb);
  }

  send<T extends string, P>(event: T, payload: P) {
    this.socket?.send(JSON.stringify({ event, payload }));
  }
}

export const wsClient = new WebSocketClient();
