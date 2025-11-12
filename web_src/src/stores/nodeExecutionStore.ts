import { create } from "zustand";
import { QueryClient } from "@tanstack/react-query";
import {
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflow,
  WorkflowsListNodeExecutionsResponse,
  WorkflowsListNodeQueueItemsResponse,
  WorkflowsListNodeEventsResponse,
} from "@/api-client";
import {
  nodeExecutionsQueryOptions,
  nodeQueueItemsQueryOptions,
  nodeEventsQueryOptions,
} from "@/hooks/useWorkflowData";

interface NodeExecutionData {
  executions: WorkflowsWorkflowNodeExecution[];
  queueItems: WorkflowsWorkflowNodeQueueItem[];
  events: WorkflowsWorkflowEvent[];
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
  initializeFromWorkflow: (workflow: WorkflowsWorkflow) => void;
  loadNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  refetchNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  getNodeData: (nodeId: string) => NodeExecutionData;
  updateNodeExecution: (nodeId: string, execution: WorkflowsWorkflowNodeExecution) => void;
  updateNodeEvent: (nodeId: string, event: WorkflowsWorkflowEvent) => void;
  addNodeQueueItem: (nodeId: string, queueItem: WorkflowsWorkflowNodeQueueItem) => void;
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
 * Updates a child execution within the array of parent executions.
 *
 * @param executions The array of parent executions.
 * @param childExecution The child execution to update.
 * @returns
 */
function updateChildExecution(
  executions: WorkflowsWorkflowNodeExecution[],
  childExecution: WorkflowsWorkflowNodeExecution,
): WorkflowsWorkflowNodeExecution[] {
  const parentIndex = executions.findIndex((e) => e.id === childExecution.parentExecutionId);

  /*
   * Parent not found yet.
   * Add as top-level temporarily - will be nested when parent arrives.
   */
  if (parentIndex < 0) {
    const existingIndex = executions.findIndex((e) => e.id === childExecution.id);
    if (existingIndex >= 0) {
      return executions.map((e, i) => (i === existingIndex ? childExecution : e));
    }
    return [childExecution, ...executions];
  }

  /*
   * Parent found - update child within parent's childExecutions.
   */
  const parent = executions[parentIndex];
  const childExecutions = parent.childExecutions || [];
  const childIndex = childExecutions.findIndex((ce) => ce.id === childExecution.id);

  const updatedChildren =
    childIndex >= 0
      ? childExecutions.map((ce, i) => (i === childIndex ? childExecution : ce))
      : [...childExecutions, childExecution];

  return executions.map((e, i) => (i === parentIndex ? { ...e, childExecutions: updatedChildren } : e));
}

/**
 * Updates a parent execution within the array of parent executions.
 *
 * @param executions The array of parent executions.
 * @param parentExecution The parent execution to update.
 * @returns The updated array of parent executions.
 */
function updateParentExecution(
  executions: WorkflowsWorkflowNodeExecution[],
  parentExecution: WorkflowsWorkflowNodeExecution,
): WorkflowsWorkflowNodeExecution[] {
  const existingIndex = executions.findIndex((e) => e.id === parentExecution.id);

  /*
   * Update existing parent - preserve childExecutions
   */
  if (existingIndex >= 0) {
    const existing = executions[existingIndex];
    const finalChildExecutions = parentExecution.childExecutions || existing.childExecutions || [];
    return executions.map((e, i) =>
      i === existingIndex ? { ...parentExecution, childExecutions: finalChildExecutions } : e,
    );
  }

  /*
   * New parent execution arriving.
   * Check for orphaned children that belong to it
   */
  const orphanedChildren = executions.filter((e) => e.parentExecutionId === parentExecution.id);
  if (orphanedChildren.length === 0) {
    return [{ ...parentExecution, childExecutions: parentExecution.childExecutions || [] }, ...executions];
  }

  /*
   * Move orphaned children from top-level into parent
   */
  const withoutOrphans = executions.filter((e) => e.parentExecutionId !== parentExecution.id);
  const parentWithChildren = {
    ...parentExecution,
    childExecutions: [...(parentExecution.childExecutions || []), ...orphanedChildren],
  };

  return [parentWithChildren, ...withoutOrphans];
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

    // Populate with next queue items from workflow.status
    workflow.status?.nextQueueItems?.forEach((item) => {
      if (!item.nodeId) return;

      const existing = initialData.get(item.nodeId) || { ...emptyNodeData };
      initialData.set(item.nodeId, {
        ...existing,
        queueItems: [item],
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
    const current = get().data.get(nodeId);

    // Skip if already loaded or loading
    if (current?.isLoaded || current?.isLoading) {
      return;
    }

    // Mark as loading
    set((state) => {
      const newData = new Map(state.data);
      newData.set(nodeId, {
        ...(newData.get(nodeId) || emptyNodeData),
        isLoading: true,
      });
      return { data: newData, version: state.version + 1 };
    });

    try {
      // Fetch full data in parallel
      const [executionsResult, queueItemsResult, eventsResult]: [
        WorkflowsListNodeExecutionsResponse,
        WorkflowsListNodeQueueItemsResponse,
        WorkflowsListNodeEventsResponse,
      ] = await Promise.all([
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeExecutionsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ executions: [] }),
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeQueueItemsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ items: [] }),
        nodeType === "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeEventsQueryOptions(workflowId, nodeId, { limit: 10 }))
          : Promise.resolve({ events: [] }),
      ]);

      // Update with full data
      set((state) => {
        const newData = new Map(state.data);
        newData.set(nodeId, {
          executions: executionsResult.executions || [],
          queueItems: queueItemsResult.items || [],
          events: eventsResult.events || [],
          totalInHistoryCount: nodeType === "TYPE_TRIGGER" ? eventsResult?.totalCount || 0 : executionsResult?.totalCount || 0,
          totalInQueueCount: queueItemsResult?.totalCount || 0,
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData, version: state.version + 1 };
      });
    } catch (error) {
      console.error("Failed to load node data:", error);

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
      // Fetch fresh data in parallel
      const [executionsResult, queueItemsResult, eventsResult]: [
        WorkflowsListNodeExecutionsResponse,
        WorkflowsListNodeQueueItemsResponse,
        WorkflowsListNodeEventsResponse,
      ] = await Promise.all([
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeExecutionsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ executions: [] }),
        nodeType !== "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeQueueItemsQueryOptions(workflowId, nodeId))
          : Promise.resolve({ items: [] }),
        nodeType === "TYPE_TRIGGER"
          ? queryClient.fetchQuery(nodeEventsQueryOptions(workflowId, nodeId, { limit: 10 }))
          : Promise.resolve({ events: [] }),
      ]);

      // Update with fresh data
      set((state) => {
        const newData = new Map(state.data);
        newData.set(nodeId, {
          executions: executionsResult.executions || [],
          queueItems: queueItemsResult.items || [], 
          events: eventsResult.events || [],
          totalInHistoryCount: nodeType !== "TYPE_TRIGGER" ? executionsResult?.totalCount || 0 : eventsResult?.totalCount || 0,
          totalInQueueCount: queueItemsResult?.totalCount || 0,
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData, version: state.version + 1 };
      });
    } catch (error) {
      console.error("Failed to refetch node data:", error);

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

      const updatedExecutions = execution.parentExecutionId
        ? updateChildExecution(existing.executions, execution)
        : updateParentExecution(existing.executions, execution);

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
