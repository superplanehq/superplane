import { useEffect } from 'react';
import useWebSocket from 'react-use-websocket';
import { EventMap, ServerEvent } from '../types/events';
import { useCanvasStore } from "../store/canvasStore";
import { EventSourceWithEvents } from '../store/types';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events using a single WebSocket connection
 */
export function useWebsocketEvents(canvasId: string): void {
  // Get store access methods directly within the hook
  const updateWebSocketConnectionStatus = useCanvasStore((s) => s.updateWebSocketConnectionStatus);
  const eventSources = useCanvasStore((s) => s.event_sources);
  const updateStage = useCanvasStore((s) => s.updateStage);
  const updateEventSource = useCanvasStore((s) => s.updateEventSource);
  const addStage = useCanvasStore((s) => s.addStage);
  const syncStageEvents = useCanvasStore((s) => s.syncStageEvents);
  const addEventSource = useCanvasStore((s) => s.addEventSource);
  const updateCanvas = useCanvasStore((s) => s.updateCanvas);


  // WebSocket setup
  const { lastJsonMessage, readyState } = useWebSocket<ServerEvent>(
    `${SOCKET_SERVER_URL}${canvasId}`,
    {
      shouldReconnect: () => true,
      reconnectAttempts: 10,
      heartbeat: false,
      reconnectInterval: 3000,
      onOpen: () => console.log('WebSocket connected'),
      onError: (error) => console.error('WebSocket error:', error),
      onClose: (event) => console.log('WebSocket closed:', event),
      share: false, // Setting share to false to avoid issues with multiple connections
    }
  );

  // Update connection status in the store
  useEffect(() => {
    updateWebSocketConnectionStatus(readyState);
  }, [readyState, updateWebSocketConnectionStatus]);

  // Process incoming WebSocket messages
  useEffect(() => {
    if (!lastJsonMessage) return;

    const { event, payload } = lastJsonMessage;

    // Declare variables outside of case statements to avoid lexical declaration errors
    let newEventPayload: EventMap['new_stage_event'];
    let approvedEventPayload: EventMap['stage_event_approved'];
    let executionFinishedPayload: EventMap['execution_finished']
    let executionStartedPayload: EventMap['execution_started']
    let eventSourceWithNewEvent: EventSourceWithEvents | undefined;
    let updatedEventSource: EventSourceWithEvents;
    
    // Route the event to the appropriate handler
    switch (event) {
      case 'stage_added':
        addStage(payload as EventMap['stage_added']);
        break;
      case 'stage_updated':
        updateStage(payload as EventMap['stage_updated']);
        break;
      case 'event_source_added':
        addEventSource(payload as EventMap['event_source_added']);
        break;
      case 'canvas_updated':
        updateCanvas(payload as EventMap['canvas_updated']);
        break;
      case 'new_stage_event':
        // For stage events, we need to get the current stage first
        newEventPayload = payload as EventMap['new_stage_event'];
        eventSourceWithNewEvent = eventSources.find(es => es.metadata!.id === newEventPayload.source_id);

        syncStageEvents(canvasId, newEventPayload.stage_id);

        if (eventSourceWithNewEvent) {
          updatedEventSource = {
            ...eventSourceWithNewEvent,
            events: [...(eventSourceWithNewEvent.events || []), {
              ...newEventPayload,
              createdAt: newEventPayload.timestamp
            }]
          };
          updateEventSource(updatedEventSource);

        } else {
          console.warn(`Event source not found for new event: ${newEventPayload.source_id}`);
        }

        break;
      case 'stage_event_approved':
        approvedEventPayload = payload as EventMap['stage_event_approved'];
        syncStageEvents(canvasId, approvedEventPayload.stage_id);
        break;
      case 'execution_finished':
        executionFinishedPayload = payload as EventMap['execution_finished'];
        syncStageEvents(canvasId, executionFinishedPayload.stage_id);
        break;
      case 'execution_started':
        executionStartedPayload = payload as EventMap['execution_started'];
        syncStageEvents(canvasId, executionStartedPayload.stage_id);
        break;
      default:
        console.warn('Unhandled event type:', event);
    }


  }, [lastJsonMessage, addEventSource, addStage, updateCanvas, updateStage, syncStageEvents]);
}
