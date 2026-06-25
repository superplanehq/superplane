import { useCallback, useEffect, useRef } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasRun,
} from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import {
  parseRunsFiltersFromQueryKey,
  upsertExecutionIntoInfiniteRunsData,
  upsertRunIntoDescribeRunData,
  upsertRunIntoInfiniteData,
  type InfiniteRunsPage,
} from "./canvasInfiniteCache";
import { canvasKeys } from "./useCanvasData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/`;

type CanvasWebsocketPayload = {
  canvasId: string;
  versionId?: string;
};

type RepositoryBranchUpdatedPayload = {
  canvasId: string;
  branch?: string;
  headSha?: string;
  materializationStatus?: string;
  materializationError?: string;
};

type CanvasLifecycleEventName =
  | "canvas_updated"
  | "canvas_version_updated"
  | "canvas_version_deleted"
  | "canvas_deleted"
  | "repository_branch_updated";

type CanvasStagingEventName = "staging_updated";

type WebsocketPayload =
  | CanvasesCanvasNodeExecution
  | CanvasesCanvasEvent
  | CanvasesCanvasNodeQueueItem
  | CanvasesCanvasRun
  | CanvasWebsocketPayload;

interface QueuedMessage {
  data: {
    event: string;
    payload: WebsocketPayload;
  };
  timestamp: number;
}

function queryKeyStartsWith(queryKey: readonly unknown[], prefix: readonly unknown[]): boolean {
  return prefix.every((part, index) => queryKey[index] === part);
}

function isDraftRepositoryFileQuery(queryKey: readonly unknown[], canvasId: string, versionId: string): boolean {
  const repositoryPrefix = canvasKeys.repository(canvasId);
  const fileSegmentIndex = repositoryPrefix.length;
  return (
    queryKeyStartsWith(queryKey, repositoryPrefix) &&
    queryKey.length === repositoryPrefix.length + 4 &&
    queryKey[fileSegmentIndex] === "file" &&
    queryKey[fileSegmentIndex + 2] === versionId &&
    queryKey[fileSegmentIndex + 3] === "staged"
  );
}

// Refreshes caches that read a draft version's staging layer. versionStagedDetail,
// consoleStaged and staged repositoryFileContent keys all end with "staged";
// repositoryFile keys feed the visible Files tab editor and include the draft id.
function invalidateStagedCanvasQueries(queryClient: QueryClient, canvasId: string, versionId: string): void {
  queryClient.invalidateQueries({ queryKey: canvasKeys.versionStaging(canvasId, versionId) });
  queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId) });
  queryClient.invalidateQueries({
    predicate: (query) => {
      const key = query.queryKey;
      if (!Array.isArray(key) || !key.includes(versionId)) {
        return false;
      }

      return key[key.length - 1] === "staged" || isDraftRepositoryFileQuery(key, canvasId, versionId);
    },
  });
}

export function useCanvasWebsocket(
  canvasId: string,
  organizationId: string,
  onNodeEvent?: (nodeId: string, event: string) => void,
  onWorkflowEvent?: (event: CanvasesCanvasEvent, eventName: string) => void,
  onExecutionEvent?: (execution: CanvasesCanvasNodeExecution, eventName: string) => void,
  onCanvasLifecycleEvent?: (payload: CanvasWebsocketPayload, eventName: CanvasLifecycleEventName) => boolean | void,
  shouldApplyCanvasUpdate?: () => boolean,
  processRuntimeEvents = true,
  enabled = true,
  onCanvasStagingEvent?: (payload: CanvasWebsocketPayload, eventName: CanvasStagingEventName) => boolean | void,
): void {
  const nodeExecutionStore = useNodeExecutionStore();
  const queryClient = useQueryClient();

  // Queue for messages per nodeId
  const messageQueues = useRef<Map<string, QueuedMessage[]>>(new Map());
  const processingNodes = useRef<Set<string>>(new Set());

  const handleCanvasLifecycleEvent = useCallback(
    (eventName: CanvasLifecycleEventName, payload: WebsocketPayload) => {
      // Canvas structure changed from another actor (e.g. CLI), refresh cache.
      const canvasMessage = payload as Partial<CanvasWebsocketPayload & RepositoryBranchUpdatedPayload>;
      if (!canvasMessage.canvasId || canvasMessage.canvasId !== canvasId) {
        return;
      }

      if (eventName === "canvas_version_updated" && !canvasMessage.versionId) {
        return;
      }

      if (eventName === "canvas_version_deleted" && !canvasMessage.versionId) {
        return;
      }

      const shouldInvalidateLifecycleQueries =
        onCanvasLifecycleEvent?.(canvasMessage as CanvasWebsocketPayload, eventName) !== false;
      if (!shouldInvalidateLifecycleQueries) {
        return;
      }

      if (eventName === "canvas_deleted") {
        queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
        return;
      }

      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });

      if (eventName === "repository_branch_updated") {
        queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
        return;
      }

      if (eventName === "canvas_version_updated") {
        queryClient.invalidateQueries({ queryKey: canvasKeys.consoleAll(canvasId) });
        return;
      }

      if (eventName === "canvas_version_deleted") {
        queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
        return;
      }

      if (!shouldApplyCanvasUpdate?.()) {
        return;
      }

      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    },
    [canvasId, organizationId, queryClient, onCanvasLifecycleEvent, shouldApplyCanvasUpdate],
  );

  const hasConnectedOnce = useRef(false);

  const patchRunInCache = useCallback(
    (run: CanvasesCanvasRun) => {
      const queries = queryClient.getQueriesData<InfiniteData<InfiniteRunsPage>>({
        queryKey: canvasKeys.infiniteRuns(canvasId),
      });

      for (const [queryKey, data] of queries) {
        if (!data) {
          continue;
        }

        const filters = parseRunsFiltersFromQueryKey(queryKey);
        const next = upsertRunIntoInfiniteData(data, run, filters);
        if (next !== data) {
          queryClient.setQueryData(queryKey, next);
        }
      }

      if (run.id) {
        queryClient.setQueryData<{ run?: CanvasesCanvasRun }>(canvasKeys.run(canvasId, run.id), (current) =>
          upsertRunIntoDescribeRunData(current, run),
        );
      }
    },
    [queryClient, canvasId],
  );

  const patchExecutionInCache = useCallback(
    (execution: CanvasesCanvasNodeExecution) => {
      queryClient.setQueriesData<InfiniteData<InfiniteRunsPage>>(
        { queryKey: canvasKeys.infiniteRuns(canvasId) },
        (old) => upsertExecutionIntoInfiniteRunsData(old, execution),
      );
    },
    [queryClient, canvasId],
  );

  const invalidateMemoryEntries = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: canvasKeys.canvasMemoryEntries(canvasId),
    });
  }, [queryClient, canvasId]);

  const processMessage = useCallback(
    (data: QueuedMessage["data"]) => {
      const payload = data.payload;
      const isCanvasLifecycleEvent =
        data.event === "canvas_updated" ||
        data.event === "canvas_version_updated" ||
        data.event === "canvas_version_deleted" ||
        data.event === "canvas_deleted" ||
        data.event === "repository_branch_updated";
      // Staging events fire while editing a draft (not the live version), so they
      // must bypass the runtime-event gate that is disabled outside the live view.
      const isCanvasStagingEvent = data.event === "staging_updated";
      // Memory updates can happen from manual mutations regardless of the live
      // view, so they bypass the runtime-event gate as well.
      const isMemoryUpdatedEvent = data.event === "memory_updated";
      if (!isCanvasLifecycleEvent && !isCanvasStagingEvent && !isMemoryUpdatedEvent && !processRuntimeEvents) {
        return;
      }

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
              nodeExecutionStore.updateNodeExecution(execution.nodeId, execution);

              patchExecutionInCache(execution);

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
        case "run_started":
        case "run_finished": {
          const run = payload as CanvasesCanvasRun;
          if (!run.canvasId || run.canvasId !== canvasId) {
            break;
          }

          patchRunInCache(run);
          break;
        }
        case "canvas_updated":
        case "canvas_version_updated":
        case "canvas_version_deleted":
        case "canvas_deleted":
        case "repository_branch_updated":
          handleCanvasLifecycleEvent(data.event as CanvasLifecycleEventName, payload);
          break;
        case "staging_updated": {
          // A draft's staging layer changed in another tab (or this one). Refresh
          // the staged caches so the diff badge, console and files tabs reflect
          // the uncommitted changes.
          const stagingMessage = payload as Partial<CanvasWebsocketPayload>;
          if (!stagingMessage.canvasId || stagingMessage.canvasId !== canvasId || !stagingMessage.versionId) {
            break;
          }

          const shouldInvalidateStagingQueries =
            onCanvasStagingEvent?.(stagingMessage as CanvasWebsocketPayload, "staging_updated") !== false;
          if (shouldInvalidateStagingQueries) {
            invalidateStagedCanvasQueries(queryClient, canvasId, stagingMessage.versionId);
          }
          break;
        }
        case "memory_updated": {
          // Canvas memory changed (from a node execution, manual mutation, or
          // another tab). Invalidate the shared memory cache so the Memory tab
          // and any memory-bound widget refetches once.
          const memoryMessage = payload as Partial<CanvasWebsocketPayload>;
          if (memoryMessage.canvasId === canvasId) {
            invalidateMemoryEntries();
          }
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
      onCanvasStagingEvent,
      processRuntimeEvents,
      handleCanvasLifecycleEvent,
      patchRunInCache,
      patchExecutionInCache,
      invalidateMemoryEntries,
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

  const handleWebSocketOpen = useCallback(() => {
    if (!hasConnectedOnce.current) {
      hasConnectedOnce.current = true;
      return;
    }

    queryClient.invalidateQueries({
      queryKey: canvasKeys.infiniteRuns(canvasId),
    });
    // Refresh memory in case mutations happened while we were disconnected; we
    // no longer poll, so the websocket is the only push channel.
    invalidateMemoryEntries();
  }, [queryClient, canvasId, invalidateMemoryEntries]);

  // Cleanup on unmount
  useEffect(() => {
    const queues = messageQueues.current;
    const processing = processingNodes.current;
    return () => {
      queues.clear();
      processing.clear();
    };
  }, []);

  useWebSocket(
    `${SOCKET_SERVER_URL}${canvasId}?organization_id=${organizationId}`,
    {
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      heartbeat: false,
      reconnectInterval: 3000,
      onOpen: handleWebSocketOpen,
      onError: () => {},
      onClose: () => {},
      share: false,
      onMessage: onMessage,
    },
    enabled,
  );
}
