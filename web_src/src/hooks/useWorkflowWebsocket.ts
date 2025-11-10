import { useCallback } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient } from "@tanstack/react-query";
import { WorkflowsWorkflowNodeExecution, WorkflowsWorkflowEvent, WorkflowsWorkflow } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { workflowKeys } from "./useWorkflowData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

export function useWorkflowWebsocket(workflowId: string, organizationId: string): void {
  const nodeExecutionStore = useNodeExecutionStore();
  const queryClient = useQueryClient();

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      try {
        const data = JSON.parse(event.data as string);
        const payload = data.payload;
        // eslint-disable-next-line no-console
        console.log("[WS] message", { event: data.event, payload });

        switch (data.event) {
          case "event_created":
            // Payload contains the full WorkflowEvent
            if (payload && payload.nodeId) {
              const workflowEvent = payload as WorkflowsWorkflowEvent;
              nodeExecutionStore.updateNodeEvent(workflowEvent.nodeId!, workflowEvent);

              // Also refetch queue items for downstream nodes of the emitting node.
              // This keeps "Next in queue" up-to-date without a full page refresh.
              try {
                const workflow = queryClient.getQueryData(workflowKeys.detail(organizationId, workflowId)) as
                  | WorkflowsWorkflow
                  | undefined;

                if (workflow?.spec?.edges && workflow?.spec?.nodes) {
                  const edges = workflow.spec.edges || [];
                  const nodes = workflow.spec.nodes || [];

                  // Find edges that start from this node and, if channel provided, match it
                  const matchingEdges = edges.filter((e: any) => {
                    if (!e?.sourceId) return false;
                    if (e.sourceId !== workflowEvent.nodeId) return false;
                    if (workflowEvent.channel && e.channel && e.channel !== workflowEvent.channel) return false;
                    // If either side doesn't specify channel, treat as match
                    return true;
                  });

                  matchingEdges.forEach((edge: any) => {
                    const targetId = edge?.targetId;
                    if (!targetId) return;
                    const targetNode = nodes.find((n: any) => n.id === targetId);
                    const nodeType = (targetNode?.type as string) || "";
                    // Trigger nodes do not have queues
                    if (nodeType === "TYPE_TRIGGER") return;
                    // eslint-disable-next-line no-console
                    console.log("[WS] refetch queue for target node", { targetId, nodeType });
                    nodeExecutionStore.refetchNodeData(workflowId, targetId, nodeType, queryClient).catch(() => {});
                  });
                }
              } catch (e) {
                // Best-effort; ignore errors
              }
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
              }
            }
            break;
          case "queue_item_created": {
            // Payload is a serialized WorkflowNodeQueueItem
            const item = payload as any; // matches WorkflowsWorkflowNodeQueueItem shape
            const nodeId = item?.nodeId;
            if (nodeId) {
              // eslint-disable-next-line no-console
              console.log("[WS] queue_item_created", item);
              // Prefer a light refetch for the target node queue list (keeps pagination/simple)
              const workflow = queryClient.getQueryData(workflowKeys.detail(organizationId, workflowId)) as
                | WorkflowsWorkflow
                | undefined;
              const nodeType = workflow?.spec?.nodes?.find((n: any) => n.id === nodeId)?.type || "";
              if (nodeType !== "TYPE_TRIGGER") {
                // eslint-disable-next-line no-console
                console.log("[WS] refetch due to queue_item_created", { nodeId, nodeType });
                nodeExecutionStore.refetchNodeData(workflowId, nodeId, nodeType, queryClient).catch(() => {});
              }
            }
            break;
          }
          case "queue_item_deleted": {
            const nodeId = (payload as any)?.node_id || (payload as any)?.nodeId;
            if (nodeId) {
              // eslint-disable-next-line no-console
              console.log("[WS] queue_item_deleted", payload);
              const workflow = queryClient.getQueryData(workflowKeys.detail(organizationId, workflowId)) as
                | WorkflowsWorkflow
                | undefined;
              const nodeType = workflow?.spec?.nodes?.find((n: any) => n.id === nodeId)?.type || "";
              if (nodeType !== "TYPE_TRIGGER") {
                // eslint-disable-next-line no-console
                console.log("[WS] refetch due to queue_item_deleted", { nodeId, nodeType });
                nodeExecutionStore.refetchNodeData(workflowId, nodeId, nodeType, queryClient).catch(() => {});
              }
            }
            break;
          }
          default:
            break;
        }
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    },
    [nodeExecutionStore, queryClient, workflowId],
  );

  useWebSocket(`${SOCKET_SERVER_URL}${workflowId}?organization_id=${organizationId}`, {
    shouldReconnect: () => true,
    reconnectAttempts: 10,
    heartbeat: false,
    reconnectInterval: 3000,
    onOpen: () => {
      // eslint-disable-next-line no-console
      console.log("[WS] open", { url: `${SOCKET_SERVER_URL}${workflowId}?organization_id=${organizationId}` });
    },
    onError: (e) => {
      // eslint-disable-next-line no-console
      console.error("[WS] error", e);
    },
    onClose: (e) => {
      // eslint-disable-next-line no-console
      console.log("[WS] close", e.code, e.reason);
    },
    share: false, // Setting share to false to avoid issues with multiple connections
    onMessage: onMessage,
  });
}
