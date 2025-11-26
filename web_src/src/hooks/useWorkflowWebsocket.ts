import { useCallback, useEffect, useRef } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient } from "@tanstack/react-query";
import { WorkflowsWorkflowNodeExecution, WorkflowsWorkflowEvent, WorkflowsWorkflowNodeQueueItem } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { workflowKeys } from "./useWorkflowData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

interface QueuedMessage {
  data: {
    event: string;
    payload: WorkflowsWorkflowNodeExecution | WorkflowsWorkflowEvent | WorkflowsWorkflowNodeQueueItem;
  };
  timestamp: number;
}

export function useWorkflowWebsocket(
  workflowId: string,
  organizationId: string,
  onNodeEvent?: (nodeId: string, event: string) => void,
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
          if (payload && "nodeId" in payload && payload.nodeId) {
            const workflowEvent = payload as WorkflowsWorkflowEvent;
            nodeExecutionStore.updateNodeEvent(workflowEvent.nodeId!, workflowEvent);
            onNodeEvent?.(workflowEvent.nodeId!, data.event);
          }
          break;
        case "execution_created":
        case "execution_started":
        case "execution_finished":
          if (payload && "nodeId" in payload && payload.nodeId) {
            const execution = payload as WorkflowsWorkflowNodeExecution;
            if (execution.nodeId) {
              const storeNodeId =
                execution.parentExecutionId && execution.nodeId.includes(":")
                  ? execution.nodeId.split(":")[0]
                  : execution.nodeId;

              nodeExecutionStore.updateNodeExecution(storeNodeId, execution);

              if (execution.rootEvent?.id) {
                queryClient.invalidateQueries({
                  queryKey: workflowKeys.eventExecution(workflowId, execution.rootEvent.id),
                });
              }
              onNodeEvent?.(execution.nodeId!, data.event);
            }
          }
          break;
        case "queue_item_created":
          if (payload && "nodeId" in payload && payload.nodeId) {
            const queueItem = payload as WorkflowsWorkflowNodeQueueItem;
            nodeExecutionStore.addNodeQueueItem(queueItem.nodeId!, queueItem);
            onNodeEvent?.(queueItem.nodeId!, data.event);
          }
          break;
        case "queue_item_consumed":
          if (payload && "nodeId" in payload && payload.nodeId && "id" in payload && payload.id) {
            const queueItem = payload as WorkflowsWorkflowNodeQueueItem;
            nodeExecutionStore.removeNodeQueueItem(queueItem.nodeId!, queueItem.id!);
            onNodeEvent?.(queueItem.nodeId!, data.event);
          }
          break;
        default:
          break;
      }
    },
    [nodeExecutionStore, queryClient, workflowId, onNodeEvent],
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

  useWebSocket(`${SOCKET_SERVER_URL}${workflowId}?organization_id=${organizationId}`, {
    shouldReconnect: () => true,
    reconnectAttempts: 10,
    heartbeat: false,
    reconnectInterval: 3000,
    onOpen: () => {},
    onError: () => {},
    onClose: () => {},
    share: false,
    onMessage: onMessage,
  });
}
