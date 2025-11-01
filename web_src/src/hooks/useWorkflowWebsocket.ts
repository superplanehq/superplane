import { useCallback, useEffect, useRef } from 'react';
import useWebSocket from 'react-use-websocket';
import { ServerEvent } from '@/pages/canvas/types/events';

const SOCKET_SERVER_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/`;

export function useWorkflowWebsocket(
  workflowId: string,
  organizationId: string,
  refetchEvents: (nodeId: string) => void,
  refetchExecutions: (nodeId: string) => void,
): void {

  const eventTimeoutsRef = useRef<Map<string, NodeJS.Timeout>>(new Map());
  const executionTimeoutsRef = useRef<Map<string, NodeJS.Timeout>>(new Map());
  const runningNodesRef = useRef<Set<string>>(new Set());
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const enqueueEventRefetch = useCallback((nodeId: string) => {
    const existingTimeout = eventTimeoutsRef.current.get(nodeId);
    if (existingTimeout) {
      clearTimeout(existingTimeout);
    }

    const timeout = setTimeout(() => {
      refetchEvents(nodeId);
      eventTimeoutsRef.current.delete(nodeId);
    }, 500); // 500ms debounce

    eventTimeoutsRef.current.set(nodeId, timeout);
  }, [refetchEvents]);

  const enqueueExecutionRefetch = useCallback((nodeId: string) => {
    const existingTimeout = executionTimeoutsRef.current.get(nodeId);
    if (existingTimeout) {
      clearTimeout(existingTimeout);
    }

    const timeout = setTimeout(() => {
      refetchExecutions(nodeId);
      executionTimeoutsRef.current.delete(nodeId);
    }, 500); // 500ms debounce

    executionTimeoutsRef.current.set(nodeId, timeout);
  }, [refetchExecutions]);

  const startPolling = useCallback(() => {
    if (pollingIntervalRef.current) return; // Already polling

    pollingIntervalRef.current = setInterval(() => {
      runningNodesRef.current.forEach((nodeId) => {
        refetchExecutions(nodeId);
      });
    }, 3000); // Poll every 3 seconds
  }, [refetchExecutions]);

  const stopPolling = useCallback(() => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  const addRunningNode = useCallback((nodeId: string) => {
    runningNodesRef.current.add(nodeId);
    if (runningNodesRef.current.size === 1) {
      startPolling();
    }
  }, [startPolling]);

  const removeRunningNode = useCallback((nodeId: string) => {
    runningNodesRef.current.delete(nodeId);
    if (runningNodesRef.current.size === 0) {
      stopPolling();
    }
  }, [stopPolling]);

  const onMessage = useCallback((event: MessageEvent<unknown>) => {
    try {
      const data = JSON.parse(event.data as string);
      const payload = data.payload; // payload.nodeId

      switch (data.event) {
        case 'event_created':
          if (payload.nodeId) {
            enqueueEventRefetch(payload.nodeId);
          }
          break;
        case 'execution_created':
          if (payload.nodeId) {
            enqueueExecutionRefetch(payload.nodeId);
          }
          break;
        case 'execution_started':
          if (payload.nodeId) {
            addRunningNode(payload.nodeId);
            enqueueExecutionRefetch(payload.nodeId);
          }
          break;
        case 'execution_finished':
          if (payload.nodeId) {
            removeRunningNode(payload.nodeId);
            enqueueExecutionRefetch(payload.nodeId);
          }
          break;
        default:
          break;
      }
    } catch (error) {
      console.error('Error parsing message:', error);
    }
  }, [enqueueEventRefetch, enqueueExecutionRefetch, addRunningNode, removeRunningNode]);

  useWebSocket<ServerEvent>(
    `${SOCKET_SERVER_URL}${workflowId}?organization_id=${organizationId}`,
    {
      shouldReconnect: () => true,
      reconnectAttempts: 10,
      heartbeat: false,
      reconnectInterval: 3000,
      onOpen: () => {},
      onError: () => {},
      onClose: () => {},
      share: false, // Setting share to false to avoid issues with multiple connections
      onMessage: onMessage
    }
  );

  useEffect(() => {
    return () => {
      stopPolling();
      // Clear all pending timeouts
      eventTimeoutsRef.current.forEach((timeout) => clearTimeout(timeout));
      executionTimeoutsRef.current.forEach((timeout) => clearTimeout(timeout));
      eventTimeoutsRef.current.clear();
      executionTimeoutsRef.current.clear();
      runningNodesRef.current.clear();
    };
  }, [stopPolling]);
}
