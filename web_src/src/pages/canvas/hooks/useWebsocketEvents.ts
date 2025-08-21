import { useEffect } from 'react';
import useWebSocket from 'react-use-websocket';
import { EventMap, ServerEvent } from '../types/events';
import { useCanvasStore } from "../store/canvasStore";
import { EventSourceWithEvents } from '../store/types';
import { pollEventSourceUntilNoPending } from '../utils/eventSourcePolling';
import { SuperplaneEventSource, SuperplaneFilterType } from '@/api-client';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events using a single WebSocket connection
 */
export function useWebsocketEvents(canvasId: string, organizationId: string): void {
  // Get store access methods directly within the hook
  const updateWebSocketConnectionStatus = useCanvasStore((s) => s.updateWebSocketConnectionStatus);
  const eventSources = useCanvasStore((s) => s.eventSources);
  const updateStage = useCanvasStore((s) => s.updateStage);
  const addStage = useCanvasStore((s) => s.addStage);
  const addConnectionGroup = useCanvasStore((s) => s.addConnectionGroup);
  const syncStageEvents = useCanvasStore((s) => s.syncStageEvents);
  const syncEventSourceEvents = useCanvasStore((s) => s.syncEventSourceEvents);
  const addEventSource = useCanvasStore((s) => s.addEventSource);
  const updateCanvas = useCanvasStore((s) => s.updateCanvas);
  const syncToReactFlow = useCanvasStore((s) => s.syncToReactFlow);
  const lockedNodes = useCanvasStore((s) => s.lockedNodes);

  // WebSocket setup
  const { lastJsonMessage, readyState } = useWebSocket<ServerEvent>(
    `${SOCKET_SERVER_URL}${canvasId}?organization_id=${organizationId}`,
    {
      shouldReconnect: () => true,
      reconnectAttempts: 10,
      heartbeat: false,
      reconnectInterval: 3000,
      onOpen: () => {},
      onError: () => {},
      onClose: () => {},
      share: false, // Setting share to false to avoid issues with multiple connections
    }
  );

  const syncReactFlowWithTimeout = (autoLayout: boolean) => {
    setTimeout(() => {
      syncToReactFlow({ autoLayout });
    }, 100);
  }

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
    let eventSource: SuperplaneEventSource;
    
    // Route the event to the appropriate handler
    switch (event) {
      case 'stage_added':
        addStage(payload as EventMap['stage_added'], false);
        syncReactFlowWithTimeout(lockedNodes);
        break;
      case 'connection_group_added':
        addConnectionGroup(payload as EventMap['connection_group_added']);
        syncReactFlowWithTimeout(lockedNodes);
        break;
      case 'stage_updated':
        updateStage(payload as EventMap['stage_updated']);
        syncReactFlowWithTimeout(lockedNodes);
        break;
      case 'event_source_added':
        eventSource = payload;
        eventSource.spec?.events?.forEach(event => {
          event.filters?.forEach(filter => {
            if (typeof filter.type === 'number') {
              const filterTypes = ['FILTER_TYPE_UNKNOWN', 'FILTER_TYPE_DATA', 'FILTER_TYPE_HEADER'];
              filter.type = filterTypes[filter.type] as SuperplaneFilterType;
            }
          });
        });

        addEventSource(eventSource as EventSourceWithEvents);
        syncReactFlowWithTimeout(lockedNodes);
        break;
      case 'canvas_updated':
        updateCanvas(payload as EventMap['canvas_updated']);
        break;

      case 'event_created':
        if (payload.source_type === 'event-source') {
          eventSourceWithNewEvent = eventSources.find(es => es.metadata!.id === payload.source_id);
          pollEventSourceUntilNoPending(canvasId, eventSourceWithNewEvent?.metadata?.id || '');
        }
        break;

      case 'new_stage_event':
        newEventPayload = payload as EventMap['new_stage_event'];
        eventSourceWithNewEvent = eventSources.find(es => es.metadata!.id === newEventPayload.source_id);

        syncStageEvents(canvasId, newEventPayload.stage_id);

        setTimeout(() => {
          syncStageEvents(canvasId, newEventPayload.stage_id);
        }, 3000);

        if (eventSourceWithNewEvent) {
          syncEventSourceEvents(canvasId, newEventPayload.source_id);
        } else {
          console.error('Event source with new event not found:', newEventPayload.source_id)
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
    }


  }, [lastJsonMessage, addEventSource, addStage, updateCanvas, updateStage, syncStageEvents]);
}
