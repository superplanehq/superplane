import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from "@tanstack/react-query";
import {
  workflowsListWorkflows,
  workflowsDescribeWorkflow,
  workflowsCreateWorkflow,
  workflowsUpdateWorkflow,
  workflowsDeleteWorkflow,
  workflowsListNodeExecutions,
  workflowsListWorkflowEvents,
  workflowsListEventExecutions,
  workflowsListChildExecutions,
  workflowsListNodeQueueItems,
  workflowsListNodeEvents,
  triggersListTriggers,
  triggersDescribeTrigger,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../utils/withOrganizationHeader";

// Query Keys
export const workflowKeys = {
  all: ["workflows"] as const,
  lists: () => [...workflowKeys.all, "list"] as const,
  list: (orgId: string) => [...workflowKeys.lists(), orgId] as const,
  details: () => [...workflowKeys.all, "detail"] as const,
  detail: (orgId: string, id: string) => [...workflowKeys.details(), orgId, id] as const,
  nodeExecutions: () => [...workflowKeys.all, "nodeExecutions"] as const,
  nodeExecution: (workflowId: string, nodeId: string, states?: string[]) =>
    [...workflowKeys.nodeExecutions(), workflowId, nodeId, ...(states || [])] as const,
  events: () => [...workflowKeys.all, "events"] as const,
  eventList: (workflowId: string) => [...workflowKeys.events(), workflowId] as const,
  eventExecutions: () => [...workflowKeys.all, "eventExecutions"] as const,
  eventExecution: (workflowId: string, eventId: string) =>
    [...workflowKeys.eventExecutions(), workflowId, eventId] as const,
  childExecutions: () => [...workflowKeys.all, "childExecutions"] as const,
  childExecution: (workflowId: string, executionId: string) =>
    [...workflowKeys.childExecutions(), workflowId, executionId] as const,
  nodeQueueItems: () => [...workflowKeys.all, "nodeQueueItems"] as const,
  nodeQueueItem: (workflowId: string, nodeId: string) =>
    [...workflowKeys.nodeQueueItems(), workflowId, nodeId] as const,
  nodeEvents: () => [...workflowKeys.all, "nodeEvents"] as const,
  nodeEvent: (workflowId: string, nodeId: string, limit?: number) =>
    [...workflowKeys.nodeEvents(), workflowId, nodeId, limit] as const,
  nodeEventHistory: (workflowId: string, nodeId: string) =>
    [...workflowKeys.nodeEvents(), "infinite", workflowId, nodeId] as const,
  nodeExecutionHistory: (workflowId: string, nodeId: string) =>
    [...workflowKeys.nodeExecutions(), "infinite", workflowId, nodeId] as const,
};

export const triggerKeys = {
  all: ["triggers"] as const,
  lists: () => [...triggerKeys.all, "list"] as const,
  list: () => [...triggerKeys.lists()] as const,
  details: () => [...triggerKeys.all, "detail"] as const,
  detail: (name: string) => [...triggerKeys.details(), name] as const,
};

// Hooks for fetching workflows
export const useWorkflows = (organizationId: string) => {
  return useQuery({
    queryKey: workflowKeys.list(organizationId),
    queryFn: async () => {
      const response = await workflowsListWorkflows(withOrganizationHeader({}));
      return response.data?.workflows || [];
    },
    enabled: !!organizationId,
  });
};

export const useWorkflow = (organizationId: string, workflowId: string) => {
  return useQuery({
    queryKey: workflowKeys.detail(organizationId, workflowId),
    queryFn: async () => {
      const response = await workflowsDescribeWorkflow(
        withOrganizationHeader({
          path: { id: workflowId },
        }),
      );
      return response.data?.workflow;
    },
    enabled: !!organizationId && !!workflowId,
  });
};

export const useCreateWorkflow = (organizationId: string) => {
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

      return await workflowsCreateWorkflow(
        withOrganizationHeader({
          body: {
            workflow: payload,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowKeys.list(organizationId) });
    },
  });
};

export const useUpdateWorkflow = (organizationId: string, workflowId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      return await workflowsUpdateWorkflow(
        withOrganizationHeader({
          path: { id: workflowId },
          body: {
            workflow: {
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
      queryClient.invalidateQueries({ queryKey: workflowKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: workflowKeys.detail(organizationId, workflowId) });
    },
  });
};

export const useDeleteWorkflow = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (workflowId: string) => {
      return await workflowsDeleteWorkflow(
        withOrganizationHeader({
          path: { id: workflowId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowKeys.list(organizationId) });
    },
  });
};

export const useNodeExecutions = (
  workflowId: string,
  nodeId: string,
  options?: {
    states?: string[];
  },
) => {
  return useQuery({
    queryKey: workflowKeys.nodeExecution(workflowId, nodeId, options?.states),
    queryFn: async () => {
      const response = await workflowsListNodeExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
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
    enabled: !!workflowId && !!nodeId,
  });
};

export const useWorkflowEvents = (workflowId: string) => {
  return useQuery({
    queryKey: workflowKeys.eventList(workflowId),
    queryFn: async () => {
      const response = await workflowsListWorkflowEvents(
        withOrganizationHeader({
          path: { workflowId },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!workflowId,
  });
};

export const useEventExecutions = (workflowId: string, eventId: string | null) => {
  return useQuery({
    queryKey: workflowKeys.eventExecution(workflowId, eventId!),
    queryFn: async () => {
      const response = await workflowsListEventExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
            eventId: eventId!,
          },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!workflowId && !!eventId,
  });
};

export const useChildExecutions = (workflowId: string, executionId: string | null) => {
  return useQuery({
    queryKey: workflowKeys.childExecution(workflowId, executionId!),
    queryFn: async () => {
      const response = await workflowsListChildExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
            executionId: executionId!,
          },
          body: {},
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!workflowId && !!executionId,
  });
};

export const useNodeQueueItems = (workflowId: string, nodeId: string) => {
  return useQuery({
    queryKey: workflowKeys.nodeQueueItem(workflowId, nodeId),
    queryFn: async () => {
      const response = await workflowsListNodeQueueItems(
        withOrganizationHeader({
          path: {
            workflowId,
            nodeId,
          },
        }),
      );
      return response.data;
    },
    refetchOnWindowFocus: false,
    enabled: !!workflowId && !!nodeId,
  });
};

export const nodeEventsQueryOptions = (
  workflowId: string,
  nodeId: string,
  options?: {
    limit?: number;
  },
) => ({
  queryKey: workflowKeys.nodeEvent(workflowId, nodeId, options?.limit),
  queryFn: async () => {
    const response = await workflowsListNodeEvents(
      withOrganizationHeader({
        path: {
          workflowId,
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
  enabled: !!workflowId && !!nodeId,
});

export const nodeExecutionsQueryOptions = (
  workflowId: string,
  nodeId: string,
  options?: {
    states?: string[];
    limit?: number;
  },
) => ({
  queryKey: workflowKeys.nodeExecution(workflowId, nodeId, options?.states),
  queryFn: async () => {
    const response = await workflowsListNodeExecutions(
      withOrganizationHeader({
        path: {
          workflowId,
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
  enabled: !!workflowId && !!nodeId,
});

export const nodeQueueItemsQueryOptions = (workflowId: string, nodeId: string) => ({
  queryKey: workflowKeys.nodeQueueItem(workflowId, nodeId),
  queryFn: async () => {
    const response = await workflowsListNodeQueueItems(
      withOrganizationHeader({
        path: {
          workflowId,
          nodeId,
        },
      }),
    );
    return response.data;
  },
  refetchOnWindowFocus: false,
  enabled: !!workflowId && !!nodeId,
});

export const eventExecutionsQueryOptions = (workflowId: string, eventId: string) => ({
  queryKey: workflowKeys.eventExecution(workflowId, eventId),
  queryFn: async () => {
    const response = await workflowsListEventExecutions(
      withOrganizationHeader({
        path: {
          workflowId,
          eventId,
        },
      }),
    );
    return response.data;
  },
  staleTime: 30 * 1000, // 30 seconds
  gcTime: 5 * 60 * 1000, // 5 minutes
  enabled: !!workflowId && !!eventId,
});

export const useNodeEvents = (workflowId: string, nodeId: string) => {
  return useQuery(nodeEventsQueryOptions(workflowId, nodeId));
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

export const useInfiniteNodeEvents = (workflowId: string, nodeId: string, enabled: boolean = false) => {
  return useInfiniteQuery({
    queryKey: workflowKeys.nodeEventHistory(workflowId, nodeId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await workflowsListNodeEvents(
        withOrganizationHeader({
          path: {
            workflowId,
            nodeId,
          },
          query: {
            limit: 20,
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

      if (lastPage?.events && lastPage.events.length === 20) {
        const lastEvent = lastPage.events[lastPage.events.length - 1];
        return lastEvent.createdAt;
      }
      return undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!workflowId && !!nodeId,
    refetchOnWindowFocus: false,
  });
};

export const useInfiniteNodeExecutions = (workflowId: string, nodeId: string, enabled: boolean = false) => {
  return useInfiniteQuery({
    queryKey: workflowKeys.nodeExecutionHistory(workflowId, nodeId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await workflowsListNodeExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
            nodeId,
          },
          query: {
            limit: 20,
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

      if (lastPage?.executions && lastPage.executions.length === 20) {
        const lastExecution = lastPage.executions[lastPage.executions.length - 1];
        return lastExecution.createdAt;
      }
      return undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!workflowId && !!nodeId,
    refetchOnWindowFocus: false,
  });
};
