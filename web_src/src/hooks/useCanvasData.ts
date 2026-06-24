import { useQuery, useMutation, useQueryClient, useInfiniteQuery, useQueries } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { upsertRunIntoDescribeRunData } from "./canvasInfiniteCache";
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
  canvasesDeleteCanvas,
  canvasesPublishCanvasVersion,
  canvasesListNodeExecutions,
  canvasesListRuns,
  canvasesDescribeRun,
  canvasesListCanvasMemories,
  canvasesDeleteCanvasMemory,
  canvasesCreateCanvasMemoryNamespace,
  canvasesUpdateCanvasMemoryNamespace,
  canvasesListEventExecutions,
  canvasesListNodeQueueItems,
  canvasesListNodeEvents,
  canvasesGetCanvasRepository,
  canvasesListCanvasRepositoryFiles,
  canvasesStageCanvasRepositoryFile,
  canvasesCommitCanvasStaging,
  canvasesDiscardCanvasStaging,
  canvasesApplyCanvasAutoLayout,
  triggersListTriggers,
  triggersDescribeTrigger,
  widgetsListWidgets,
  widgetsDescribeWidget,
} from "../api-client/sdk.gen";
import type {
  CanvasFoldersCanvasFolder,
  CanvasesCanvas,
  CanvasesCanvasSummary,
  CanvasesCanvasRun,
  CanvasesCanvasRunResult,
  CanvasesCanvasRunState,
  CanvasesCanvasVersion,
  CanvasesCanvasRepositoryFileOperation,
  CanvasesListCanvasRepositoryFilesResponse,
  SuperplaneComponentsNode,
  ComponentsPosition,
} from "../api-client/types.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { registerLocalStagingWrite } from "../lib/canvasStagingEcho";
import { draftVersionId } from "../lib/draftVersion";
import { analytics } from "../lib/analytics";
import { isDraftVersion, isPublishedVersion } from "../pages/app/lib/canvas-versions";
import {
  canvasVersionWithSpecFromYaml,
  fetchCanvasVersionWithSpec,
  fetchCanvasVersionStagingSummary,
  fetchConsoleSpecFromRepository,
  fetchRepositorySpecFileContent,
} from "../pages/app/lib/repository-spec-files";
import { encodeRepositoryFileContent } from "../pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "../pages/app/lib/workflow-spec-paths";
import { matchesCommittedCanvasYaml, matchesCommittedConsoleYaml } from "../pages/app/lib/staging-content-match";
import { dematerializeConsoleSpec, materializeConsoleSpec } from "../pages/app/lib/workflow-spec-files";

function versionWithSpecFromYaml(
  version: CanvasesCanvasVersion | undefined,
  canvasYaml: string | undefined,
): CanvasesCanvasVersion | undefined {
  return canvasVersionWithSpecFromYaml(version, canvasYaml);
}

type SpecAutoLayout = { algorithm?: string; scope?: string; nodeIds?: string[] };

// stageSpecOperations writes canvas.yaml/console.yaml edits to the draft
// version's staging layer without mutating the committed version row.
async function stageSpecOperations(
  canvasId: string,
  versionId: string,
  operations: CanvasesCanvasRepositoryFileOperation[],
) {
  registerLocalStagingWrite(canvasId, versionId);
  await canvasesStageCanvasRepositoryFile(
    withOrganizationHeader({
      path: { canvasId, versionId },
      body: { operations },
    }),
  );
}

async function discardStagedPaths(canvasId: string, versionId: string, paths: string[]) {
  registerLocalStagingWrite(canvasId, versionId);
  await canvasesDiscardCanvasStaging(
    withOrganizationHeader({
      path: { canvasId, versionId },
      body: paths.length > 0 ? { paths } : {},
    }),
  );
}

// applyCanvasStagingAutoLayout lays out the staged canvas.yaml and re-stages it.
async function applyCanvasStagingAutoLayout(canvasId: string, versionId: string, autoLayout: SpecAutoLayout) {
  registerLocalStagingWrite(canvasId, versionId);
  await canvasesApplyCanvasAutoLayout(
    withOrganizationHeader({
      path: { canvasId, versionId },
      body: { autoLayout },
    }),
  );
}

// commitCanvasStaging parses the staged spec files into the draft version row
// and clears staging.
async function commitCanvasStaging(canvasId: string, versionId: string) {
  await canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId, versionId },
      body: {},
    }),
  );
}

// stageCommitSpecOperations stages spec edits, optionally lays out the staged
// canvas, then commits staging into the draft version row. This preserves the
// "edit spec -> draft version updated" behavior on top of the staging layer.
async function stageCommitSpecOperations(
  canvasId: string,
  versionId: string,
  operations: CanvasesCanvasRepositoryFileOperation[],
  autoLayout?: SpecAutoLayout,
) {
  await stageSpecOperations(canvasId, versionId, operations);
  const touchesCanvasYaml = operations.some((operation) => operation.path === CANVAS_YAML_PATH && !operation.delete);
  if (autoLayout && touchesCanvasYaml) {
    await applyCanvasStagingAutoLayout(canvasId, versionId, autoLayout);
  }
  await commitCanvasStaging(canvasId, versionId);
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
  details: () => [...canvasKeys.all, "detail"] as const,
  detail: (orgId: string, id: string) => [...canvasKeys.details(), orgId, id] as const,
  versions: () => [...canvasKeys.all, "versions"] as const,
  versionList: (canvasId: string) => [...canvasKeys.versions(), canvasId] as const,
  versionHistory: (canvasId: string) => [...canvasKeys.versions(), canvasId, "history"] as const,
  versionDetails: () => [...canvasKeys.versions(), "detail"] as const,
  versionDetail: (canvasId: string, versionId: string) =>
    [...canvasKeys.versionDetails(), canvasId, versionId] as const,
  // Staged reads overlay uncommitted edits on the committed version. They must
  // not share the committed `versionDetail` cache entry, otherwise a committed
  // (stage=false) fetch for the same version overwrites the staged content and
  // the editor loses pending edits. Kept as a prefix-extension of versionDetail
  // so prefix invalidations of versionDetail also refresh the staged read.
  versionStagedDetail: (canvasId: string, versionId: string) =>
    [...canvasKeys.versionDetails(), canvasId, versionId, "staged"] as const,
  versionStaging: (canvasId: string, versionId: string) =>
    [...canvasKeys.versions(), "staging", canvasId, versionId] as const,
  draftBranches: (canvasId: string) => [...canvasKeys.all, "draftBranches", canvasId] as const,
  nodeExecutions: () => [...canvasKeys.all, "nodeExecutions"] as const,
  nodeExecution: (canvasId: string, nodeId: string, states?: string[], limit?: number) =>
    [
      ...canvasKeys.nodeExecutions(),
      canvasId,
      nodeId,
      ...(states || []),
      ...(limit === undefined ? [] : [limit]),
    ] as const,
  runs: () => [...canvasKeys.all, "runs"] as const,
  infiniteRuns: (canvasId: string, filters?: CanvasRunsFilters) =>
    [
      ...canvasKeys.runs(),
      canvasId,
      "infinite",
      ...(filters?.states?.length ? ["states", ...filters.states] : []),
      ...(filters?.results?.length ? ["results", ...filters.results] : []),
    ] as const,
  run: (canvasId: string, runId: string) => [...canvasKeys.runs(), canvasId, runId] as const,
  eventExecutions: () => [...canvasKeys.all, "eventExecutions"] as const,
  eventExecution: (canvasId: string, eventId: string) => [...canvasKeys.eventExecutions(), canvasId, eventId] as const,
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
  // Staged console overlays uncommitted edits; kept separate from the committed
  // `console` entry so a committed (stage=false) refetch cannot overwrite the
  // editor's pending edits. Prefix-extends `console`, so invalidating `console`
  // (or `consoleAll`) also refreshes the staged read.
  consoleStaged: (canvasId: string, versionId?: string) =>
    [...canvasKeys.console(canvasId, versionId), "staged"] as const,
  consoleAll: (canvasId: string) => [...canvasKeys.all, "console", canvasId] as const,
  repository: (canvasId: string) => [...canvasKeys.all, "repository", canvasId] as const,
  repositoryFiles: (canvasId: string) => [...canvasKeys.repository(canvasId), "files"] as const,
  repositoryFile: (canvasId: string, path: string, versionId?: string, stage = false) =>
    [...canvasKeys.repository(canvasId), "file", path, versionId ?? "live", stage ? "staged" : "committed"] as const,
  // Raw repository-file content keyed per stage so cached reads can be reused
  // and deduped (e.g. the Files diff and committed-baseline lookups). It
  // prefix-extends `repositoryFile`, so any invalidation of a file (or the
  // whole repository) also clears its cached content.
  repositoryFileContent: (canvasId: string, path: string, versionId: string | undefined, stage: boolean) =>
    [...canvasKeys.repositoryFile(canvasId, path, versionId), "content", stage ? "staged" : "committed"] as const,
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
        }),
      );
      return response.data?.canvases || [];
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

function resolveCanvasVersionStage(stage: boolean | CanvasesCanvasVersion | null | undefined): boolean {
  if (typeof stage === "boolean") {
    return stage;
  }

  return !!stage && isDraftVersion(stage);
}

export const useCanvasVersion = (
  organizationId: string,
  canvasId: string,
  versionId: string,
  enabled = true,
  stage: boolean | CanvasesCanvasVersion | null = false,
) => {
  const readStaged = resolveCanvasVersionStage(stage);
  return useQuery({
    queryKey: readStaged
      ? canvasKeys.versionStagedDetail(canvasId, versionId)
      : canvasKeys.versionDetail(canvasId, versionId),
    queryFn: async () => fetchCanvasVersionWithSpec(canvasId, versionId, readStaged),
    enabled: !!organizationId && !!canvasId && !!versionId && enabled,
  });
};

// useCanvasVersionStaging exposes the uncommitted StagingSummary for a draft
// version so the UI can show orange "uncommitted changes" indicators and gate
// the Commit / Reset actions.
export const useCanvasVersionStaging = (canvasId: string, versionId: string | undefined, enabled = true) => {
  return useQuery({
    queryKey: canvasKeys.versionStaging(canvasId, versionId ?? ""),
    queryFn: async () => {
      const state = await fetchCanvasVersionStagingSummary(canvasId, versionId!);
      return state ?? { hasStaging: false, stagedPaths: [] };
    },
    enabled: enabled && !!canvasId && !!versionId,
    staleTime: 0,
  });
};

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
    mutationFn: async (data: { name?: string; description?: string }) => {
      return await canvasesUpdateCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
          body: {
            name: data.name,
            description: data.description,
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
            spec: updatedSpec ?? current.spec,
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

function updateCanvasListFolderMembership(
  canvases: CanvasesCanvasSummary[] | undefined,
  data: UpdateCanvasFolderMembershipInput,
) {
  if (!canvases) {
    return canvases;
  }

  const targetCanvasIds = new Set(data.canvasIds);

  return canvases.map((canvas) => {
    const canvasId = canvas.id;
    if (!canvasId) {
      return canvas;
    }

    if (targetCanvasIds.has(canvasId)) {
      return {
        ...canvas,
        folderId: data.folderId,
      };
    }

    if (canvas.folderId !== data.folderId) {
      return canvas;
    }

    return {
      ...canvas,
      folderId: undefined,
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

      const previousCanvases = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
      const previousFolders = queryClient.getQueryData<CanvasFoldersCanvasFolder[]>(
        canvasKeys.folderList(organizationId),
      );

      queryClient.setQueryData(canvasKeys.list(organizationId), (current: CanvasesCanvasSummary[] | undefined) =>
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

// Refetches the draft branches and reports whether the given version still
// exists, so commit/publish can pre-check a possibly-stale draft id.
export const ensureDraftVersionExists = async (
  queryClient: QueryClient,
  organizationId: string,
  canvasId: string,
  versionId: string,
): Promise<boolean> => {
  if (!organizationId || !canvasId || !versionId) {
    return false;
  }

  const branches = await queryClient.fetchQuery({
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
    // Always hit the network — a cached list (app-wide 5-min staleTime) could
    // still contain a draft that was just deleted, defeating the guard.
    staleTime: 0,
  });

  return branches.some((branch) => draftVersionId(branch) === versionId);
};

type UseCreateDraftBranchOptions = {
  registerIgnoredCreateDraftEcho?: (targetCanvasId: string) => () => void;
};

export const useCreateDraftBranch = (canvasId: string, options?: UseCreateDraftBranchOptions) => {
  const queryClient = useQueryClient();

  return useMutation({
    onMutate: () => {
      const releaseCreateDraftEcho = options?.registerIgnoredCreateDraftEcho?.(canvasId);
      return { releaseCreateDraftEcho };
    },
    mutationFn: async (displayName?: string) => {
      return await canvasesCreateCanvasVersion(
        withOrganizationHeader({
          path: { canvasId },
          body: displayName ? { displayName } : {},
        }),
      );
    },
    onSuccess: () => {
      // Creating a draft does not change the live canvas, so we intentionally do
      // not invalidate canvasKeys.detail here — doing so would trigger a
      // DescribeCanvas refetch while entering/staying in edit mode.
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
    },
    onError: (_error, _variables, context) => {
      context?.releaseCreateDraftEcho?.();
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

      // Stage-only: write canvas.yaml to the draft's staging layer and (when
      // requested) lay it out. The committed version row is only updated by an
      // explicit Commit (useCommitCanvasStaging).
      const canvasMatchesCommitted = await matchesCommittedCanvasYaml(canvasId, data.versionId, data.canvasYaml);
      if (canvasMatchesCommitted) {
        await discardStagedPaths(canvasId, data.versionId, [CANVAS_YAML_PATH]);
      } else {
        await stageSpecOperations(canvasId, data.versionId, [
          {
            path: CANVAS_YAML_PATH,
            content: encodeRepositoryFileContent(data.canvasYaml),
          },
        ]);
      }
      if (data.autoLayout && !canvasMatchesCommitted) {
        await applyCanvasStagingAutoLayout(canvasId, data.versionId, data.autoLayout);
      }

      const [describeResponse, canvasYaml] = await Promise.all([
        canvasesDescribeCanvasVersion(
          withOrganizationHeader({
            path: { canvasId, versionId: data.versionId },
          }),
        ),
        fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, data.versionId, true),
      ]);

      const version = versionWithSpecFromYaml(describeResponse.data?.version, canvasYaml);
      return { data: { canvasYaml, version, stagingSummary: describeResponse.data?.stagingSummary } };
    },
    onSuccess: (response, variables) => {
      const version = versionWithSpecFromYaml(response?.data?.version, response?.data?.canvasYaml);

      if (variables.versionId) {
        queryClient.setQueryData(
          canvasKeys.versionStaging(canvasId, variables.versionId),
          response?.data?.stagingSummary ?? { hasStaging: false, stagedPaths: [] },
        );
      }

      if (!version) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        return;
      }

      if (variables.versionId) {
        // `version` carries the effective staged spec (fetched with stage=true),
        // so it belongs to the staged cache entry the editor reads — not the
        // committed `versionDetail` used by the draft branch list.
        queryClient.setQueryData(canvasKeys.versionStagedDetail(canvasId, variables.versionId), version);
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
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
        queryClient.invalidateQueries({
          queryKey: canvasKeys.repositoryFile(canvasId, CANVAS_YAML_PATH, variables.versionId),
        });
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
      const cachedList = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
      const cachedSummary = cachedList?.find((canvas) => canvas.id === canvasId);
      const nodeCount = cachedDetail?.spec?.nodes?.length ?? cachedSummary?.nodes?.length ?? 0;

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

export type CanvasRunsFilters = {
  states?: CanvasesCanvasRunState[];
  results?: CanvasesCanvasRunResult[];
};

export const useDescribeRun = (canvasId: string, runId: string | null, enabled = true) => {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: canvasKeys.run(canvasId, runId!),
    queryFn: async () => {
      const response = await canvasesDescribeRun(
        withOrganizationHeader({
          path: {
            canvasId,
            runId: runId!,
          },
        }),
      );
      const described = response.data;
      if (!described?.run) {
        return described;
      }

      const current = queryClient.getQueryData<{ run?: CanvasesCanvasRun }>(canvasKeys.run(canvasId, runId!));
      return upsertRunIntoDescribeRunData(current, described.run);
    },
    refetchOnWindowFocus: false,
    enabled: !!canvasId && !!runId && enabled,
  });
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
    refetchInterval: 60_000,
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
  // Memory updates are pushed via the `memory_updated` websocket event
  // (see useCanvasWebsocket), so we no longer poll on an interval. The
  // websocket handler invalidates this query whenever memory changes from a
  // node execution, manual mutation, or another tab; a reconnect also
  // invalidates so we never miss updates received while disconnected.
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

// fetchCanvasConsoleData reads console.yaml from the repository and parses it
// into console data. Shared by useCanvasConsole and committed-baseline lookups
// so both reuse the same query cache entry (deduping the read).
export async function fetchCanvasConsoleData(
  canvasId: string,
  versionId: string | undefined,
  stage: boolean,
): Promise<CanvasConsoleData | undefined> {
  const spec = await fetchConsoleSpecFromRepository(canvasId, versionId, stage);
  if (!spec) {
    return undefined;
  }
  return consoleDataFromYaml(canvasId, versionId, spec.consoleYaml);
}

export const useCanvasConsole = (
  canvasId: string,
  versionId: string | undefined,
  enabled: boolean = true,
  stage = false,
) => {
  return useQuery({
    queryKey: stage ? canvasKeys.consoleStaged(canvasId, versionId) : canvasKeys.console(canvasId, versionId),
    queryFn: () => fetchCanvasConsoleData(canvasId, versionId, stage),
    enabled: enabled && !!canvasId,
    staleTime: 30_000,
  });
};

type UseUpdateCanvasConsoleOptions = {
  registerIgnoredCanvasVersionUpdatedEcho?: (savingVersionId?: string) => () => void;
  getMutationGeneration?: () => number;
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
      // Console edits are stage-only; write to the staged cache the editor reads.
      const queryKey = canvasKeys.consoleStaged(canvasId, versionId);
      const mutationGeneration = options?.getMutationGeneration?.() ?? 0;
      if (input.panels === undefined || input.layout === undefined) {
        return { previous: queryClient.getQueryData<CanvasConsoleData>(queryKey), queryKey, mutationGeneration };
      }

      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<CanvasConsoleData>(queryKey);
      queryClient.setQueryData(
        queryKey,
        toCanvasConsole(canvasId, versionId, { panels: input.panels, layout: input.layout }, previous),
      );
      return { previous, queryKey, mutationGeneration };
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

        const consoleMatchesCommitted = await matchesCommittedConsoleYaml(canvasId, versionId, consoleYaml);
        if (consoleMatchesCommitted) {
          await discardStagedPaths(canvasId, versionId, [CONSOLE_YAML_PATH]);
        } else {
          await stageSpecOperations(canvasId, versionId, [
            {
              path: CONSOLE_YAML_PATH,
              content: encodeRepositoryFileContent(consoleYaml),
            },
          ]);
        }

        const [describeResponse, spec] = await Promise.all([
          canvasesDescribeCanvasVersion(
            withOrganizationHeader({
              path: { canvasId, versionId },
            }),
          ),
          fetchConsoleSpecFromRepository(canvasId, versionId, true),
        ]);
        const consoleData = spec
          ? consoleDataFromYaml(canvasId, versionId, spec.consoleYaml)
          : consoleDataFromYaml(canvasId, versionId, consoleYaml);
        return {
          consoleData,
          stagingSummary: describeResponse.data?.stagingSummary,
        };
      } catch (error) {
        releaseCanvasVersionUpdatedEcho?.();
        throw error;
      }
    },
    onError: (_error, _input, context) => {
      if (!context) return;
      const latestGeneration = options?.getMutationGeneration?.() ?? context.mutationGeneration;
      if (context.mutationGeneration !== latestGeneration) {
        return;
      }
      queryClient.setQueryData(context.queryKey, context.previous);
    },
    onSuccess: (result, _input, context) => {
      const latestGeneration = options?.getMutationGeneration?.() ?? context?.mutationGeneration;
      if (context && context.mutationGeneration !== latestGeneration) {
        return;
      }
      if (result.consoleData) {
        queryClient.setQueryData(canvasKeys.consoleStaged(canvasId, versionId), result.consoleData);
      }
      if (versionId) {
        queryClient.setQueryData(
          canvasKeys.versionStaging(canvasId, versionId),
          result.stagingSummary ?? { hasStaging: false, stagedPaths: [] },
        );
      }
    },
  });
};

export type CanvasConsoleQueryResult = ReturnType<typeof useCanvasConsole>;
export type UpdateCanvasConsoleMutationResult = ReturnType<typeof useUpdateCanvasConsole>;

async function fetchRepositoryFileContent(
  canvasId: string,
  path: string,
  versionId?: string,
  stage = false,
): Promise<string> {
  return fetchRepositorySpecFileContent(canvasId, path, versionId, stage);
}

// fetchRepositoryFileContentCached reads raw repository-file content through the
// React Query cache so callers (the Files diff, committed baselines, selection)
// reuse and dedupe identical reads. Committed (stage=false) content only changes
// on publish/commit, so it is cached; staged (stage=true) content changes on
// every autosave, so it always refetches to stay correct.
export function fetchRepositoryFileContentCached(
  queryClient: QueryClient,
  canvasId: string,
  path: string,
  versionId: string | undefined,
  stage: boolean,
): Promise<string> {
  return queryClient.fetchQuery({
    queryKey: canvasKeys.repositoryFileContent(canvasId, path, versionId, stage),
    queryFn: () => fetchRepositorySpecFileContent(canvasId, path, versionId, stage),
    staleTime: stage ? 0 : 30_000,
  });
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
  stage = false,
) => {
  const normalizedPath = path ?? "";
  return useQuery({
    queryKey: canvasKeys.repositoryFile(canvasId, normalizedPath, versionId, stage),
    queryFn: async () => {
      const content = await fetchRepositoryFileContent(canvasId, normalizedPath, versionId, stage);
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
      autoLayout?: SpecAutoLayout;
    }) => {
      if (!input.versionId) {
        throw new Error("version id is required");
      }

      await stageCommitSpecOperations(canvasId, input.versionId, input.operations, input.autoLayout);
      return undefined;
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

// useStageCanvasSpecFiles writes canvas.yaml/console.yaml edits to the draft
// version's staging layer (no commit). Returns the updated StagingSummary.
export const useStageCanvasSpecFiles = (canvasId: string, versionId: string) => {
  return useMutation({
    mutationFn: async (operations: CanvasesCanvasRepositoryFileOperation[]) => {
      registerLocalStagingWrite(canvasId, versionId);
      const response = await canvasesStageCanvasRepositoryFile(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: { operations },
        }),
      );
      return response.data?.stagingSummary;
    },
  });
};

// useCommitCanvasStaging parses staged spec files into the draft version row and
// clears staging.
export const useCommitCanvasStaging = (canvasId: string, versionId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const response = await canvasesCommitCanvasStaging(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: {},
        }),
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.setQueryData(canvasKeys.versionStaging(canvasId, versionId), { hasStaging: false, stagedPaths: [] });
      // Committing staged edits only changes the draft version, not the live
      // canvas, so we skip invalidating canvasKeys.detail to avoid a
      // DescribeCanvas refetch while staying in edit mode.
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(canvasId, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionStaging(canvasId, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.console(canvasId, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFile(canvasId, CANVAS_YAML_PATH, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFile(canvasId, CONSOLE_YAML_PATH, versionId) });
    },
  });
};

// useDiscardCanvasStaging deletes staging rows for a draft version. Pass paths
// to revert specific files; omit to discard everything.
export const useDiscardCanvasStaging = (canvasId: string, versionId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (paths?: string[]) => {
      registerLocalStagingWrite(canvasId, versionId);
      const response = await canvasesDiscardCanvasStaging(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: paths && paths.length > 0 ? { paths } : {},
        }),
      );
      return response.data?.stagingSummary;
    },
    onSuccess: (stagingSummary) => {
      queryClient.setQueryData(
        canvasKeys.versionStaging(canvasId, versionId),
        stagingSummary ?? { hasStaging: false, stagedPaths: [] },
      );
      // Discarding staging reverts draft files but stays in the draft, so the
      // live canvas is unchanged; skip canvasKeys.detail to avoid DescribeCanvas.
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(canvasId, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.console(canvasId, versionId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId) });
    },
  });
};

// useStageRepositoryFiles stages arbitrary repository file edits (non-spec
// files like README.md) into the draft version's staging layer and refreshes
// the StagingSummary so the header switches to Reset/Commit. Unlike spec files,
// these are committed to git (not the version row) by CommitCanvasStaging.
export const useStageRepositoryFiles = (canvasId: string, versionId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (operations: CanvasesCanvasRepositoryFileOperation[]) => {
      registerLocalStagingWrite(canvasId, versionId);
      const response = await canvasesStageCanvasRepositoryFile(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: { operations },
        }),
      );
      return response.data?.stagingSummary;
    },
    onSuccess: (stagingSummary, operations) => {
      queryClient.setQueryData(
        canvasKeys.versionStaging(canvasId, versionId),
        stagingSummary ?? { hasStaging: false, stagedPaths: [] },
      );
      queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId) });

      const affectedPaths = new Set(
        operations.map((operation) => operation.path).filter((path): path is string => !!path),
      );
      for (const path of affectedPaths) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFile(canvasId, path, versionId) });
      }
    },
  });
};

// useDiscardRepositoryFilePaths reverts specific staged paths (used when a Files
// tab edit is undone back to the committed content), refreshing StagingSummary.
export const useDiscardRepositoryFilePaths = (canvasId: string, versionId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (paths: string[]) => {
      registerLocalStagingWrite(canvasId, versionId);
      const response = await canvasesDiscardCanvasStaging(
        withOrganizationHeader({
          path: { canvasId, versionId },
          body: paths.length > 0 ? { paths } : {},
        }),
      );
      return response.data?.stagingSummary;
    },
    onSuccess: (stagingSummary, paths) => {
      queryClient.setQueryData(
        canvasKeys.versionStaging(canvasId, versionId),
        stagingSummary ?? { hasStaging: false, stagedPaths: [] },
      );
      queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId) });
      for (const path of new Set(paths)) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFile(canvasId, path, versionId) });
      }
    },
  });
};

export type CanvasRepositoryFilesQueryResult = ReturnType<typeof useCanvasRepositoryFiles>;
export type CanvasRepositoryFileQueryResult = ReturnType<typeof useCanvasRepositoryFile>;
export type CommitCanvasRepositoryFilesMutationResult = ReturnType<typeof useCommitCanvasRepositoryFiles>;
