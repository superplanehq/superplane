import { useCallback, useEffect, useRef } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { EventMap } from '@/canvas/types/events';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

export const useWebSocketEvent = <K extends keyof EventMap>(
  event: K,
  handler: (payload: EventMap[K]) => void,
  canvasId: string
) => {
  const socketUrl = `${SOCKET_SERVER_URL}${canvasId}`;
  const handlerRef = useRef(handler);
  
  // Update handler ref when handler changes
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  const { sendMessage, readyState } = useWebSocket(socketUrl, {
    onMessage: (wsEvent) => {
      try {
        const data = JSON.parse(wsEvent.data) as { event: K; payload: EventMap[K] };
        if (data.event === event) {
          handlerRef.current(data.payload);
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    },
    shouldReconnect: () => true, // Always attempt to reconnect
    reconnectAttempts: 10,
    reconnectInterval: 3000,
    onError: (error) => {
      console.error('WebSocket error:', error);
    },
    onClose: (closeEvent) => {
      console.log('WebSocket closed:', closeEvent);
    },
  });

  // Send a message through the WebSocket
  const sendEvent = useCallback((payload: EventMap[K]) => {
    sendMessage(JSON.stringify({ event, payload }));
  }, [event, sendMessage]);

  // Connection status
  const connectionStatus = {
    [ReadyState.CONNECTING]: 'connecting',
    [ReadyState.OPEN]: 'connected',
    [ReadyState.CLOSING]: 'disconnecting',
    [ReadyState.CLOSED]: 'disconnected',
    [ReadyState.UNINSTANTIATED]: 'uninstantiated',
  }[readyState];

  return {
    sendEvent,
    connectionStatus: connectionStatus as 'connecting' | 'connected' | 'disconnected' | 'disconnecting' | 'uninstantiated',
    readyState,
  };
};

export const useWebSocketConnection = (canvasId: string) => {
  const { readyState } = useWebSocket(`${SOCKET_SERVER_URL}${canvasId}`, {
    shouldReconnect: () => true, // Always attempt to reconnect
    reconnectAttempts: 10,
    reconnectInterval: 3000,
    onError: (error) => {
      console.error('WebSocket error:', error);
    },
    onClose: (closeEvent) => {
      console.log('WebSocket closed:', closeEvent);
    },
  });

  const connectionStatus = {
    [ReadyState.CONNECTING]: 'connecting',
    [ReadyState.OPEN]: 'connected',
    [ReadyState.CLOSING]: 'disconnecting',
    [ReadyState.CLOSED]: 'disconnected',
    [ReadyState.UNINSTANTIATED]: 'uninstantiated',
  }[readyState];

  return {
    connectionStatus: connectionStatus as 'connecting' | 'connected' | 'disconnected' | 'disconnecting' | 'uninstantiated',
    readyState,
  };
};
