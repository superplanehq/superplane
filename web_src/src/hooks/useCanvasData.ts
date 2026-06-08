import { useQuery, useMutation, useQueryClient, useInfiniteQuery, useQueries } from "@tanstack/react-query";
import {
  canvasesListCanvases,
  canvasesDescribeCanvas,
  canvasesDescribeCanvasVersion,
  canvasesCreateCanvas,
  canvasesUpdateCanvas,
  canvasFoldersListCanvasFolders,
  canvasFoldersCreateCanvasFolder,
  canvasFoldersUpdateCanvasFolder,
  canvasFoldersUpdateCanvasFolderPosition,
  canvasFoldersDeleteCanvasFolder,
  canvasesCreateCanvasVersion,
  canvasesDeleteCanvasVersion,
  canvasesListCanvasVersions,
  canvasesCreateCanvasChangeRequest,
  canvasesActOnCanvasChangeRequest,
  canvasesResolveCanvasChangeRequest,
  canvasesListCanvasChangeRequests,
  canvasesDescribeCanvasChangeRequest,
  canvasesDeleteCanvas,
  canvasesPublishCanvasVersion,
  canvasesListNodeExecutions,
  canvasesListCanvasEvents,
  canvasesListRuns,
  canvasesListCanvasMemories,
  canvasesDeleteCanvasMemory,
  canvasesCreateCanvasMemoryNamespace,
  canvasesUpdateCanvasMemoryNamespace,
  canvasesListEventExecutions,
  canvasesListChildExecutions,
  canvasesListNodeQueueItems,
  canvasesListNodeEvents,
  canvasesGetCanvasRepository,
  canvasesListCanvasRepositoryFiles,
  canvasesCommitCanvasRepositoryFiles,
  triggersListTriggers,
  triggersDescribeTrigger,
  widgetsListWidgets,
  widgetsDescribeWidget,
} from "../api-client/sdk.gen";
import type {
  CanvasFoldersCanvasFolder,
  CanvasesCanvas,
  CanvasesCanvasRunResult,
  CanvasesCanvasRunState,
  CanvasesCanvasVersion,
  CanvasesCanvasRepositoryFileOperation,
  CanvasesListCanvasRepositoryFilesResponse,
  SuperplaneComponentsNode,
  ComponentsPosition,
} from "../api-client/types.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { analytics } from "../lib/analytics";
import { isPublishedVersion } from "../pages/app/lib/canvas-versions";
import {
  canvasVersionWithSpecFromYaml,
  fetchCanvasVersionWithSpec,
  fetchConsoleSpecFromRepository,
  fetchRepositorySpecFileContent,
} from "../pages/app/lib/repository-spec-files";
import { encodeRepositoryFileContent } from "../pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "../pages/app/lib/workflow-spec-paths";
import { dematerializeConsoleSpec, materializeConsoleSpec } from "../pages/app/lib/workflow-spec-files";

function versionWithSpecFromYaml(
  version: CanvasesCanvasVersion | undefined,
  canvasYaml: string | undefined,
): CanvasesCanvasVersion | undefined {
  return canvasVersionWithSpecFromYaml(version, canvasYaml);
}

export type CanvasConsoleData = {
  canvasId: string;
  versionId?: string;
  updatedAt?: string;
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  consoleYaml: string;
};

function consoleDataFromYaml(
  canvasId: string,
  versionId: string | undefined,
  consoleYaml: string,
): CanvasConsoleData | undefined {
  const parsed = dematerializeConsoleSpec(consoleYaml);
  if (!parsed) {
    return undefined;
  }

  return {
    canvasId,
    versionId,
    panels: parsed.panels,
    layout: parsed.layout,
    consoleYaml,
  };
}

// Query Keys
export const canvasKeys = {
  all: ["canvases"] as const,
  lists: () => [...canvasKeys.all, "list"] as const,
  list: (orgId: string) => [...canvasKeys.lists(), orgId] as const,
  folders: () => [...canvasKeys.all, "folders"] as const,
  folderList: (orgId: string) => [...canvasKeys.folders(), orgId] as const,
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
  draftBranches: (canvasId: string) => [...canvasKeys.all, "draftBranches", canvasId] as const,
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
  nodeExecution: (canvasId: string, nodeId: string, states?: string[], limit?: number) =>
    [
      ...canvasKeys.nodeExecutions(),
      canvasId,
      nodeId,
      ...(states || []),
      ...(limit === undefined ? [] : [limit]),
    ] as const,
  events: () => [...canvasKeys.all, "events"] as const,
  eventList: (canvasId: string, limit?: number) => [...canvasKeys.events(), canvasId, limit] as const,
  infiniteEvents: (canvasId: string) => [...canvasKeys.events(), canvasId, "infinite"] as const,
  runs: () => [...canvasKeys.all, "runs"] as const,
  infiniteRuns: (canvasId: string, filters?: CanvasRunsFilters) =>
    [
      ...canvasKeys.runs(),
      canvasId,
      "infinite",
      ...(filters?.states?.length ? ["states", ...filters.states] : []),
      ...(filters?.results?.length ? ["results", ...filters.results] : []),
    ] as const,
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
  console: (canvasId: string, versionId?: string) =>
    [...canvasKeys.all, "console", canvasId, versionId ?? "live"] as const,
  consoleAll: (canvasId: string) => [...canvasKeys.all, "console", canvasId] as const,
  repository: (canvasId: string) => [...canvasKeys.all, "repository", canvasId] as const,
  repositoryFiles: (canvasId: string) => [...canvasKeys.repository(canvasId), "files"] as const,
  repositoryFile: (canvasId: string, path: string, versionId?: string) =>
    [...canvasKeys.repository(canvasId), "file", path, versionId ?? "live"] as const,
};

export interface ConsolePanel {
  id: string;
  type: string;
  content: Record<string, unknown>;
}

export interface ConsoleLayoutItem {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
}

export const CANVAS_FOLDER_COLORS = ["blue", "green", "purple", "yellow", "slate", "orange"] as const;
export type CanvasFolderColor = (typeof CANVAS_FOLDER_COLORS)[number];
export const DEFAULT_CANVAS_FOLDER_COLOR: CanvasFolderColor = "blue";

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

export const NODE_EXECUTION_HISTORY_PAGE_SIZE = 10;

// Hooks for fetching canvases
export const useCanvases = (organizationId: string) => {
  return useQuery({
    queryKey: canvasKeys.list(organizationId),
    queryFn: async () => {
      const response = await canvasesListCanvases(
        withOrganizationHeader({
          organizationId,
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
          organizationId,
          query: { includeTemplates: true },
        }),
      );
      const canvases = response.data?.canvases || [];
      return canvases.filter((canvas) => canvas.metadata?.isTemplate);
    },
    enabled: !!organizationId,
  });
};

export const useCanvasFolders = (organizationId: string) => {
  return useQuery({
    queryKey: canvasKeys.folderList(organizationId),
    queryFn: async () => {
      const response = await canvasFoldersListCanvasFolders(withOrganizationHeader({ organizationId }));
      return response.data?.folders || [];
    },
    enabled: !!organizationId,
  });
};

type UseCanvasOptions = {
  enabled?: boolean;
  staleTime?: number;
  refetchOnWindowFocus?: boolean;
  refetchOnReconnect?: boolean;
  refetchOnMount?: boolean;
};

export const useCanvas = (organizationId: string, canvasId: string, options: UseCanvasOptions = {}) => {
  const {
    enabled = true,
    staleTime = 0,
    refetchOnWindowFocus = true,
    refetchOnReconnect = true,
    refetchOnMount = true,
  } = options;

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
    staleTime,
    refetchOnWindowFocus,
    refetchOnReconnect,
    refetchOnMount,
    enabled: enabled && !!organizationId && !!canvasId,
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
  limit: number = 50,
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
        (acc, page) => acc + (page?.versions?.filter((version) => isPublishedVersion(version)).length || 0),
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
    queryFn: async () => fetchCanvasVersionWithSpec(canvasId, versionId),
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
          query: { limit: 25, statusFilter: "all" },
        }),
      );
      return response.data?.changeRequests || [];
    },
    enabled: !!organizationId && !!canvasId,
  });
};

type CanvasChangeRequestFilter = "open" | "rejected" | "merged" | "all";

type CanvasGraphData = {
  nodes?: unknown[];
  edges?: unknown[];
};

type PositionedNode = SuperplaneComponentsNode & {
  id: string;
  position: ComponentsPosition;
};

const versionSortTimestamp = (version: CanvasesCanvasVersion): number => {
  const raw = version?.metadata?.updatedAt || version?.metadata?.createdAt;
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
    mutationFn: async (
      data: {
        name: string;
        description?: string;
        method?: "ui" | "cli" | "yaml_import" | "template";
        templateId?: string;
      } & CanvasGraphData,
    ) => {
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
    onSuccess: (response, variables) => {
      // Invalidate the list to refresh the canvas list
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });

      // Set the workflow detail in cache immediately so it's available when navigating
      if (response?.data?.canvas?.metadata?.id) {
        queryClient.setQueryData(
          canvasKeys.detail(organizationId, response.data.canvas.metadata.id),
          response.data.canvas,
        );
        analytics.canvasCreate(
          response.data.canvas.metadata.id,
          organizationId,
          variables.method ?? "ui",
          variables.templateId,
          !!variables.description,
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
      changeManagement?: {
        enabled?: boolean;
        approvals?: Array<{ type?: string; userId?: string; roleName?: string }>;
      };
    }) => {
      return await canvasesUpdateCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
          body: {
            name: data.name,
            description: data.description,
            changeManagement: data.changeManagement,
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
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), (current: CanvasesCanvas | undefined) => {
          if (!current) {
            return current;
          }

          const updatedMetadata = updatedCanvas.metadata;
          const updatedSpec = updatedCanvas.spec;

          return {
            ...current,
            metadata: {
              ...current.metadata,
              name: updatedMetadata?.name ?? variables.name ?? current.metadata?.name,
              description: updatedMetadata?.description ?? variables.description ?? current.metadata?.description,
            },
            spec: {
              ...current.spec,
              changeManagement:
                updatedSpec?.changeManagement ?? variables.changeManagement ?? current.spec?.changeManagement,
            },
          };
        });
      }
    },
  });
};

export const useCreateCanvasFolder = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { title: string; backgroundColor?: CanvasFolderColor }) => {
      return await canvasFoldersCreateCanvasFolder(
        withOrganizationHeader({
          organizationId,
          body: {
            folder: {
              spec: {
                title: data.title,
                backgroundColor: data.backgroundColor || DEFAULT_CANVAS_FOLDER_COLOR,
              },
            },
          },
        }),
      );
    },
    onSuccess: (response) => {
      const createdFolder = response?.data?.folder;
      queryClient.setQueryData(
        canvasKeys.folderList(organizationId),
        (current: CanvasFoldersCanvasFolder[] | undefined) => {
          if (!createdFolder?.metadata?.id) {
            return current;
          }

          const nextFolders = current ? [...current] : [];
          const existingFolderIndex = nextFolders.findIndex(
            (folder) => folder.metadata?.id === createdFolder.metadata?.id,
          );
          if (existingFolderIndex >= 0) {
            nextFolders[existingFolderIndex] = createdFolder;
          } else {
            nextFolders.unshift(createdFolder);
          }

          return nextFolders;
        },
      );
      queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
    },
  });
};

export const useUpdateCanvasFolder = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { folderId: string; title: string; backgroundColor: CanvasFolderColor }) => {
      return await canvasFoldersUpdateCanvasFolder(
        withOrganizationHeader({
          organizationId,
          path: { id: data.folderId },
          body: {
            folder: {
              spec: {
                title: data.title,
                backgroundColor: data.backgroundColor,
              },
            },
          },
        }),
      );
    },
    onSuccess: (response) => {
      const updatedFolder = response?.data?.folder;
      queryClient.setQueryData(
        canvasKeys.folderList(organizationId),
        (current: CanvasFoldersCanvasFolder[] | undefined) => {
          if (!current || !updatedFolder?.metadata?.id) {
            return current;
          }

          const nextFolders = current.map((folder) =>
            folder.metadata?.id === updatedFolder.metadata?.id ? updatedFolder : folder,
          );
          return nextFolders;
        },
      );
      queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
    },
  });
};

export const useMoveCanvasFolder = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { folderId: string; direction: "DIRECTION_UP" | "DIRECTION_DOWN" }) => {
      return await canvasFoldersUpdateCanvasFolderPosition(
        withOrganizationHeader({
          organizationId,
          path: { id: data.folderId },
          body: {
            direction: data.direction,
          },
        }),
      );
    },
    onSuccess: (response) => {
      const folders = response?.data?.folders;
      if (folders) {
        queryClient.setQueryData(canvasKeys.folderList(organizationId), folders);
      }

      queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
    },
  });
};

export const useDeleteCanvasFolder = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (folderId: string) => {
      return await canvasFoldersDeleteCanvasFolder(
        withOrganizationHeader({
          organizationId,
          path: { id: folderId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    },
  });
};

type UpdateCanvasFolderMembershipInput = {
  folderId: string;
  title: string;
  backgroundColor: CanvasFolderColor;
  canvasIds: string[];
};

type CanvasMetadataWithFolder = CanvasesCanvas["metadata"] & {
  folderId?: string;
};

function updateCanvasListFolderMembership(
  canvases: CanvasesCanvas[] | undefined,
  data: UpdateCanvasFolderMembershipInput,
) {
  if (!canvases) {
    return canvases;
  }

  const targetCanvasIds = new Set(data.canvasIds);

  return canvases.map((canvas) => {
    const metadata = canvas.metadata as CanvasMetadataWithFolder | undefined;
    const canvasId = metadata?.id;
    if (!canvasId) {
      return canvas;
    }

    if (targetCanvasIds.has(canvasId)) {
      return {
        ...canvas,
        metadata: {
          ...metadata,
          folderId: data.folderId,
        },
      };
    }

    if (metadata.folderId !== data.folderId) {
      return canvas;
    }

    return {
      ...canvas,
      metadata: {
        ...metadata,
        folderId: undefined,
      },
    };
  });
}

function updateCanvasFolderListMembership(
  folders: CanvasFoldersCanvasFolder[] | undefined,
  data: UpdateCanvasFolderMembershipInput,
) {
  if (!folders) {
    return folders;
  }

  const targetCanvasIds = new Set(data.canvasIds);

  return folders.map((folder) => {
    const folderId = folder.metadata?.id;
    if (!folderId) {
      return folder;
    }

    if (folderId === data.folderId) {
      return {
        ...folder,
        spec: {
          ...folder.spec,
          title: data.title,
          backgroundColor: data.backgroundColor,
          canvases: data.canvasIds.map((id) => ({ id })),
        },
      };
    }

    const canvases = folder.spec?.canvases || [];
    const nextCanvases = canvases.filter((canvas) => !canvas.id || !targetCanvasIds.has(canvas.id));
    if (nextCanvases.length === canvases.length) {
      return folder;
    }

    return {
      ...folder,
      spec: {
        ...folder.spec,
        canvases: nextCanvases,
      },
    };
  });
}

export const useUpdateCanvasFolderMembership = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: UpdateCanvasFolderMembershipInput) => {
      return await canvasFoldersUpdateCanvasFolder(
        withOrganizationHeader({
          organizationId,
          path: { id: data.folderId },
          body: {
            folder: {
              spec: {
                title: data.title,
                backgroundColor: data.backgroundColor,
                canvases: data.canvasIds.map((id) => ({ id })),
              },
            },
            replaceMembership: true,
          },
        }),
      );
    },
    onMutate: async (data) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: canvasKeys.list(organizationId) }),
        queryClient.cancelQueries({ queryKey: canvasKeys.folderList(organizationId) }),
      ]);

      const previousCanvases = queryClient.getQueryData<CanvasesCanvas[]>(canvasKeys.list(organizationId));
      const previousFolders = queryClient.getQueryData<CanvasFoldersCanvasFolder[]>(
        canvasKeys.folderList(organizationId),
      );

      queryClient.setQueryData(canvasKeys.list(organizationId), (current: CanvasesCanvas[] | undefined) =>
        updateCanvasListFolderMembership(current, data),
      );
      queryClient.setQueryData(
        canvasKeys.folderList(organizationId),
        (current: CanvasFoldersCanvasFolder[] | undefined) => updateCanvasFolderListMembership(current, data),
      );

      return { previousCanvases, previousFolders };
    },
    onError: (_error, _data, context) => {
      if (context?.previousCanvases) {
        queryClient.setQueryData(canvasKeys.list(organizationId), context.previousCanvases);
      }

      if (context?.previousFolders) {
        queryClient.setQueryData(canvasKeys.folderList(organizationId), context.previousFolders);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
    },
  });
};

export const useListDraftBranches = (organizationId: string, canvasId: string, enabled = true) => {
  return useQuery({
    queryKey: canvasKeys.draftBranches(canvasId),
    queryFn: async () => {
      const response = await canvasesListCanvasVersions(
        withOrganizationHeader({
          path: { canvasId },
          query: { state: "STATE_DRAFT" },
        }),
      );
      return response.data?.versions ?? [];
    },
    enabled: enabled && !!organizationId && !!canvasId,
  });
};

export const useCreateDraftBranch = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (displayName?: string) => {
      return await canvasesCreateCanvasVersion(
        withOrganizationHeader({
          path: { canvasId },
          body: displayName ? { displayName } : {},
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
    },
  });
};

export const useDeleteDraftBranch = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (versionId: string) => {
      return await canvasesDeleteCanvasVersion(
        withOrganizationHeader({
          path: { canvasId, versionId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
    },
  });
};

export const usePublishCanvasVersion = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (versionId: string) => {
      return await canvasesPublishCanvasVersion(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: {},
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.consoleAll(canvasId) });
    },
  });
};

export const useUpdateCanvasVersion = (organizationId: string, canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      versionId?: string;
      canvasYaml: string;
      autoLayout?: { algorithm?: string; scope?: string; nodeIds?: string[] };
      preserveLocalCanvasState?: boolean;
      invalidateRelatedQueries?: boolean;
    }) => {
      if (!data.versionId) {
        throw new Error("version id is required");
      }

      await canvasesCommitCanvasRepositoryFiles(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            versionId: data.versionId,
            message: "Update canvas.yaml",
            operations: [
              {
                path: CANVAS_YAML_PATH,
                content: encodeRepositoryFileContent(data.canvasYaml),
              },
            ],
            ...(data.autoLayout ? { autoLayout: data.autoLayout } : {}),
          },
        }),
      );

      const [describeResponse, canvasYaml] = await Promise.all([
        canvasesDescribeCanvasVersion(
          withOrganizationHeader({
            path: { canvasId, versionId: data.versionId },
          }),
        ),
        fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, data.versionId),
      ]);

      const version = versionWithSpecFromYaml(describeResponse.data?.version, canvasYaml);
      return { data: { canvasYaml, version } };
    },
    onSuccess: (response, variables) => {
      const version = versionWithSpecFromYaml(response?.data?.version, response?.data?.canvasYaml);
      if (!version) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        return;
      }

      if (variables.versionId) {
        queryClient.setQueryData(canvasKeys.versionDetail(canvasId, variables.versionId), version);
      }

      queryClient.setQueryData(canvasKeys.versionList(canvasId), (current: CanvasesCanvasVersion[] | undefined) => {
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

      if (!variables.preserveLocalCanvasState) {
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), (current: CanvasesCanvas | undefined) => {
          if (!current) {
            return current;
          }

          const currentNodeMetadataById = new Map(
            (current.spec?.nodes ?? [])
              .filter((node) => Boolean(node.id) && node.metadata !== undefined && node.metadata !== null)
              .map((node) => [node.id as string, node.metadata] as const),
          );

          const mergeServerNodeWithLocalMetadata = (serverNode: SuperplaneComponentsNode): SuperplaneComponentsNode => {
            if (!serverNode.id) {
              return serverNode;
            }

            const localMetadata = currentNodeMetadataById.get(serverNode.id);
            if (localMetadata === undefined || localMetadata === null || serverNode.metadata !== undefined) {
              return serverNode;
            }

            return { ...serverNode, metadata: localMetadata };
          };

          // When the server computed a new layout (autoLayout), accept the
          // server positions as authoritative. Otherwise preserve current
          // local node positions to avoid overwriting positions that changed
          // while the save was in flight.
          if (variables.autoLayout) {
            const mergedNodes = (version.spec?.nodes ?? []).map(mergeServerNodeWithLocalMetadata);
            return { ...current, spec: { ...current.spec, ...version.spec, nodes: mergedNodes } };
          }

          const currentPositionsByNodeId = new Map(
            (current.spec?.nodes ?? [])
              .filter((node): node is PositionedNode => Boolean(node.id && node.position))
              .map((node) => [node.id, node.position] as const),
          );

          const mergedNodes = (version.spec?.nodes ?? []).map((rawServerNode) => {
            const serverNode = mergeServerNodeWithLocalMetadata(rawServerNode);
            if (!serverNode.id) {
              return serverNode;
            }

            const localPosition = currentPositionsByNodeId.get(serverNode.id);
            if (localPosition) {
              return { ...serverNode, position: localPosition };
            }
            return serverNode;
          });

          return {
            ...current,
            spec: { ...current.spec, ...version.spec, nodes: mergedNodes },
          };
        });
      }

      if (variables.invalidateRelatedQueries !== false) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequests() });
        queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        queryClient.invalidateQueries({
          queryKey: canvasKeys.repositoryFile(canvasId, CANVAS_YAML_PATH, variables.versionId),
        });
      }
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
      nodes?: unknown[];
      edges?: unknown[];
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
      // Capture node count before removing from cache.
      // Fall back to the list cache if the detail page was never opened.
      const cachedDetail = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
      const cachedList = queryClient.getQueryData<CanvasesCanvas[]>(canvasKeys.list(organizationId));
      const cachedCanvas = cachedDetail ?? cachedList?.find((c) => c.metadata?.id === canvasId);
      const nodeCount = cachedCanvas?.spec?.nodes?.length ?? 0;

      // Remove from cache immediately before deletion to prevent 404 flash
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });

      const result = await canvasesDeleteCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
        }),
      );
      return { result, nodeCount };
    },
    onSuccess: ({ nodeCount }, canvasId) => {
      // Ensure it's removed (in case it wasn't already)
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      // Invalidate the list to refresh the canvas list
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      analytics.canvasDelete(canvasId, organizationId, nodeCount);
    },
  });
};

export const useInfiniteCanvasEvents = (canvasId: string, enabled = true) => {
  const limit = 25;

  return useInfiniteQuery({
    queryKey: canvasKeys.infiniteEvents(canvasId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListCanvasEvents(
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
    staleTime: 0,
    refetchOnWindowFocus: false,
    enabled: !!canvasId && enabled,
  });
};

export type CanvasRunsFilters = {
  states?: CanvasesCanvasRunState[];
  results?: CanvasesCanvasRunResult[];
};

export const useInfiniteCanvasRuns = (canvasId: string, filters: CanvasRunsFilters = {}, enabled = true) => {
  const limit = 25;

  return useInfiniteQuery({
    queryKey: canvasKeys.infiniteRuns(canvasId, filters),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await canvasesListRuns(
        withOrganizationHeader({
          path: { canvasId },
          query: {
            limit,
            ...(filters.states?.length ? { states: filters.states } : {}),
            ...(filters.results?.length ? { results: filters.results } : {}),
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: (lastPage, allPages) => {
      const currentLoadedCount = allPages.reduce((acc, page) => acc + (page?.runs?.length || 0), 0);
      const totalCount = lastPage?.totalCount || 0;

      if (currentLoadedCount >= totalCount) return undefined;
      return lastPage?.lastTimestamp;
    },
    initialPageParam: undefined as string | undefined,
    staleTime: 0,
    refetchOnWindowFocus: false,
    enabled: !!canvasId && enabled,
  });
};

export type CanvasMemoryEntrySource = "node" | "manual" | "unknown";

export interface CanvasMemoryEntry {
  id: string;
  namespace: string;
  values: unknown;
  source: CanvasMemoryEntrySource;
  /** Server timestamp the entry was first persisted. ISO-8601 string. */
  createdAt?: string;
  /** Server timestamp the entry was last updated. ISO-8601 string. */
  updatedAt?: string;
}

function normalizeCanvasMemorySource(source: string | undefined): CanvasMemoryEntrySource {
  if (source === "SOURCE_MANUAL") return "manual";
  if (source === "SOURCE_NODE") return "node";
  return "unknown";
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
        source: normalizeCanvasMemorySource(item.source),
        createdAt: item.createdAt,
        updatedAt: item.updatedAt,
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

export interface CreateCanvasMemoryNamespaceInput {
  namespace: string;
  entries: unknown[];
}

export const useCreateCanvasMemoryNamespace = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ namespace, entries }: CreateCanvasMemoryNamespaceInput) => {
      await canvasesCreateCanvasMemoryNamespace(
        withOrganizationHeader({
          path: { canvasId },
          body: { namespace, entries },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.canvasMemoryEntries(canvasId) });
    },
  });
};

export interface UpdateCanvasMemoryNamespaceInput {
  namespace: string;
  newNamespace?: string;
  entries: unknown[];
}

export const useUpdateCanvasMemoryNamespace = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ namespace, newNamespace, entries }: UpdateCanvasMemoryNamespaceInput) => {
      await canvasesUpdateCanvasMemoryNamespace(
        withOrganizationHeader({
          path: { canvasId, namespace },
          body: {
            newNamespace: newNamespace && newNamespace !== namespace ? newNamespace : undefined,
            entries,
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

/**
 * Fetch executions for multiple root event ids in parallel. Each query is
 * keyed by `(canvasId, eventId)`, identical to `useEventExecutions`, so the
 * results dedupe with any single-event consumer (e.g. `RunNodeDetailModal`)
 * already in the React Query cache. Returns the per-event results array
 * along with an aggregate `isLoading` flag that's `true` while any of the
 * underlying queries hasn't resolved yet.
 */
export const useEventExecutionsBatch = (canvasId: string, eventIds: string[]) => {
  const queries = useQueries({
    queries: eventIds.map((eventId) => ({
      queryKey: canvasKeys.eventExecution(canvasId, eventId),
      queryFn: async () => {
        const response = await canvasesListEventExecutions(
          withOrganizationHeader({
            path: { canvasId, eventId },
          }),
        );
        return response.data;
      },
      refetchOnWindowFocus: false,
      enabled: !!canvasId && !!eventId,
      staleTime: 30 * 1000,
      gcTime: 5 * 60 * 1000,
    })),
  });
  const isLoading = queries.some((q) => q.isLoading);
  return { queries, isLoading };
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
  queryKey: canvasKeys.nodeExecution(canvasId, nodeId, options?.states, options?.limit),
  queryFn: async () => {
    const response = await canvasesListNodeExecutions(
      withOrganizationHeader({
        path: {
          canvasId,
          nodeId,
        },
        query: options,
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
            limit: NODE_EXECUTION_HISTORY_PAGE_SIZE,
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

export const useCanvasConsole = (canvasId: string, versionId: string | undefined, enabled: boolean = true) => {
  return useQuery({
    queryKey: canvasKeys.console(canvasId, versionId),
    queryFn: async () => {
      const spec = await fetchConsoleSpecFromRepository(canvasId, versionId);
      if (!spec) {
        return undefined;
      }
      return consoleDataFromYaml(canvasId, versionId, spec.consoleYaml);
    },
    enabled: enabled && !!canvasId,
    staleTime: 30_000,
  });
};

type UseUpdateCanvasConsoleOptions = {
  registerIgnoredCanvasVersionUpdatedEcho?: (savingVersionId?: string) => () => void;
};

function toCanvasConsole(
  canvasId: string,
  versionId: string | undefined,
  input: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] },
  previous?: CanvasConsoleData,
): CanvasConsoleData {
  const consoleYaml =
    previous?.consoleYaml ??
    materializeConsoleSpec({
      panels: input.panels,
      layout: input.layout,
      canvasId,
    });

  return {
    ...previous,
    canvasId: previous?.canvasId ?? canvasId,
    ...(versionId ? { versionId: previous?.versionId ?? versionId } : {}),
    panels: input.panels,
    layout: input.layout,
    consoleYaml,
  };
}

export const useUpdateCanvasConsole = (
  canvasId: string,
  versionId: string | undefined,
  options?: UseUpdateCanvasConsoleOptions,
) => {
  const queryClient = useQueryClient();
  return useMutation({
    onMutate: async (input) => {
      const queryKey = canvasKeys.console(canvasId, versionId);
      if (input.panels === undefined || input.layout === undefined) {
        return { previous: queryClient.getQueryData<CanvasConsoleData>(queryKey), queryKey };
      }

      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<CanvasConsoleData>(queryKey);
      queryClient.setQueryData(
        queryKey,
        toCanvasConsole(canvasId, versionId, { panels: input.panels, layout: input.layout }, previous),
      );
      return { previous, queryKey };
    },
    mutationFn: async (input: { panels?: ConsolePanel[]; layout?: ConsoleLayoutItem[]; consoleYaml?: string }) => {
      if (!versionId) {
        throw new Error("version id is required");
      }

      const releaseCanvasVersionUpdatedEcho = options?.registerIgnoredCanvasVersionUpdatedEcho?.(versionId);
      try {
        const consoleYaml =
          input.consoleYaml ??
          materializeConsoleSpec({
            panels: input.panels ?? [],
            layout: input.layout ?? [],
            canvasId,
          });

        await canvasesCommitCanvasRepositoryFiles(
          withOrganizationHeader({
            path: { canvasId },
            body: {
              versionId,
              message: "Update console.yaml",
              operations: [
                {
                  path: CONSOLE_YAML_PATH,
                  content: encodeRepositoryFileContent(consoleYaml),
                },
              ],
            },
          }),
        );

        const spec = await fetchConsoleSpecFromRepository(canvasId, versionId);
        if (!spec) {
          return consoleDataFromYaml(canvasId, versionId, consoleYaml);
        }
        return consoleDataFromYaml(canvasId, versionId, spec.consoleYaml);
      } catch (error) {
        releaseCanvasVersionUpdatedEcho?.();
        throw error;
      }
    },
    onError: (_error, _input, context) => {
      if (!context) return;
      queryClient.setQueryData(context.queryKey, context.previous);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(canvasKeys.console(canvasId, versionId), data);
    },
  });
};

export type CanvasConsoleQueryResult = ReturnType<typeof useCanvasConsole>;
export type UpdateCanvasConsoleMutationResult = ReturnType<typeof useUpdateCanvasConsole>;

async function fetchRepositoryFileContent(canvasId: string, path: string, versionId?: string): Promise<string> {
  return fetchRepositorySpecFileContent(canvasId, path, versionId);
}

export const useCanvasRepository = (canvasId: string, enabled: boolean = true) => {
  return useQuery({
    queryKey: canvasKeys.repository(canvasId),
    queryFn: async () => {
      const response = await canvasesGetCanvasRepository(
        withOrganizationHeader({
          path: { canvasId },
        }),
      );
      return response.data?.repository;
    },
    enabled: enabled && !!canvasId,
    staleTime: 30_000,
    refetchInterval: (query) => {
      const state = query.state.data?.status?.state;
      return state === "STATE_PENDING" ? 3000 : false;
    },
  });
};

export const useCanvasRepositoryFiles = (canvasId: string, enabled: boolean = true) => {
  return useQuery({
    queryKey: canvasKeys.repositoryFiles(canvasId),
    queryFn: async () => {
      const response = await canvasesListCanvasRepositoryFiles(
        withOrganizationHeader({
          path: { canvasId },
        }),
      );
      return response.data;
    },
    enabled: enabled && !!canvasId,
    staleTime: 15_000,
  });
};

export const useCanvasRepositoryFile = (
  canvasId: string,
  path: string | null,
  enabled: boolean = true,
  versionId?: string,
) => {
  const normalizedPath = path ?? "";
  return useQuery({
    queryKey: canvasKeys.repositoryFile(canvasId, normalizedPath, versionId),
    queryFn: async () => {
      const content = await fetchRepositoryFileContent(canvasId, normalizedPath, versionId);
      return {
        path: normalizedPath,
        content,
      };
    },
    enabled: enabled && !!canvasId && !!normalizedPath,
    staleTime: 15_000,
  });
};

export const useCommitCanvasRepositoryFiles = (canvasId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      message: string;
      operations: CanvasesCanvasRepositoryFileOperation[];
      expectedHeadSha?: string;
      versionId?: string;
      autoLayout?: { algorithm?: string; scope?: string; nodeIds?: string[] };
    }) => {
      const response = await canvasesCommitCanvasRepositoryFiles(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            message: input.message,
            operations: input.operations,
            expectedHeadSha: input.expectedHeadSha,
            versionId: input.versionId,
            ...(input.autoLayout ? { autoLayout: input.autoLayout } : {}),
          },
        }),
      );
      return response.data;
    },
    onSuccess: (_data, input) => {
      queryClient.setQueryData<CanvasesListCanvasRepositoryFilesResponse | undefined>(
        canvasKeys.repositoryFiles(canvasId),
        (current) => {
          const paths = new Set(
            (current?.files || []).map((file) => file.path).filter((path): path is string => !!path),
          );

          for (const operation of input.operations) {
            const path = operation.path;
            if (!path) continue;

            if (operation.delete) {
              paths.delete(path);
              continue;
            }

            paths.add(path);
          }

          return {
            ...current,
            files: Array.from(paths)
              .sort((left, right) => left.localeCompare(right))
              .map((path) => ({ path })),
          };
        },
      );
      queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId) });
      queryClient.invalidateQueries({ queryKey: [...canvasKeys.repository(canvasId), "file"] });
      if (input.versionId) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(canvasId, input.versionId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.console(canvasId, input.versionId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.consoleAll(canvasId) });
      }
    },
  });
};

export type CanvasRepositoryFilesQueryResult = ReturnType<typeof useCanvasRepositoryFiles>;
export type CanvasRepositoryFileQueryResult = ReturnType<typeof useCanvasRepositoryFile>;
export type CommitCanvasRepositoryFilesMutationResult = ReturnType<typeof useCommitCanvasRepositoryFiles>;
