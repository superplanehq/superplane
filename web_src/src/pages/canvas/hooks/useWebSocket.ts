// src/hooks/useWebSocketEvent.ts
import { useEffect } from 'react';
import { wsClient } from '@/canvas/lib/websocketClient';
import { EventMap } from '@/canvas/types/events';

export const useWebSocketEvent = <K extends keyof EventMap>(
  event: K,
  handler: (payload: EventMap[K]) => void,
  canvasId: string
) => {
  useEffect(() => {
    wsClient.connect('ws://localhost:8000/ws/'+canvasId);
    wsClient.register(event, handler);

    return () => {
      wsClient.unregister(event, handler);
    };
  }, [event, handler, canvasId]);
};
