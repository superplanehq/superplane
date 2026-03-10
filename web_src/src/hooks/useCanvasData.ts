import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from "@tanstack/react-query";
import {
  canvasesListCanvases,
  canvasesDescribeCanvas,
  canvasesDescribeCanvasVersion,
  canvasesCreateCanvas,
  canvasesUpdateCanvas,
  canvasesCreateCanvasVersion,
  canvasesListCanvasVersions,
  canvasesUpdateCanvasVersion,
  canvasesUpdateCanvasVersion2,
  canvasesCreateCanvasChangeRequest,
  canvasesActOnCanvasChangeRequest,
  canvasesResolveCanvasChangeRequest,
  canvasesListCanvasChangeRequests,
  canvasesDescribeCanvasChangeRequest,
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
  versions: () => [...canvasKeys.all, "versions"] as const,
  versionList: (canvasId: string) => [...canvasKeys.versions(), canvasId] as const,
  versionHistory: (canvasId: string) => [...canvasKeys.versions(), canvasId, "history"] as const,
  versionDetails: () => [...canvasKeys.versions(), "detail"] as const,
  versionDetail: (canvasId: string, versionId: string) =>
    [...canvasKeys.versionDetails(), canvasId, versionId] as const,
  changeRequests: () => [...canvasKeys.all, "changeRequests"] as const,
  changeRequestList: (canvasId: string) => [...canvasKeys.changeRequests(), canvasId] as const,
  changeRequestHistory: (
    canvasId: string,
    statusFilter: string,
    onlyMine: boolean,
    searchQuery: string,
    limit: number,
  ) =>
    [
      ...canvasKeys.changeRequestList(canvasId),
      "history",
      statusFilter,
      onlyMine ? "mine" : "all",
      searchQuery,
      limit,
    ] as const,
  changeRequestDetails: () => [...canvasKeys.changeRequests(), "detail"] as const,
  changeRequestDetail: (canvasId: string, changeRequestId: string) =>
    [...canvasKeys.changeRequestDetails(), canvasId, changeRequestId] as const,
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

export const useCanvasVersions = (organizationId: string, canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.versionList(canvasId),
    queryFn: async () => {
      const response = await canvasesListCanvasVersions(
        withOrganizationHeader({
          path: { canvasId },
          query: { limit: 1 },
        }),
      );
      return response.data?.versions || [];
    },
    enabled: !!organizationId && !!canvasId,
  });
};

export const useInfiniteCanvasLiveVersions = (
  organizationId: string,
  canvasId: string,
  enabled: boolean = true,
  limit: number = 20,
) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.versionHistory(canvasId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListCanvasVersions(
        withOrganizationHeader({
          path: { canvasId },
          query: {
            limit,
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const loadedPublishedCount = allPages.reduce(
        (acc, page) => acc + (page?.versions?.filter((version) => version.metadata?.isPublished).length || 0),
        0,
      );
      const totalCount = lastPage?.totalCount || 0;

      if (loadedPublishedCount >= totalCount) return undefined;
      if (!lastPage?.hasNextPage) return undefined;
      return lastPage?.lastTimestamp || undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!organizationId && !!canvasId,
    refetchOnWindowFocus: false,
  });
};

export const useCanvasVersion = (organizationId: string, canvasId: string, versionId: string, enabled = true) => {
  return useQuery({
    queryKey: canvasKeys.versionDetail(canvasId, versionId),
    queryFn: async () => {
      const response = await canvasesDescribeCanvasVersion(
        withOrganizationHeader({
          path: { canvasId, versionId },
        }),
      );
      return response.data?.version;
    },
    enabled: !!organizationId && !!canvasId && !!versionId && enabled,
  });
};

export const useCanvasChangeRequests = (organizationId: string, canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.changeRequestList(canvasId),
    queryFn: async () => {
      const response = await canvasesListCanvasChangeRequests(
        withOrganizationHeader({
          path: { canvasId },
          query: { limit: 100, statusFilter: "all" },
        }),
      );
      return response.data?.changeRequests || [];
    },
    enabled: !!organizationId && !!canvasId,
  });
};

type CanvasChangeRequestFilter = "open" | "rejected" | "merged" | "all";

const versionSortTimestamp = (version: any): number => {
  const raw = version?.metadata?.publishedAt || version?.metadata?.updatedAt || version?.metadata?.createdAt;
  if (!raw) return 0;
  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? 0 : parsed;
};

export const useInfiniteCanvasChangeRequests = (
  organizationId: string,
  canvasId: string,
  options?: {
    enabled?: boolean;
    limit?: number;
    statusFilter?: CanvasChangeRequestFilter;
    onlyMine?: boolean;
    searchQuery?: string;
  },
) => {
  const limit = options?.limit || 10;
  const statusFilter = options?.statusFilter || "open";
  const onlyMine = options?.onlyMine || false;
  const searchQuery = options?.searchQuery || "";
  const enabled = options?.enabled ?? true;

  return useInfiniteQuery({
    queryKey: canvasKeys.changeRequestHistory(canvasId, statusFilter, onlyMine, searchQuery, limit),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListCanvasChangeRequests(
        withOrganizationHeader({
          path: { canvasId },
          query: {
            limit,
            statusFilter,
            onlyMine,
            ...(searchQuery ? { query: searchQuery } : {}),
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const loadedCount = allPages.reduce((acc, page) => acc + (page?.changeRequests?.length || 0), 0);
      const totalCount = lastPage?.totalCount || 0;
      if (loadedCount >= totalCount) return undefined;
      if (!lastPage?.hasNextPage) return undefined;
      return lastPage?.lastTimestamp || undefined;
    },
    initialPageParam: undefined as string | undefined,
    enabled: enabled && !!organizationId && !!canvasId,
    refetchOnWindowFocus: false,
  });
};

export const useCanvasChangeRequest = (
  organizationId: string,
  canvasId: string,
  changeRequestId: string,
  enabled = true,
) => {
  return useQuery({
    queryKey: canvasKeys.changeRequestDetail(canvasId, changeRequestId),
    queryFn: async () => {
      const response = await canvasesDescribeCanvasChangeRequest(
        withOrganizationHeader({
          path: { canvasId, changeRequestId },
        }),
      );
      return response.data?.changeRequest;
    },
    enabled: !!organizationId && !!canvasId && !!changeRequestId && enabled,
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
    mutationFn: async (data: {
      name?: string;
      description?: string;
      canvasVersioningEnabled?: boolean;
      changeRequestApprovalConfig?: {
        items?: Array<{ type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE"; userId?: string; roleName?: string }>;
      };
    }) => {
      return await canvasesUpdateCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
          body: {
            name: data.name,
            description: data.description,
            canvasVersioningEnabled: data.canvasVersioningEnabled,
            changeRequestApprovalConfig: data.changeRequestApprovalConfig,
          },
        }),
      );
    },
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });

      const updatedCanvas = response?.data?.canvas;
      if (updatedCanvas) {
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), (current: any | undefined) => {
          if (!current) {
            return current;
          }

          const updatedMetadata = updatedCanvas.metadata;

          return {
            ...current,
            metadata: {
              ...current.metadata,
              name: updatedMetadata?.name ?? variables.name ?? current.metadata?.name,
              description: updatedMetadata?.description ?? variables.description ?? current.metadata?.description,
              canvasVersioningEnabled:
                updatedMetadata?.canvasVersioningEnabled ??
                variables.canvasVersioningEnabled ??
                current.metadata?.canvasVersioningEnabled,
              changeRequestApprovalConfig:
                updatedMetadata?.changeRequestApprovalConfig ??
                variables.changeRequestApprovalConfig ??
                current.metadata?.changeRequestApprovalConfig,
            },
          };
        });
      }
    },
  });
};

export const useCreateCanvasVersion = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await canvasesCreateCanvasVersion(
        withOrganizationHeader({
          path: { canvasId },
          body: {},
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
    },
  });
};

export const useUpdateCanvasVersion = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      versionId?: string;
      name: string;
      description?: string;
      nodes?: any[];
      edges?: any[];
      autoLayout?: { algorithm?: string; scope?: string; nodeIds?: string[] };
    }) => {
      const body = {
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
        autoLayout: data.autoLayout,
      };

      if (data.versionId) {
        return await canvasesUpdateCanvasVersion(
          withOrganizationHeader({
            path: { canvasId, versionId: data.versionId },
            body,
          }),
        );
      }

      return await canvasesUpdateCanvasVersion2(
        withOrganizationHeader({
          path: { canvasId },
          body,
        }),
      );
    },
    onSuccess: (response, variables) => {
      const version = response?.data?.version;
      if (!version) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        return;
      }

      if (variables.versionId) {
        queryClient.setQueryData(canvasKeys.versionDetail(canvasId, variables.versionId), version);
      }

      queryClient.setQueryData(canvasKeys.versionList(canvasId), (current: any[] | undefined) => {
        if (!current) {
          return current;
        }

        let found = false;
        const next = current.map((item) => {
          if (item?.metadata?.id === version.metadata?.id) {
            found = true;
            return version;
          }
          return item;
        });

        if (!found) {
          next.unshift(version);
        }

        next.sort((left, right) => versionSortTimestamp(right) - versionSortTimestamp(left));
        return next;
      });

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), (current: any | undefined) => {
        if (!current) {
          return current;
        }

        return {
          ...current,
          spec: version.spec,
        };
      });

      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequests() });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
    },
  });
};

export const useCreateCanvasChangeRequest = (_organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { versionId?: string; title?: string; description?: string }) => {
      return await canvasesCreateCanvasChangeRequest(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            title: data.title,
            description: data.description,
            ...(data.versionId ? { versionId: data.versionId } : {}),
          },
        }),
      );
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });

      const changeRequest = response?.data?.changeRequest;
      const changeRequestID = changeRequest?.metadata?.id;
      if (changeRequest && changeRequestID) {
        queryClient.setQueryData(canvasKeys.changeRequestDetail(canvasId, changeRequestID), changeRequest);
      }
    },
  });
};

export const useActOnCanvasChangeRequest = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      changeRequestId: string;
      action: "ACTION_APPROVE" | "ACTION_UNAPPROVE" | "ACTION_REJECT" | "ACTION_REOPEN" | "ACTION_PUBLISH";
    }) => {
      return await canvasesActOnCanvasChangeRequest(
        withOrganizationHeader({
          path: { canvasId, changeRequestId: data.changeRequestId },
          body: {
            action: data.action,
          },
        }),
      );
    },
    onSuccess: (_response, variables) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequests() });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });
      queryClient.removeQueries({ queryKey: canvasKeys.changeRequestDetail(canvasId, variables.changeRequestId) });
    },
  });
};

export const useResolveCanvasChangeRequest = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      changeRequestId: string;
      name: string;
      description?: string;
      nodes?: any[];
      edges?: any[];
      autoLayout?: { algorithm?: string; scope?: string; nodeIds?: string[] };
    }) => {
      return await canvasesResolveCanvasChangeRequest(
        withOrganizationHeader({
          path: { canvasId, changeRequestId: data.changeRequestId },
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
            autoLayout: data.autoLayout,
          },
        }),
      );
    },
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequests() });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });

      const version = response?.data?.version;
      if (version?.metadata?.id) {
        queryClient.setQueryData(canvasKeys.versionDetail(canvasId, version.metadata.id), version);
      }

      const changeRequest = response?.data?.changeRequest;
      if (changeRequest?.metadata?.id) {
        queryClient.setQueryData(canvasKeys.changeRequestDetail(canvasId, changeRequest.metadata.id), changeRequest);
      } else {
        queryClient.removeQueries({ queryKey: canvasKeys.changeRequestDetail(canvasId, variables.changeRequestId) });
      }
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

export const useCanvasEvents = (canvasId: string, enabled = true) => {
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
    enabled: !!canvasId && enabled,
  });
};

export interface CanvasMemoryEntry {
  id: string;
  namespace: string;
  values: unknown;
}

export const useCanvasMemoryEntries = (canvasId: string, enabled = true) => {
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
    enabled: !!canvasId && enabled,
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
