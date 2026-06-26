import { create } from "zustand";
import type { QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasEvent,
  CanvasesCanvas,
  CanvasesListNodeExecutionsResponse,
  CanvasesListNodeQueueItemsResponse,
  CanvasesListNodeEventsResponse,
} from "@/api-client";
import {
  NODE_EXECUTION_HISTORY_PAGE_SIZE,
  nodeExecutionsQueryOptions,
  nodeQueueItemsQueryOptions,
  nodeEventsQueryOptions,
} from "@/hooks/useCanvasData";
import { shouldAcceptExecutionUpdate } from "@/hooks/canvasInfiniteCache";

interface NodeExecutionData {
  executions: CanvasesCanvasNodeExecution[];
  queueItems: CanvasesCanvasNodeQueueItem[];
  events: CanvasesCanvasEvent[];
  isLoading: boolean;
  isLoaded: boolean;
  totalInHistoryCount: number;
  totalInQueueCount: number;
}

interface NodeExecutionStore {
  // State
  data: Map<string, NodeExecutionData>;
  version: number; // Version counter to track updates

  // Actions
  initializeFromWorkflow: (workflow: CanvasesCanvas) => void;
  loadNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  refetchNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  getNodeData: (nodeId: string) => NodeExecutionData;
  updateNodeExecution: (nodeId: string, execution: CanvasesCanvasNodeExecution) => void;
  updateNodeEvent: (nodeId: string, event: CanvasesCanvasEvent) => void;
  addNodeQueueItem: (nodeId: string, queueItem: CanvasesCanvasNodeQueueItem) => void;
  removeNodeQueueItem: (nodeId: string, queueItemId: string) => void;
  clear: () => void;
}

const emptyNodeData: NodeExecutionData = {
  executions: [],
  queueItems: [],
  events: [],
  isLoading: false,
  isLoaded: false,
  totalInHistoryCount: 0,
  totalInQueueCount: 0,
};

/**
 * Invalidates queries for a specific node based on its type
 */
async function invalidateNodeQueries(workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) {
  await Promise.all([
    nodeType !== "TYPE_TRIGGER"
      ? queryClient.invalidateQueries(nodeExecutionsQueryOptions(workflowId, nodeId))
      : Promise.resolve(),
    nodeType !== "TYPE_TRIGGER"
      ? queryClient.invalidateQueries(nodeQueueItemsQueryOptions(workflowId, nodeId))
      : Promise.resolve(),
    nodeType === "TYPE_TRIGGER"
      ? queryClient.invalidateQueries(nodeEventsQueryOptions(workflowId, nodeId))
      : Promise.resolve(),
  ]);
}

/**
 * Fetches node data based on its type
 */
async function fetchNodeData(
  workflowId: string,
  nodeId: string,
  nodeType: string,
  queryClient: QueryClient,
): Promise<[CanvasesListNodeExecutionsResponse, CanvasesListNodeQueueItemsResponse, CanvasesListNodeEventsResponse]> {
  return await Promise.all([
    nodeType !== "TYPE_TRIGGER"
      ? queryClient.fetchQuery(
          nodeExecutionsQueryOptions(workflowId, nodeId, { limit: NODE_EXECUTION_HISTORY_PAGE_SIZE }),
        )
      : Promise.resolve({ executions: [] }),
    nodeType !== "TYPE_TRIGGER"
      ? queryClient.fetchQuery(nodeQueueItemsQueryOptions(workflowId, nodeId))
      : Promise.resolve({ items: [] }),
    nodeType === "TYPE_TRIGGER"
      ? queryClient.fetchQuery(nodeEventsQueryOptions(workflowId, nodeId))
      : Promise.resolve({ events: [] }),
  ]);
}

export const useNodeExecutionStore = create<NodeExecutionStore>((set, get) => ({
  data: new Map(),
  version: 0,

  initializeFromWorkflow: (workflow) => {
    const initialData = new Map<string, NodeExecutionData>();

    // Populate with last executions from workflow.status
    workflow.status?.lastExecutions?.forEach((execution) => {
      if (!execution.nodeId) return;

      const existing = initialData.get(execution.nodeId) || { ...emptyNodeData };
      initialData.set(execution.nodeId, {
        ...existing,
        executions: [execution],
      });
    });

    // Populate with last events from workflow.status
    workflow.status?.lastEvents?.forEach((event) => {
      if (!event.nodeId) return;

      const existing = initialData.get(event.nodeId) || { ...emptyNodeData };
      initialData.set(event.nodeId, {
        ...existing,
        events: [event],
      });
    });

    set({ data: initialData, version: get().version + 1 });
  },

  loadNodeData: async (workflowId, nodeId, nodeType, queryClient) => {
    set((state) => {
      const newData = new Map(state.data);
      newData.set(nodeId, {
        ...(newData.get(nodeId) || emptyNodeData),
        isLoading: true,
      });
      return { data: newData, version: state.version + 1 };
    });

    try {
      // Invalidate queries first to ensure fresh data
      await invalidateNodeQueries(workflowId, nodeId, nodeType, queryClient);

      // Fetch full data in parallel
      const [executionsResult, queueItemsResult, eventsResult] = await fetchNodeData(
        workflowId,
        nodeId,
        nodeType,
        queryClient,
      );

      // Update with full data
      set((state) => {
        const newData = new Map(state.data);
        newData.set(nodeId, {
          executions: executionsResult.executions || [],
          queueItems: queueItemsResult.items || [],
          events: eventsResult.events || [],
          totalInHistoryCount:
            nodeType === "TYPE_TRIGGER" ? eventsResult?.totalCount || 0 : executionsResult?.totalCount || 0,
          totalInQueueCount: queueItemsResult?.totalCount || 0,
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData, version: state.version + 1 };
      });
    } catch {
      // Mark as not loading on error
      set((state) => {
        const newData = new Map(state.data);
        const existing = newData.get(nodeId);
        if (existing) {
          newData.set(nodeId, { ...existing, isLoading: false });
        }
        return { data: newData, version: state.version + 1 };
      });
    }
  },

  refetchNodeData: async (workflowId, nodeId, nodeType, queryClient) => {
    // Similar to loadNodeData but always refetches, even if already loaded
    // This is used by websocket updates to refresh data

    // Mark as loading
    set((state) => {
      const newData = new Map(state.data);
      const existing = newData.get(nodeId) || emptyNodeData;
      newData.set(nodeId, {
        ...existing,
        isLoading: true,
      });
      return { data: newData, version: state.version + 1 };
    });

    try {
      // Invalidate queries first to ensure fresh data
      await invalidateNodeQueries(workflowId, nodeId, nodeType, queryClient);

      // Fetch fresh data in parallel
      const [executionsResult, queueItemsResult, eventsResult]: [
        CanvasesListNodeExecutionsResponse,
        CanvasesListNodeQueueItemsResponse,
        CanvasesListNodeEventsResponse,
      ] = await Promise.all([
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(
              nodeExecutionsQueryOptions(workflowId, nodeId, { limit: NODE_EXECUTION_HISTORY_PAGE_SIZE }),
            )
          : Promise.resolve({ executions: [] }),
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeQueueItemsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ items: [] }),
        nodeType === "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeEventsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ events: [] }),
      ]);

      // Update with fresh data
      set((state) => {
        const newData = new Map(state.data);
        newData.set(nodeId, {
          executions: executionsResult.executions || [],
          queueItems: queueItemsResult.items || [],
          events: eventsResult.events || [],
          totalInHistoryCount:
            nodeType !== "TYPE_TRIGGER" ? executionsResult?.totalCount || 0 : eventsResult?.totalCount || 0,
          totalInQueueCount: queueItemsResult?.totalCount || 0,
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData, version: state.version + 1 };
      });
    } catch {
      // Mark as not loading on error
      set((state) => {
        const newData = new Map(state.data);
        const existing = newData.get(nodeId);
        if (existing) {
          newData.set(nodeId, { ...existing, isLoading: false });
        }
        return { data: newData, version: state.version + 1 };
      });
    }
  },

  getNodeData: (nodeId) => {
    return get().data.get(nodeId) || emptyNodeData;
  },

  updateNodeExecution: (nodeId, execution) => {
    set((state) => {
      const newData = new Map(state.data);
      const existing = newData.get(nodeId) || emptyNodeData;
      const existingIndex = existing.executions.findIndex((e) => e.id === execution.id);

      // Ignore stale out-of-order updates so a finished node isn't downgraded
      // back to running.
      if (existingIndex >= 0 && !shouldAcceptExecutionUpdate(existing.executions[existingIndex], execution)) {
        return state;
      }

      const updatedExecutions =
        existingIndex >= 0
          ? existing.executions.map((e, i) => (i === existingIndex ? execution : e))
          : [execution, ...existing.executions];

      newData.set(nodeId, {
        ...existing,
        executions: updatedExecutions,
        isLoaded: true,
      });
      return { data: newData, version: state.version + 1 };
    });
  },

  updateNodeEvent: (nodeId, event) => {
    set((state) => {
      const newData = new Map(state.data);
      const existing = newData.get(nodeId) || emptyNodeData;

      // Add or update the event in the list
      const existingIndex = existing.events.findIndex((e) => e.id === event.id);
      const updatedEvents =
        existingIndex >= 0
          ? existing.events.map((e, i) => (i === existingIndex ? event : e))
          : [event, ...existing.events];

      newData.set(nodeId, {
        ...existing,
        events: updatedEvents,
        isLoaded: true,
      });
      return { data: newData, version: state.version + 1 };
    });
  },

  addNodeQueueItem: (nodeId, queueItem) => {
    set((state) => {
      const newData = new Map(state.data);
      const existing = newData.get(nodeId) || emptyNodeData;

      // Add the queue item to the beginning of the list (most recent first)
      const updatedQueueItems = [queueItem, ...existing.queueItems];

      newData.set(nodeId, {
        ...existing,
        queueItems: updatedQueueItems,
        isLoaded: true,
      });
      return { data: newData, version: state.version + 1 };
    });
  },

  removeNodeQueueItem: (nodeId, queueItemId) => {
    set((state) => {
      const newData = new Map(state.data);
      const existing = newData.get(nodeId) || emptyNodeData;

      // Remove the queue item with the matching ID
      const updatedQueueItems = existing.queueItems.filter((item) => item.id !== queueItemId);

      newData.set(nodeId, {
        ...existing,
        queueItems: updatedQueueItems,
        isLoaded: true,
      });
      return { data: newData, version: state.version + 1 };
    });
  },

  clear: () => {
    set({ data: new Map(), version: 0 });
  },
}));
