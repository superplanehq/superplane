import { useEffect } from 'react';
import useWebSocket from 'react-use-websocket';
import { EventMap, ServerEvent } from '../../types/events';
import { handleStageAdded } from "./stage_added";
import { handleStageUpdated } from "./stage_updated";
import { handleEventSourceAdded } from "./event_source_added";
import { handleCanvasUpdated } from "./canvas_updated";
import { handleStageEventCreated } from "./stage_event_created";
import { handleStageEventApproved } from "./stage_event_approved";
import { useCanvasStore } from "../canvasStore";

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events using a single WebSocket connection
 */
export function useSetupEventHandlers(canvasId: string): void {
  const store = useCanvasStore();
  const setWebSocketConnectionStatus = useCanvasStore((state) => state.updateWebSocketConnectionStatus);
  
  const { lastJsonMessage, readyState } = useWebSocket<ServerEvent>(
    `${SOCKET_SERVER_URL}${canvasId}`,
    {
      shouldReconnect: () => true,
      reconnectAttempts: 10,
      heartbeat: false,
      onMessage: (event) => {
        console.log('WebSocket message:', event.data);
      },
      reconnectInterval: 3000,
      share: true,
      onError: (error) => {
        console.error('WebSocket error:', error);
      },
      onClose: (closeEvent) => {
        console.log('WebSocket closed:', closeEvent);
      },
    }
  );

  // Update connection status in the store
  useEffect(() => {
    setWebSocketConnectionStatus(readyState);
  }, [readyState, setWebSocketConnectionStatus]);

  // Handle incoming messages
  useEffect(() => {
    if (!lastJsonMessage) return;

    const { event, payload } = lastJsonMessage;
    
    switch (event) {
      case 'stage_added':
        handleStageAdded(payload as EventMap['stage_added'], store);
        break;
      case 'stage_updated':
        handleStageUpdated(payload as EventMap['stage_updated'], store);
        break;
      case 'event_source_added':
        handleEventSourceAdded(payload as EventMap['event_source_added'], store);
        break;
      case 'canvas_updated':
        handleCanvasUpdated(payload as EventMap['canvas_updated'], store);
        break;
      case 'new_stage_event':
        handleStageEventCreated(payload as EventMap['new_stage_event'], store);
        break;
      case 'stage_event_approved':
        handleStageEventApproved(payload as EventMap['stage_event_approved'], store);
        break;
      default:
        console.warn('Unhandled event type:', event);
    }
  // do not add store to the dependency array, or you will get an infinite loop
  }, [lastJsonMessage]);
}
