import { useCallback, useRef, useEffect } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient } from "@tanstack/react-query";
import { WorkflowsWorkflowNodeExecution, WorkflowsWorkflowEvent, WorkflowsWorkflowNodeQueueItem } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { workflowKeys } from "./useWorkflowData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

interface QueuedMessage {
  id: string;
  timestamp: number;
  sequenceNumber: number;
  event: string;
  payload: any; // eslint-disable-line @typescript-eslint/no-explicit-any
}

interface BatchedUpdate {
  nodeId: string;
  operations: Array<{
    type:
      | "execution_created"
      | "execution_started"
      | "execution_finished"
      | "event_created"
      | "queue_item_created"
      | "queue_item_consumed";
    data: any; // eslint-disable-line @typescript-eslint/no-explicit-any
    event: string;
  }>;
}

const BATCH_INTERVAL = 16;
const MESSAGE_TIMEOUT = 5000;

export function useWorkflowWebsocket(
  workflowId: string,
  organizationId: string,
  onNodeEvent?: (nodeId: string, event: string) => void,
): void {
  const nodeExecutionStore = useNodeExecutionStore();
  const queryClient = useQueryClient();

  const messageQueue = useRef<QueuedMessage[]>([]);
  const isProcessing = useRef(false);
  const sequenceCounter = useRef(0);
  const batchedUpdates = useRef<Map<string, BatchedUpdate>>(new Map());
  const batchTimer = useRef<NodeJS.Timeout | null>(null);

  const processBatchedUpdates = useCallback(() => {
    if (batchedUpdates.current.size === 0) return;

    console.log("Processing batched updates at:", new Date().toISOString(), "Batch size:", batchedUpdates.current.size);

    for (const [nodeId, batch] of batchedUpdates.current.entries()) {
      console.log(
        `Processing batch for nodeId: ${nodeId}, operations:`,
        batch.operations.map((op) => op.type),
      );
      for (const operation of batch.operations) {
        switch (operation.type) {
          case "event_created": {
            nodeExecutionStore.updateNodeEvent(nodeId, operation.data);
            onNodeEvent?.(nodeId, operation.event);
            break;
          }
          case "execution_created":
          case "execution_started":
          case "execution_finished": {
            const execution = operation.data as WorkflowsWorkflowNodeExecution;
            const storeNodeId =
              execution.parentExecutionId && execution.nodeId?.includes(":")
                ? execution.nodeId.split(":")[0]
                : execution.nodeId!;

            console.log("Updating store with execution:", {
              storeNodeId,
              executionType: operation.type,
              execution: {
                id: execution.id,
                nodeId: execution.nodeId,
                state: execution.state,
                result: execution.result,
              },
              event: operation.event,
            });

            nodeExecutionStore.updateNodeExecution(storeNodeId, execution);

            if (execution.rootEvent?.id) {
              queryClient.invalidateQueries({
                queryKey: workflowKeys.eventExecution(workflowId, execution.rootEvent.id),
              });
            }
            onNodeEvent?.(execution.nodeId!, operation.event);
            break;
          }
          case "queue_item_created": {
            nodeExecutionStore.addNodeQueueItem(nodeId, operation.data);
            onNodeEvent?.(nodeId, operation.event);
            break;
          }
          case "queue_item_consumed": {
            const queueItem = operation.data as WorkflowsWorkflowNodeQueueItem;
            if (queueItem.id) {
              nodeExecutionStore.removeNodeQueueItem(nodeId, queueItem.id);
              onNodeEvent?.(nodeId, operation.event);
            }
            break;
          }
        }
      }
    }

    batchedUpdates.current.clear();
  }, [nodeExecutionStore, queryClient, workflowId, onNodeEvent]);

  const addToBatch = useCallback(
    (nodeId: string, operation: BatchedUpdate["operations"][0]) => {
      if (!batchedUpdates.current.has(nodeId)) {
        batchedUpdates.current.set(nodeId, { nodeId, operations: [] });
      }
      batchedUpdates.current.get(nodeId)!.operations.push(operation);

      if (batchTimer.current) {
        clearTimeout(batchTimer.current);
      }
      batchTimer.current = setTimeout(processBatchedUpdates, BATCH_INTERVAL);
    },
    [processBatchedUpdates],
  );

  const processMessage = useCallback(
    (message: QueuedMessage) => {
      const { payload, event } = message;

      console.log("Processing websocket message:", { event, payload });

      switch (event) {
        case "event_created":
          if (payload && payload.nodeId) {
            const workflowEvent = payload as WorkflowsWorkflowEvent;
            addToBatch(workflowEvent.nodeId!, {
              type: "event_created",
              data: workflowEvent,
              event,
            });
          }
          break;
        case "execution_created":
        case "execution_started":
        case "execution_finished":
          if (payload && payload.nodeId) {
            const execution = payload as WorkflowsWorkflowNodeExecution;
            if (execution.nodeId) {
              const storeNodeId =
                execution.parentExecutionId && execution.nodeId.includes(":")
                  ? execution.nodeId.split(":")[0]
                  : execution.nodeId;

              console.log("Processing execution:", {
                event,
                originalNodeId: execution.nodeId,
                storeNodeId,
                state: execution.state,
                result: execution.result,
                parentExecutionId: execution.parentExecutionId,
                messageTimestamp: message.timestamp,
                messageId: message.id,
                createdAt: execution.createdAt,
                updatedAt: execution.updatedAt,
              });

              addToBatch(storeNodeId, {
                type: event, // Use actual event type instead of "execution"
                data: execution,
                event,
              });
            }
          }
          break;
        case "queue_item_created":
          if (payload && payload.nodeId) {
            const queueItem = payload as WorkflowsWorkflowNodeQueueItem;
            addToBatch(queueItem.nodeId!, {
              type: "queue_item_created",
              data: queueItem,
              event,
            });
          }
          break;
        case "queue_item_consumed":
          if (payload && payload.nodeId && payload.id) {
            const queueItem = payload as WorkflowsWorkflowNodeQueueItem;
            addToBatch(queueItem.nodeId!, {
              type: "queue_item_consumed",
              data: queueItem,
              event,
            });
          }
          break;
        default:
          break;
      }
    },
    [addToBatch],
  );

  const processQueue = useCallback(async () => {
    if (isProcessing.current || messageQueue.current.length === 0) {
      return;
    }

    isProcessing.current = true;

    messageQueue.current.sort((a, b) => {
      if (a.timestamp !== b.timestamp) {
        return a.timestamp - b.timestamp;
      }
      return a.sequenceNumber - b.sequenceNumber;
    });

    while (messageQueue.current.length > 0) {
      const message = messageQueue.current.shift()!;

      if (Date.now() - message.timestamp > MESSAGE_TIMEOUT) {
        console.warn("Skipping old message:", message);
        continue;
      }

      processMessage(message);
    }

    isProcessing.current = false;

    if (messageQueue.current.length > 0) {
      setTimeout(processQueue, 0);
    }
  }, [processMessage]);

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      try {
        const data = JSON.parse(event.data as string);
        const payload = data.payload;

        const queuedMessage: QueuedMessage = {
          id: `${Date.now()}-${sequenceCounter.current++}`,
          timestamp: Date.now(),
          sequenceNumber: sequenceCounter.current,
          event: data.event,
          payload: payload,
        };

        messageQueue.current.push(queuedMessage);

        processQueue();
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    },
    [processQueue],
  );

  useEffect(() => {
    return () => {
      if (batchTimer.current) {
        clearTimeout(batchTimer.current);
        processBatchedUpdates();
      }
    };
  }, [processBatchedUpdates]);

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
