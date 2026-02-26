import { useCallback, useEffect, useRef } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient } from "@tanstack/react-query";
import { CanvasesCanvasNodeExecution, CanvasesCanvasEvent, CanvasesCanvasNodeQueueItem } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { canvasKeys } from "./useCanvasData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

type CanvasWebsocketPayload = {
  id?: string;
  canvasId?: string;
};

type CanvasLifecycleEventName = "canvas_updated" | "canvas_deleted";

type WebsocketPayload =
  | CanvasesCanvasNodeExecution
  | CanvasesCanvasEvent
  | CanvasesCanvasNodeQueueItem
  | CanvasWebsocketPayload;

interface QueuedMessage {
  data: {
    event: string;
    payload: WebsocketPayload;
  };
  timestamp: number;
}

export function useCanvasWebsocket(
  canvasId: string,
  organizationId: string,
  onNodeEvent?: (nodeId: string, event: string) => void,
  onWorkflowEvent?: (event: CanvasesCanvasEvent, eventName: string) => void,
  onExecutionEvent?: (execution: CanvasesCanvasNodeExecution, eventName: string) => void,
  onCanvasLifecycleEvent?: (payload: CanvasWebsocketPayload, eventName: CanvasLifecycleEventName) => void,
  shouldApplyCanvasUpdate?: () => boolean,
): void {
  const nodeExecutionStore = useNodeExecutionStore();
  const queryClient = useQueryClient();

  // Queue for messages per nodeId
  const messageQueues = useRef<Map<string, QueuedMessage[]>>(new Map());
  const processingNodes = useRef<Set<string>>(new Set());

  const processMessage = useCallback(
    (data: QueuedMessage["data"]) => {
      const payload = data.payload;

      switch (data.event) {
        case "event_created":
        case "workflow_event_created":
          if (payload && "nodeId" in payload && payload.nodeId) {
            const workflowEvent = payload as CanvasesCanvasEvent;
            nodeExecutionStore.updateNodeEvent(workflowEvent.nodeId!, workflowEvent);
            onNodeEvent?.(workflowEvent.nodeId!, data.event);
            onWorkflowEvent?.(workflowEvent, data.event);
          }
          break;
        case "execution_created":
        case "execution_started":
        case "execution_finished":
          if (payload && "nodeId" in payload && payload.nodeId) {
            const execution = payload as CanvasesCanvasNodeExecution;
            if (execution.nodeId) {
              const storeNodeId =
                execution.parentExecutionId && execution.nodeId.includes(":")
                  ? execution.nodeId.split(":")[0]
                  : execution.nodeId;

              nodeExecutionStore.updateNodeExecution(storeNodeId, execution);

              if (execution.rootEvent?.id) {
                queryClient.invalidateQueries({
                  queryKey: canvasKeys.eventExecution(canvasId, execution.rootEvent.id),
                });
              }
              onNodeEvent?.(execution.nodeId!, data.event);
              onExecutionEvent?.(execution, data.event);
            }
          }
          break;
        case "queue_item_created":
          if (payload && "nodeId" in payload && payload.nodeId) {
            const queueItem = payload as CanvasesCanvasNodeQueueItem;
            nodeExecutionStore.addNodeQueueItem(queueItem.nodeId!, queueItem);
            onNodeEvent?.(queueItem.nodeId!, data.event);
          }
          break;
        case "queue_item_consumed":
          if (payload && "nodeId" in payload && payload.nodeId && "id" in payload && payload.id) {
            const queueItem = payload as CanvasesCanvasNodeQueueItem;
            nodeExecutionStore.removeNodeQueueItem(queueItem.nodeId!, queueItem.id!);
            onNodeEvent?.(queueItem.nodeId!, data.event);
          }
          break;
        case "canvas_updated":
        case "canvas_deleted": {
          // Canvas structure changed from another actor (e.g. CLI), refresh cache.
          const canvasMessage = payload as CanvasWebsocketPayload;
          if (canvasMessage.canvasId && canvasMessage.canvasId !== canvasId) {
            break;
          }
          onCanvasLifecycleEvent?.(canvasMessage, data.event);

          if (data.event === "canvas_deleted") {
            queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
            break;
          }

          if (!shouldApplyCanvasUpdate?.()) {
            break;
          }

          queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
          queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
          break;
        }
        default:
          break;
      }
    },
    [
      nodeExecutionStore,
      queryClient,
      canvasId,
      onNodeEvent,
      onWorkflowEvent,
      onExecutionEvent,
      onCanvasLifecycleEvent,
      shouldApplyCanvasUpdate,
    ],
  );

  const processQueue = useCallback(
    async (nodeId: string) => {
      // If already processing this node, skip
      if (processingNodes.current.has(nodeId)) {
        return;
      }

      const queue = messageQueues.current.get(nodeId);
      if (!queue || queue.length === 0) {
        return;
      }

      processingNodes.current.add(nodeId);

      try {
        // Process messages in order
        while (queue.length > 0) {
          const message = queue.shift();
          if (message) {
            processMessage(message.data);
            // Small delay to ensure state updates are applied
            await new Promise((resolve) => setTimeout(resolve, 0));
          }
        }
      } finally {
        processingNodes.current.delete(nodeId);

        // Check if new messages arrived while processing
        const remainingQueue = messageQueues.current.get(nodeId);
        if (remainingQueue && remainingQueue.length > 0) {
          // Schedule next processing
          setTimeout(() => processQueue(nodeId), 0);
        }
      }
    },
    [processMessage],
  );

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      try {
        const data = JSON.parse(event.data as string);
        const payload = data.payload;

        // Extract nodeId from payload
        let nodeId: string | undefined;
        if (payload && "nodeId" in payload && payload.nodeId) {
          nodeId = payload.nodeId as string;

          // For child executions, use parent nodeId for queuing
          if (data.event.startsWith("execution_") && "parentExecutionId" in payload && payload.parentExecutionId) {
            if (nodeId.includes(":")) {
              nodeId = nodeId.split(":")[0];
            }
          }
        }

        if (!nodeId) {
          // If no nodeId, process immediately (shouldn't happen based on your logic)
          processMessage(data);
          return;
        }

        // Add to queue
        if (!messageQueues.current.has(nodeId)) {
          messageQueues.current.set(nodeId, []);
        }

        const queue = messageQueues.current.get(nodeId)!;
        queue.push({
          data,
          timestamp: Date.now(),
        });

        // Trigger processing
        processQueue(nodeId);
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    },
    [processMessage, processQueue],
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      messageQueues.current.clear();
      processingNodes.current.clear();
    };
  }, []);

  useWebSocket(`${SOCKET_SERVER_URL}${canvasId}?organization_id=${organizationId}`, {
    shouldReconnect: () => true,
    reconnectAttempts: Number.POSITIVE_INFINITY,
    heartbeat: false,
    reconnectInterval: 3000,
    onOpen: () => {},
    onError: () => {},
    onClose: () => {},
    share: false,
    onMessage: onMessage,
  });
}
