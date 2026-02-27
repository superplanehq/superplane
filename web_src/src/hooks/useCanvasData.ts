import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from "@tanstack/react-query";
import {
  canvasesListCanvases,
  canvasesDescribeCanvas,
  canvasesCreateCanvas,
  canvasesUpdateCanvas,
  canvasesDeleteCanvas,
  canvasesListNodeExecutions,
  canvasesListCanvasEvents,
  canvasesListCanvasMemories,
  canvasesDeleteCanvasMemory,
  canvasesListEventExecutions,
  canvasesListChildExecutions,
  canvasesListNodeQueueItems,
  canvasesListNodeEvents,
  triggersListTriggers,
  triggersDescribeTrigger,
  widgetsListWidgets,
  widgetsDescribeWidget,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../utils/withOrganizationHeader";

// Query Keys
export const canvasKeys = {
  all: ["canvases"] as const,
  lists: () => [...canvasKeys.all, "list"] as const,
  list: (orgId: string) => [...canvasKeys.lists(), orgId] as const,
  templates: () => [...canvasKeys.all, "templates"] as const,
  templateList: (orgId: string) => [...canvasKeys.templates(), orgId] as const,
  details: () => [...canvasKeys.all, "detail"] as const,
  detail: (orgId: string, id: string) => [...canvasKeys.details(), orgId, id] as const,
  nodeExecutions: () => [...canvasKeys.all, "nodeExecutions"] as const,
  nodeExecution: (canvasId: string, nodeId: string, states?: string[]) =>
    [...canvasKeys.nodeExecutions(), canvasId, nodeId, ...(states || [])] as const,
  events: () => [...canvasKeys.all, "events"] as const,
  eventList: (canvasId: string, limit?: number) => [...canvasKeys.events(), canvasId, limit] as const,
  eventExecutions: () => [...canvasKeys.all, "eventExecutions"] as const,
  eventExecution: (canvasId: string, eventId: string) => [...canvasKeys.eventExecutions(), canvasId, eventId] as const,
  childExecutions: () => [...canvasKeys.all, "childExecutions"] as const,
  childExecution: (canvasId: string, executionId: string) =>
    [...canvasKeys.childExecutions(), canvasId, executionId] as const,
  nodeQueueItems: () => [...canvasKeys.all, "nodeQueueItems"] as const,
  nodeQueueItem: (canvasId: string, nodeId: string) => [...canvasKeys.nodeQueueItems(), canvasId, nodeId] as const,
  nodeEvents: () => [...canvasKeys.all, "nodeEvents"] as const,
  nodeEvent: (canvasId: string, nodeId: string, limit?: number) =>
    [...canvasKeys.nodeEvents(), canvasId, nodeId, limit] as const,
  nodeEventHistory: (canvasId: string, nodeId: string) =>
    [...canvasKeys.nodeEvents(), "infinite", canvasId, nodeId] as const,
  nodeExecutionHistory: (canvasId: string, nodeId: string) =>
    [...canvasKeys.nodeExecutions(), "infinite", canvasId, nodeId] as const,
  nodeQueueItemHistory: (canvasId: string, nodeId: string) =>
    [...canvasKeys.nodeQueueItems(), "infinite", canvasId, nodeId] as const,
  canvasMemoryEntries: (canvasId: string) => [...canvasKeys.all, "memoryEntries", canvasId] as const,
};

export const triggerKeys = {
  all: ["triggers"] as const,
  lists: () => [...triggerKeys.all, "list"] as const,
  list: () => [...triggerKeys.lists()] as const,
  details: () => [...triggerKeys.all, "detail"] as const,
  detail: (name: string) => [...triggerKeys.details(), name] as const,
};

export const widgetKeys = {
  all: ["widgets"] as const,
  lists: () => [...widgetKeys.all, "list"] as const,
  list: () => [...widgetKeys.lists()] as const,
  details: () => [...widgetKeys.all, "detail"] as const,
  detail: (name: string) => [...widgetKeys.details(), name] as const,
};

// Hooks for fetching canvases
export const useCanvases = (organizationId: string) => {
  return useQuery({
    queryKey: canvasKeys.list(organizationId),
    queryFn: async () => {
      const response = await canvasesListCanvases(
        withOrganizationHeader({
          query: { includeTemplates: false },
        }),
      );
      return response.data?.canvases || [];
    },
    enabled: !!organizationId,
  });
};

export const useCanvasTemplates = (organizationId: string) => {
  return useQuery({
    queryKey: canvasKeys.templateList(organizationId),
    queryFn: async () => {
      const response = await canvasesListCanvases(
        withOrganizationHeader({
          query: { includeTemplates: true },
        }),
      );
      const canvases = response.data?.canvases || [];
      return canvases.filter((canvas) => canvas.metadata?.isTemplate);
    },
    enabled: !!organizationId,
  });
};

export const useCanvas = (organizationId: string, canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.detail(organizationId, canvasId),
    queryFn: async () => {
      const response = await canvasesDescribeCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
        }),
      );
      return response.data?.canvas;
    },
    enabled: !!organizationId && !!canvasId,
  });
};

export const useCreateCanvas = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      const payload = {
        metadata: {
          name: data.name,
          description: data.description || "",
        },
        spec: {
          nodes: data.nodes || [],
          edges: data.edges || [],
        },
      };

      return await canvasesCreateCanvas(
        withOrganizationHeader({
          body: {
            canvas: payload,
          },
        }),
      );
    },
    onSuccess: (response) => {
      // Invalidate the list to refresh the canvas list
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });

      // Set the workflow detail in cache immediately so it's available when navigating
      if (response?.data?.canvas?.metadata?.id) {
        queryClient.setQueryData(
          canvasKeys.detail(organizationId, response.data.canvas.metadata.id),
          response.data.canvas,
        );
      }
    },
  });
};

export const useUpdateCanvas = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      return await canvasesUpdateCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
          body: {
            canvas: {
              metadata: {
                name: data.name,
                description: data.description || "",
              },
              spec: {
                nodes: data.nodes || [],
                edges: data.edges || [],
              },
            },
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
    },
  });
};

export const useDeleteCanvas = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (canvasId: string) => {
      // Remove from cache immediately before deletion to prevent 404 flash
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });

      return await canvasesDeleteCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
        }),
      );
    },
    onSuccess: (_, canvasId) => {
      // Ensure it's removed (in case it wasn't already)
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      // Invalidate the list to refresh the canvas list
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    },
  });
};

export const useNodeExecutions = (
  canvasId: string,
  nodeId: string,
  options?: {
    states?: string[];
  },
) => {
  return useQuery({
    queryKey: canvasKeys.nodeExecution(canvasId, nodeId, options?.states),
    queryFn: async () => {
      const response = await canvasesListNodeExecutions(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId,
          },
          query: options?.states
            ? {
                states: options.states,
              }
            : undefined,
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId && !!nodeId,
  });
};

export const useCanvasEvents = (canvasId: string) => {
  const limit = 50;

  return useQuery({
    queryKey: canvasKeys.eventList(canvasId, limit),
    queryFn: async () => {
      const response = await canvasesListCanvasEvents(
        withOrganizationHeader({
          path: { canvasId },
          query: { limit },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId,
  });
};

export interface CanvasMemoryEntry {
  id: string;
  namespace: string;
  values: unknown;
}

export const useCanvasMemoryEntries = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.canvasMemoryEntries(canvasId),
    queryFn: async () => {
      const response = await canvasesListCanvasMemories(
        withOrganizationHeader({
          path: { canvasId },
        }),
      );
      const items = response.data?.items || [];
      return items.map((item) => ({
        id: item.id || "",
        namespace: item.namespace || "",
        values: item.values,
      }));
    },
    refetchOnWindowFocus: false,
    refetchInterval: 3000,
    enabled: !!canvasId,
  });
};

export const useDeleteCanvasMemoryEntry = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (memoryId: string) => {
      await canvasesDeleteCanvasMemory(
        withOrganizationHeader({
          path: {
            canvasId,
            memoryId,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.canvasMemoryEntries(canvasId) });
    },
  });
};

export const useEventExecutions = (canvasId: string, eventId: string | null) => {
  return useQuery({
    queryKey: canvasKeys.eventExecution(canvasId, eventId!),
    queryFn: async () => {
      const response = await canvasesListEventExecutions(
        withOrganizationHeader({
          path: {
            canvasId,
            eventId: eventId!,
          },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId && !!eventId,
  });
};

export const useChildExecutions = (canvasId: string, executionId: string | null) => {
  return useQuery({
    queryKey: canvasKeys.childExecution(canvasId, executionId!),
    queryFn: async () => {
      const response = await canvasesListChildExecutions(
        withOrganizationHeader({
          path: {
            canvasId,
            executionId: executionId!,
          },
          body: {},
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId && !!executionId,
  });
};

export const useNodeQueueItems = (canvasId: string, nodeId: string) => {
  return useQuery({
    queryKey: canvasKeys.nodeQueueItem(canvasId, nodeId),
    queryFn: async () => {
      const response = await canvasesListNodeQueueItems(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId,
          },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId && !!nodeId,
  });
};

export const nodeEventsQueryOptions = (
  canvasId: string,
  nodeId: string,
  options?: {
    limit?: number;
  },
) => ({
  queryKey: canvasKeys.nodeEvent(canvasId, nodeId, options?.limit),
  queryFn: async () => {
    const response = await canvasesListNodeEvents(
      withOrganizationHeader({
        path: {
          canvasId,
          nodeId,
        },
        query: options?.limit
          ? {
              limit: options.limit,
            }
          : undefined,
      }),
    );
    return response.data;
  },
  refetchOnWindowFocus: false,
  enabled: !!canvasId && !!nodeId,
});

export const nodeExecutionsQueryOptions = (
  canvasId: string,
  nodeId: string,
  options?: {
    states?: string[];
    limit?: number;
  },
) => ({
  queryKey: canvasKeys.nodeExecution(canvasId, nodeId, options?.states),
  queryFn: async () => {
    const response = await canvasesListNodeExecutions(
      withOrganizationHeader({
        path: {
          canvasId,
          nodeId,
        },
        query: options?.states
          ? {
              states: options.states,
            }
          : undefined,
      }),
    );
    return response.data;
  },
  refetchOnWindowFocus: false,
  enabled: !!canvasId && !!nodeId,
});

export const nodeQueueItemsQueryOptions = (canvasId: string, nodeId: string) => ({
  queryKey: canvasKeys.nodeQueueItem(canvasId, nodeId),
  queryFn: async () => {
    const response = await canvasesListNodeQueueItems(
      withOrganizationHeader({
        path: {
          canvasId,
          nodeId,
        },
      }),
    );
    return response.data;
  },
  refetchOnWindowFocus: false,
  enabled: !!canvasId && !!nodeId,
});

export const eventExecutionsQueryOptions = (canvasId: string, eventId: string) => ({
  queryKey: canvasKeys.eventExecution(canvasId, eventId),
  queryFn: async () => {
    const response = await canvasesListEventExecutions(
      withOrganizationHeader({
        path: {
          canvasId,
          eventId,
        },
      }),
    );
    return response.data;
  },
  staleTime: 30 * 1000, // 30 seconds
  gcTime: 5 * 60 * 1000, // 5 minutes
  enabled: !!canvasId && !!eventId,
});

export const useNodeEvents = (canvasId: string, nodeId: string) => {
  return useQuery(nodeEventsQueryOptions(canvasId, nodeId));
};

// Hooks for fetching triggers
export const useTriggers = () => {
  return useQuery({
    queryKey: triggerKeys.list(),
    queryFn: async () => {
      const response = await triggersListTriggers(withOrganizationHeader({}));
      return response.data?.triggers || [];
    },
  });
};

export const useTrigger = (triggerName: string) => {
  return useQuery({
    queryKey: triggerKeys.detail(triggerName),
    queryFn: async () => {
      const response = await triggersDescribeTrigger(
        withOrganizationHeader({
          path: { name: triggerName },
        }),
      );
      return response.data?.trigger;
    },
    enabled: !!triggerName,
  });
};

// Hooks for fetching widgets
export const useWidgets = () => {
  return useQuery({
    queryKey: widgetKeys.list(),
    queryFn: async () => {
      const response = await widgetsListWidgets(withOrganizationHeader({}));
      return response.data?.widgets || [];
    },
  });
};

export const useWidget = (widgetName: string) => {
  return useQuery({
    queryKey: widgetKeys.detail(widgetName),
    queryFn: async () => {
      const response = await widgetsDescribeWidget(
        withOrganizationHeader({
          path: { name: widgetName },
        }),
      );
      return response.data?.widget;
    },
    enabled: !!widgetName,
  });
};

export const useInfiniteNodeEvents = (canvasId: string, nodeId: string, enabled: boolean = false) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.nodeEventHistory(canvasId, nodeId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListNodeEvents(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId,
          },
          query: {
            limit: 10,
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const currentLoadedCount = allPages.reduce((acc, page) => acc + (page?.events?.length || 0), 0);
      const totalCount = lastPage?.totalCount || 0;

      if (currentLoadedCount >= totalCount) return undefined;

      if (lastPage?.events && lastPage.events.length > 0) {
        const lastEvent = lastPage.events[lastPage.events.length - 1];
        return lastEvent.createdAt;
      }
      return undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!canvasId && !!nodeId,
    refetchOnWindowFocus: false,
  });
};

export const useInfiniteNodeExecutions = (canvasId: string, nodeId: string, enabled: boolean = false) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.nodeExecutionHistory(canvasId, nodeId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListNodeExecutions(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId,
          },
          query: {
            limit: 10,
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const currentLoadedCount = allPages.reduce((acc, page) => acc + (page?.executions?.length || 0), 0);
      const totalCount = lastPage?.totalCount || 0;

      if (currentLoadedCount >= totalCount) return undefined;

      if (lastPage?.executions && lastPage.executions.length > 0) {
        const lastExecution = lastPage.executions[lastPage.executions.length - 1];
        return lastExecution.createdAt;
      }
      return undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!canvasId && !!nodeId,
    refetchOnWindowFocus: false,
  });
};

export const useInfiniteNodeQueueItems = (canvasId: string, nodeId: string, enabled: boolean = false) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.nodeQueueItemHistory(canvasId, nodeId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListNodeQueueItems(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId,
          },
          query: {
            limit: 10,
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const currentLoadedCount = allPages.reduce((acc, page) => acc + (page?.items?.length || 0), 0);
      const totalCount = lastPage?.totalCount || 0;

      if (currentLoadedCount >= totalCount) return undefined;

      if (lastPage?.items && lastPage.items.length > 0) {
        const lastQueueItem = lastPage.items[lastPage.items.length - 1];
        return lastQueueItem.createdAt;
      }
      return undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!canvasId && !!nodeId,
    refetchOnWindowFocus: false,
  });
};
