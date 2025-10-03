import { useEffect } from 'react';
import useWebSocket from 'react-use-websocket';
import { useQueryClient } from '@tanstack/react-query';
import { EventMap, ServerEvent } from '../types/events';
import { useCanvasStore } from "../store/canvasStore";
import { ConnectionGroupWithEvents, EventSourceWithEvents, Stage } from '../store/types';
import { pollConnectionGroupUntilNoPending, pollEventSourceUntilNoPending, pollStageUntilNoPending } from '../utils/eventSourcePolling';
import { stageUpdateQueue } from '../utils/stageUpdateQueue';
import { SuperplaneEventSource } from '@/api-client';
import { canvasKeys, useAddAlert } from '@/hooks/useCanvasData';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events using a single WebSocket connection
 */
export function useWebsocketEvents(canvasId: string, organizationId: string): void {
  // Get query client for cache invalidation
  const queryClient = useQueryClient();
  
  // Get store access methods directly within the hook
  const updateWebSocketConnectionStatus = useCanvasStore((s) => s.updateWebSocketConnectionStatus);
  const eventSources = useCanvasStore((s) => s.eventSources);
  const connectionGroups = useCanvasStore((s) => s.connectionGroups);
  const stages = useCanvasStore((s) => s.stages);
  const updateStage = useCanvasStore((s) => s.updateStage);
  const addStage = useCanvasStore((s) => s.addStage);
  const addConnectionGroup = useCanvasStore((s) => s.addConnectionGroup);
  const syncStageEvents = useCanvasStore((s) => s.syncStageEvents);
  const syncStageExecutions = useCanvasStore((s) => s.syncStageExecutions);
  const syncEventSourceEvents = useCanvasStore((s) => s.syncEventSourceEvents);
  const addEventSource = useCanvasStore((s) => s.addEventSource);
  const updateEventSource = useCanvasStore((s) => s.updateEventSource);
  const updateCanvas = useCanvasStore((s) => s.updateCanvas);
  const syncToReactFlow = useCanvasStore((s) => s.syncToReactFlow);
  const lockedNodes = useCanvasStore((s) => s.lockedNodes);

  const addAlert = useAddAlert(canvasId);

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
    let alertCreatedPayload: EventMap['alert_created'];
    let newEventPayload: EventMap['new_stage_event'];
    let approvedEventPayload: EventMap['stage_event_approved'];
    let discardedEventPayload: EventMap['stage_event_discarded'];
    let executionFinishedPayload: EventMap['execution_finished'];
    let executionStartedPayload: EventMap['execution_started'];
    let executionCancelledPayload: EventMap['execution_cancelled'];
    let eventSourceWithNewEvent: EventSourceWithEvents | undefined;
    let connectionGroupWithNewEvent: ConnectionGroupWithEvents | undefined;
    let stageWithNewEvent: Stage | undefined;
    let eventSource: SuperplaneEventSource;
    
    // Route the event to the appropriate handler
    switch (event) {
      case 'alert_created':
        alertCreatedPayload = payload as EventMap['alert_created'];
        addAlert.mutateAsync(alertCreatedPayload);
        break;
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
        addEventSource(eventSource as EventSourceWithEvents);
        syncReactFlowWithTimeout(lockedNodes);
        break;
      case 'event_source_updated':
        eventSource = payload;
        updateEventSource(eventSource as EventSourceWithEvents);
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

        if (payload.source_type === 'connection-group') {
          connectionGroupWithNewEvent = connectionGroups.find(es => es.metadata!.id === payload.source_id);
          pollConnectionGroupUntilNoPending(canvasId, connectionGroupWithNewEvent?.metadata?.id || '');
        }

        if (payload.source_type === 'stage') {
          stageWithNewEvent = stages.find(es => es.metadata!.id === payload.source_id);
          const stageId = stageWithNewEvent?.metadata?.id || '';
          if (stageId) {
            stageUpdateQueue.enqueue(stageId, () => pollStageUntilNoPending(canvasId, stageId));
            
            // Invalidate stage events query to update the sidebar queue
            queryClient.invalidateQueries({
              queryKey: canvasKeys.stageEvents(canvasId, stageId, ['STATE_PENDING', 'STATE_WAITING'])
            });
          }
        }
        break;

      case 'new_stage_event':
        newEventPayload = payload as EventMap['new_stage_event'];
        eventSourceWithNewEvent = eventSources.find(es => es.metadata!.id === newEventPayload.source_id);

        // Queue immediate sync
        stageUpdateQueue.enqueue(newEventPayload.stage_id, () => syncStageEvents(canvasId, newEventPayload.stage_id));

        // Queue delayed sync
        setTimeout(() => {
          stageUpdateQueue.enqueue(newEventPayload.stage_id, () => syncStageEvents(canvasId, newEventPayload.stage_id));
        }, 3000);

        // Invalidate stage events query to update the sidebar queue
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageEvents(canvasId, newEventPayload.stage_id, ['STATE_PENDING', 'STATE_WAITING'])
        });

        if (eventSourceWithNewEvent) {
          syncEventSourceEvents(canvasId, newEventPayload.source_id);
        } else {
          console.error('Event source with new event not found:', newEventPayload.source_id)
        }

        break;
      case 'stage_event_approved':
        approvedEventPayload = payload as EventMap['stage_event_approved'];
        stageUpdateQueue.enqueue(approvedEventPayload.stage_id, () => syncStageEvents(canvasId, approvedEventPayload.stage_id));
        
        // Invalidate stage events query to update the queue
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageEvents(canvasId, approvedEventPayload.stage_id, ['STATE_PENDING', 'STATE_WAITING'])
        });
        break;
      case 'stage_event_discarded':
        discardedEventPayload = payload as EventMap['stage_event_discarded'];
        stageUpdateQueue.enqueue(discardedEventPayload.stage_id, () => syncStageEvents(canvasId, discardedEventPayload.stage_id));
        
        // Invalidate stage events query to update the queue
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageEvents(canvasId, discardedEventPayload.stage_id, ['STATE_PENDING', 'STATE_WAITING'])
        });
        break;
      case 'execution_finished':
        executionFinishedPayload = payload as EventMap['execution_finished'];
        stageUpdateQueue.enqueue(executionFinishedPayload.stage_id, () => pollStageUntilNoPending(canvasId, executionFinishedPayload.stage_id), 'high');
        stageUpdateQueue.enqueue(executionFinishedPayload.stage_id, () => syncStageEvents(canvasId, executionFinishedPayload.stage_id));
        stageUpdateQueue.enqueue(executionFinishedPayload.stage_id, () => syncStageExecutions(canvasId, executionFinishedPayload.stage_id));
        
        // Invalidate executions query to update the Recent Runs section
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageExecutions(canvasId, executionFinishedPayload.stage_id)
        });
        break;
      case 'execution_started':
        executionStartedPayload = payload as EventMap['execution_started'];
        stageUpdateQueue.enqueue(executionStartedPayload.stage_id, () => pollStageUntilNoPending(canvasId, executionStartedPayload.stage_id), 'high');
        stageUpdateQueue.enqueue(executionStartedPayload.stage_id, () => syncStageEvents(canvasId, executionStartedPayload.stage_id));
        stageUpdateQueue.enqueue(executionStartedPayload.stage_id, () => syncStageExecutions(canvasId, executionStartedPayload.stage_id));
        
        // Invalidate both stage events (to remove processed events from queue) and executions (to show new execution)
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageEvents(canvasId, executionStartedPayload.stage_id, ['STATE_PENDING', 'STATE_WAITING'])
        });
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageExecutions(canvasId, executionStartedPayload.stage_id)
        });
        break;
      case 'execution_cancelled':
        executionCancelledPayload = payload as EventMap['execution_cancelled'];
        stageUpdateQueue.enqueue(executionCancelledPayload.stage_id, () => pollStageUntilNoPending(canvasId, executionCancelledPayload.stage_id), 'high');
        stageUpdateQueue.enqueue(executionCancelledPayload.stage_id, () => syncStageEvents(canvasId, executionCancelledPayload.stage_id));
        stageUpdateQueue.enqueue(executionCancelledPayload.stage_id, () => syncStageExecutions(canvasId, executionCancelledPayload.stage_id));
        
        // Invalidate executions query to update the Recent Runs section
        queryClient.invalidateQueries({
          queryKey: canvasKeys.stageExecutions(canvasId, executionCancelledPayload.stage_id)
        });
        break;
      default:
    }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastJsonMessage, addEventSource, updateEventSource, addStage, updateCanvas, updateStage, syncStageEvents, syncStageExecutions]);
}
