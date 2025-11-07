import { create } from 'zustand';
import { QueryClient } from '@tanstack/react-query';
import {
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflow,
} from '@/api-client';
import {
  nodeExecutionsQueryOptions,
  nodeQueueItemsQueryOptions,
  nodeEventsQueryOptions,
} from '@/hooks/useWorkflowData';

interface NodeExecutionData {
  executions: WorkflowsWorkflowNodeExecution[];
  queueItems: WorkflowsWorkflowNodeQueueItem[];
  events: WorkflowsWorkflowEvent[];
  isLoading: boolean;
  isLoaded: boolean;
}

interface NodeExecutionStore {
  // State
  data: Map<string, NodeExecutionData>;

  // Actions
  initializeFromWorkflow: (workflow: WorkflowsWorkflow) => void;
  loadNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  refetchNodeData: (workflowId: string, nodeId: string, nodeType: string, queryClient: QueryClient) => Promise<void>;
  getNodeData: (nodeId: string) => NodeExecutionData;
  clear: () => void;
}

const emptyNodeData: NodeExecutionData = {
  executions: [],
  queueItems: [],
  events: [],
  isLoading: false,
  isLoaded: false,
};

export const useNodeExecutionStore = create<NodeExecutionStore>((set, get) => ({
  data: new Map(),

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

    set({ data: initialData });
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
      return { data: newData };
    });

    try {
      // Fetch full data in parallel
      const [executionsResult, queueItemsResult, eventsResult] = await Promise.all([
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
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData };
      });
    } catch (error) {
      console.error('Failed to load node data:', error);

      // Mark as not loading on error
      set((state) => {
        const newData = new Map(state.data);
        const existing = newData.get(nodeId);
        if (existing) {
          newData.set(nodeId, { ...existing, isLoading: false });
        }
        return { data: newData };
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
      return { data: newData };
    });

    try {
      // Fetch fresh data in parallel
      const [executionsResult, queueItemsResult, eventsResult] = await Promise.all([
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
          isLoading: false,
          isLoaded: true,
        });
        return { data: newData };
      });
    } catch (error) {
      console.error('Failed to refetch node data:', error);

      // Mark as not loading on error
      set((state) => {
        const newData = new Map(state.data);
        const existing = newData.get(nodeId);
        if (existing) {
          newData.set(nodeId, { ...existing, isLoading: false });
        }
        return { data: newData };
      });
    }
  },

  getNodeData: (nodeId) => {
    return get().data.get(nodeId) || emptyNodeData;
  },

  clear: () => {
    set({ data: new Map() });
  },
}));
