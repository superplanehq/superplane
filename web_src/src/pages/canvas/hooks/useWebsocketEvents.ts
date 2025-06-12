import { useEffect } from 'react';
import useWebSocket from 'react-use-websocket';
import { EventMap, ServerEvent } from '../types/events';
import { useCanvasStore } from "../store/canvasStore";
import { StageWithEventQueue } from '../store/types';
import { SuperplaneStageEvent } from '@/api-client/types.gen';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events using a single WebSocket connection
 */
export function useWebsocketEvents(canvasId: string): void {
  // Get store access methods directly within the hook
  const updateWebSocketConnectionStatus = useCanvasStore((s) => s.updateWebSocketConnectionStatus);
  const stages = useCanvasStore((s) => s.stages);
  const updateStage = useCanvasStore((s) => s.updateStage);
  const addStage = useCanvasStore((s) => s.addStage);
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
    let stageWithNewEvent: StageWithEventQueue | undefined;
    let updatedStage: StageWithEventQueue;
    let approvedEventPayload: EventMap['stage_event_approved'];
    let stageWithApprovedEvent: StageWithEventQueue | undefined;
    let updatedEvents: Array<SuperplaneStageEvent>;
    
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
        stageWithNewEvent = stages.find(s => s.metadata!.id === newEventPayload.stage_id);
        if (stageWithNewEvent) {
          // Add the event to the stage's event queue
          updatedStage = {
            ...stageWithNewEvent,
            queue: [...(stageWithNewEvent.queue || []), newEventPayload]
          };
          updateStage(updatedStage);
        } else {
          console.warn(`Stage not found for new event: ${newEventPayload.stage_id}`);
        }
        break;
      case 'stage_event_approved':
        approvedEventPayload = payload as EventMap['stage_event_approved'];
        stageWithApprovedEvent = stages.find(s => s.metadata!.id === approvedEventPayload.stage_id);
        if (stageWithApprovedEvent) {
          // Update the event status in the stage's event queue
          updatedEvents = stageWithApprovedEvent.queue?.map((eventItem: SuperplaneStageEvent) => 
            eventItem.id === approvedEventPayload.id ? { ...eventItem, approved: true } : eventItem
          ) || [];
          
          updatedStage = {
            ...stageWithApprovedEvent,
            queue: updatedEvents
          };
          updateStage(updatedStage);
        } else {
          console.warn(`Stage not found for approved event: ${approvedEventPayload.stage_id}`);
        }
        break;
      default:
        console.warn('Unhandled event type:', event);
    }


  }, [lastJsonMessage]);
}
