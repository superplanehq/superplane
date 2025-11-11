import { useCallback } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient } from "@tanstack/react-query";
import { WorkflowsWorkflowNodeExecution, WorkflowsWorkflowEvent } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { workflowKeys } from "./useWorkflowData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

export function useWorkflowWebsocket(workflowId: string, organizationId: string, onNodeEvent?: (nodeId: string, event: string) => void): void {
  const nodeExecutionStore = useNodeExecutionStore();
  const queryClient = useQueryClient();

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      try {
        const data = JSON.parse(event.data as string);
        const payload = data.payload;

        switch (data.event) {
          case "event_created":
            // Payload contains the full WorkflowEvent
            if (payload && payload.nodeId) {
              const workflowEvent = payload as WorkflowsWorkflowEvent;
              nodeExecutionStore.updateNodeEvent(workflowEvent.nodeId!, workflowEvent);
              onNodeEvent?.(workflowEvent.nodeId!, data.event);
            }
            break;
          case "execution_created":
          case "execution_started":
          case "execution_finished":
            // Payload contains the full WorkflowNodeExecution
            if (payload && payload.nodeId) {
              const execution = payload as WorkflowsWorkflowNodeExecution;
              // For child executions (composite nodes), extract the parent nodeId
              // Pattern: parent-node-id:child-node-id -> use parent-node-id
              if (execution.nodeId) {
                const storeNodeId =
                  execution.parentExecutionId && execution.nodeId.includes(":")
                    ? execution.nodeId.split(":")[0]
                    : execution.nodeId;

                nodeExecutionStore.updateNodeExecution(storeNodeId, execution);

                // Invalidate execution chain query for this root event to refetch updated chain
                if (execution.rootEvent?.id) {
                  queryClient.invalidateQueries({
                    queryKey: workflowKeys.eventExecution(workflowId, execution.rootEvent.id),
                  });
                }
                onNodeEvent?.(execution.nodeId!, data.event);
              }
            }
            break;
          default:
            break;
        }
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    },
    [nodeExecutionStore, queryClient, workflowId, onNodeEvent],
  );

  useWebSocket(`${SOCKET_SERVER_URL}${workflowId}?organization_id=${organizationId}`, {
    shouldReconnect: () => true,
    reconnectAttempts: 10,
    heartbeat: false,
    reconnectInterval: 3000,
    onOpen: () => {},
    onError: () => {},
    onClose: () => {},
    share: false, // Setting share to false to avoid issues with multiple connections
    onMessage: onMessage,
  });
}
