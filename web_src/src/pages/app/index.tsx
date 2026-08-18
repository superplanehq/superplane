import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { useQueryClient } from "@tanstack/react-query";
import debounce from "lodash.debounce";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type MutableRefObject } from "react";
import { flushSync } from "react-dom";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";
import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasRun,
  CanvasesCanvasVersion,
  ActionsAction,
  ComponentsEdge,
  ComponentsIntegrationRef,
  ComponentsConcurrencySpec,
  SuperplaneComponentsNode as ComponentsNode,
  OrganizationsIntegration,
} from "@/api-client";
import { canvasesReemitTriggerEvent } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  renderCanvasRunsSidebarPanel,
  renderCanvasVersionsSidebarPanel,
} from "@/pages/app/canvasToolSidebarPanelContent";
import { usePermissions } from "@/contexts/usePermissions";
import { useComponents } from "@/hooks/useComponentData";
import {
  canvasKeys,
  useCanvas,
  useCanvasMemoryEntries,
  useDescribeCanvasVersion,
  useCreateCanvasMemoryNamespace,
  useDeleteCanvasMemoryEntry,
  useUpdateCanvasMemoryNamespace,
  useEventExecutions,
  useDescribeRun,
  useInfiniteCanvasRuns,
  useInfiniteCanvasLiveVersions,
  useTriggers,
  useUpdateCanvasVersion,
  useWidgets,
} from "@/hooks/useCanvasData";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import { useCanvasStagingResync } from "@/hooks/useCanvasStagingResync";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { buildAutocompleteExampleObj } from "./buildAutocompleteExampleObj";
import { useAutocompleteExampleContext } from "./useAutocompleteExampleContext";
import { CommitStagingDialog } from "./CommitStagingDialog";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import { analytics } from "@/lib/analytics";
import { appPath } from "@/lib/appPaths";
import { filterVisibleConfiguration } from "@/lib/components";
import { getApiErrorMessage } from "@/lib/errors";
import { setCanvasStagingEchoUserId } from "@/lib/canvasStagingEcho";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { isFactoryApp, resolveCanvasFlowDirection } from "@/lib/canvasFlowDirection";
import { DefaultLayoutEngine } from "@/lib/layout";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";
import type { BuildingBlock } from "@/ui/BuildingBlocksSidebar";
import type { CanvasNode, CanvasPageProps, NewNodeData, NodeEditData } from "@/ui/CanvasPage";
import { CANVAS_SIDEBAR_STORAGE_KEY, CanvasPage, type MissingIntegration } from "@/ui/CanvasPage";
import { CanvasPageLoadingOverlay } from "@/ui/CanvasPage/CanvasPageLoadingOverlay";
import { resolveFitViewVersionId } from "@/ui/CanvasPage/fitView";
import { useAppPageAgentSuggestions } from "./useAppPageAgentSuggestions";
import { useAutoLayoutOnUpdatePreference } from "./useAutoLayoutOnUpdatePreference";
import {
  appendWorkflowFragment,
  removeWorkflowEdges,
  removeWorkflowNodes,
  useTopologyMutationCommit,
} from "./useTopologyMutationCommit";
import type { EventState, EventStateMap } from "@/ui/componentBase";
import type { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { ACTIVE_RUN_API_STATES, statusFiltersToApiFilters, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import type { CanvasEchoRelease, CanvasSaveResult, QueuedCanvasSaveRequest } from "./canvasSaveTypes";
import { deriveConsoleNodeStatuses } from "./console/deriveNodeStatuses";
import { useConsoleModeActions } from "./console/useConsoleModeActions";
import { useConsoleTriggerNode } from "./console/useConsoleTriggerNode";
import { WorkflowPageModeOverlays } from "./WorkflowPageModeOverlays";
import { useWorkflowViewSearchParams } from "./useWorkflowViewSearchParams";
import { useFilesModeActions } from "./files/useFilesModeActions";
import { useFilesHeaderState } from "./files/useFilesHeaderState";
import { useMemoryModeActions } from "./useMemoryModeActions";
import { useWorkflowHeaderEditActions } from "./useWorkflowHeaderEditActions";
import { useWorkflowViewModeActions } from "./useWorkflowViewModeActions";
import { useRunInspectionNavigation } from "./useRunInspectionNavigation";
import { canEditCanvasMemory, shouldLoadCanvasMemoryEntries } from "./lib/canvas-memory-access";
import { CanvasPageModals } from "./CanvasPageModals";
import { resolveEditableWorkflowSnapshot } from "./lib/editable-workflow-snapshot";
import {
  resolveCanvasForView,
  syncLoadedVersionToCanvasDetail,
  isHistoricalVersionSpecLoading,
} from "./lib/resolve-canvas-for-view";
import {
  clearLiveEditSessionDraftState,
  clearLiveEditSessionSearchParams,
  isActiveCanvasVersionCurrentLive,
  resetCommittedLiveCanvasDetail,
} from "./lib/live-edit-session";
import { useActivateCanvasVersionForEditing } from "./useActivateCanvasVersionForEditing";
import { useRefreshLatestLiveCanvasData } from "./useRefreshLatestLiveCanvasData";
import type { CanvasVersionListItem } from "./lib/canvas-versions";
import { canvasVersionShell, sortVersionsDesc } from "./lib/canvas-versions";
import { useAppDraftStagingData } from "./useAppDraftStagingData";
import { useDefaultAppTab } from "./useDefaultAppTab";
import { useCanvasEditVersionState } from "./useCanvasEditVersionState";
import { useEditSessionBootstrap } from "./useEditSessionBootstrap";
import { useDraftCanvasSpecSync } from "./useDraftCanvasSpecSync";
import { fetchCanvasStagingSummary } from "./lib/repository-spec-files";
import { useEnterLiveEditSession } from "./useEnterLiveEditSession";
import { useCanvasEchoReleaseGuards } from "./useCanvasEchoReleaseGuards";
import { useCanvasLifecycleEventHandlers } from "./useCanvasLifecycleEventHandlers";
import { useDraftStagingActions } from "./useDraftStagingActions";
import { useFactoryConfigureSession, type FactoryConfigureActions } from "./useFactoryConfigureSession";
import { useFactoryConfigureInitialLayout } from "./useFactoryConfigureInitialLayout";
import { resolveFactoryEmbedCanvasChrome, resolveFactoryEmbedSidebars } from "./factoryEmbedCanvasChrome";
import { executeCommitStaging } from "./lib/commit-staging-flow";
import { buildDuplicatedEdges, buildDuplicatedNodes } from "./lib/duplicate-nodes";
import { getNodeIntegrationName, overlayIntegrationWarnings } from "./lib/node-integrations";
import { renderCanvasNodeCustomField } from "./lib/render-canvas-node-custom-field";
import { buildCanvasYamlExportPayload, materializeCanvasSpec } from "./lib/workflow-spec-files";
import { getCustomFieldRenderer, getState, getStateMap } from "./mappers";
import { resolveExecutionErrors } from "./mappers/dash0";
import type { TriggerActionModal } from "./mappers/types";
import { useCancelExecutionHandler } from "./useCancelExecutionHandler";
import { useCanvasYamlDiffModal } from "./useCanvasYamlDiffModal";
import { useSpecFileAutosave } from "./useSpecFileAutosave";
import { buildAppFiles } from "./files/lib/app-files";
import { useDraftVisualDiff } from "./useDraftVisualDiff";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import { useRunCanvasData, useRunCanvasPresentation } from "./useRunCanvasData";
import { useVisibleNodeRuntimeMaps } from "./useVisibleNodeRuntimeMaps";
import { useGetSidebarData } from "./useGetSidebarData";
import { useRunParticipantFitRequest } from "./useRunParticipantFitRequest";
import { useAgentNodeFocusRequest, type CanvasFocusRequest } from "./useAgentNodeFocusRequest";
import { isRunDetailDismissed, useRunsDetailState } from "./useRunsDetailState";
import { useComponentIconMap } from "./useComponentIconMap";
import { useRunSidebarNavigationState } from "./useRunSidebarNavigationState";
import { useCanvasAutoFocusPreference } from "@/hooks/useCanvasAutoFocusPreference";
import { useSelectedRunCanvas } from "./useSelectedRunCanvas";
import {
  clearComponentSidebarSearchParams,
  clampWorkflowViewFlagsForFactoryApp,
  getExitEditModeDisabledTooltip,
  getRunActionState,
  getWorkflowViewPresentation,
  allowsRunsSidebar,
  isNonCanvasAppViewParam,
  useWorkflowUrlViewFlags,
  clearRunInspectionSearchParams,
} from "./viewState";
import {
  buildExecutionInfo,
  buildTabData,
  generateNodeId,
  generateUniqueNodeName,
  getWorkflowSaveSignature,
} from "./utils";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import { runPositionAutoSave } from "./runPositionAutoSave";
import { syncRunInspectionViewportTransition } from "./lib/run-inspection-viewport";
import {
  clearRunDetailNodeSearchParams,
  buildCanvasLogEntries,
  getCanvasLogNodesSignature,
  getNodeAnalyticsProps,
  getRunningRunsCount,
  isCanvasLoadNotFoundError,
  isCanvasPrepLoading,
  isValidRunId,
  prepareCanvasLogNodes,
  prepareData,
  shouldClearRunDetailNode,
} from "./workflowPageHelpers";
const VERSION_ACTION_SAVE_SETTLE_TIMEOUT_MS = 5000;
const EMPTY_CANVAS_SPEC_ITEMS: never[] = [];
const RUNNING_RUNS_FILTERS = { states: [...ACTIVE_RUN_API_STATES] };

function getCanvasVersionEditPermissionState({
  canEditCanvasDraft,
  canvasDeletedRemotely,
}: {
  canEditCanvasDraft: boolean;
  canvasDeletedRemotely: boolean;
}) {
  if (canvasDeletedRemotely) {
    return {
      canStageCanvasVersion: false,
      tooltip: "This canvas was deleted in another session.",
    };
  }

  if (!canEditCanvasDraft) {
    return {
      canStageCanvasVersion: false,
      tooltip: "You don't have permission to edit this canvas.",
    };
  }

  return {
    canStageCanvasVersion: true,
    tooltip: undefined,
  };
}

function canEditCanvasDraftVersion(canUpdateCanvas: boolean, canAct: (resource: string, action: string) => boolean) {
  return canUpdateCanvas || canAct("canvases", "update_version");
}

function whenAllowed<T>(allowed: boolean, value: T): T | undefined {
  return allowed ? value : undefined;
}

/** Factory apps keep nodes always expanded — omit collapse toggle. */
function resolveFactoryAwareToggleView(
  isReadOnly: boolean,
  factoryOwnedApp: boolean,
  handler: (nodeId: string, collapsed: boolean) => void,
): ((nodeId: string, collapsed: boolean) => void) | undefined {
  if (isReadOnly || factoryOwnedApp) {
    return undefined;
  }
  return handler;
}

function resolveFactoryAutoCanvasProps(
  enabled: boolean,
  readOnly: boolean,
  callbacks: { [Key in keyof FactoryLayoutCanvasCallbacks]-?: NonNullable<CanvasPageProps[Key]> },
): Pick<
  CanvasPageProps,
  | "layoutMode"
  | "onAutoLayoutNodes"
  | "onNodePositionChange"
  | "onNodesPositionChange"
  | "onPlaceholderAdd"
  | "onPlaceholderConfigure"
> {
  if (readOnly) return { layoutMode: enabled ? "factory-auto" : "manual" };
  if (enabled) {
    return {
      layoutMode: "factory-auto",
      onPlaceholderAdd: callbacks.onPlaceholderAdd,
      onPlaceholderConfigure: callbacks.onPlaceholderConfigure,
    };
  }
  return { layoutMode: "manual", ...callbacks };
}

type FactoryLayoutCanvasCallbacks = Pick<
  CanvasPageProps,
  "onAutoLayoutNodes" | "onNodePositionChange" | "onNodesPositionChange" | "onPlaceholderAdd" | "onPlaceholderConfigure"
>;

export type { FactoryConfigureActions } from "./useFactoryConfigureSession";

export function AppPage({
  factoryEmbed = false,
  factoryConfigure = false,
  factoryConfigureActionsRef,
  onFactoryConfigureBusyChange,
  onFactoryConfigureDone,
}: {
  factoryEmbed?: boolean;
  /** Factory-shell Configure: same chrome as embed, but canvas edit session allowed. */
  factoryConfigure?: boolean;
  /** Imperative Discard/Save handlers for the factory Configure chrome (no setState bridge). */
  factoryConfigureActionsRef?: MutableRefObject<FactoryConfigureActions | null>;
  onFactoryConfigureBusyChange?: (busy: boolean) => void;
  onFactoryConfigureDone?: () => void;
} = {}) {
  const factoryViewOnly = factoryEmbed && !factoryConfigure;
  const {
    organizationId,
    appId,
    factoryId: routeFactoryId,
  } = useParams<{
    organizationId: string;
    appId?: string;
    factoryId?: string;
  }>();
  const canvasId = appId || "";
  const agentSuggestions = useAppPageAgentSuggestions(appId);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { data: me } = useMe();
  const {
    isRunInspectionMode,
    isConsoleAddPanelOpen,
    setIsConsoleAddPanelOpen,
    isConsoleYamlOpen,
    setIsConsoleYamlOpen,
    selectedRunId,
  } = useWorkflowViewSearchParams(searchParams, setSearchParams);
  const preserveRunDetailNodeOnNextRunChangeRef = useRef(false);
  const clearRunDetailNodeSearch = useCallback(() => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete("sidebar");
        next.delete("node");
        return next;
      },
      { replace: true },
    );
  }, [setSearchParams]);
  const {
    openRunDetailOnMount,
    runDetailNodeId,
    setRunDetailNodeId,
    clearDismissedRunDetail,
    detailDismissedForRunId,
    handleBackToRunList,
  } = useRunsDetailState(searchParams, isRunInspectionMode, selectedRunId, preserveRunDetailNodeOnNextRunChangeRef, {
    canvasId,
    onBackToRunList: clearRunDetailNodeSearch,
  });
  const rawUrlViewFlags = useWorkflowUrlViewFlags(searchParams);
  const { filesHeaderActionsSlotId } = useFilesHeaderState(canvasId);
  const currentUserId = me?.id;
  useEffect(() => {
    setCanvasStagingEchoUserId(currentUserId);
  }, [currentUserId]);
  const { canAct } = usePermissions();
  const [activeCanvasVersion, setActiveCanvasVersion] = useState<CanvasesCanvasVersion | null>(null);
  // True while the user is in an edit session. The session keeps the versions
  // sidebar visible even when previewing the current/old version (not editing a
  // draft), and ends only when the user explicitly exits.
  const [editSessionActive, setEditSessionActive] = useState(false);
  const [isEnteringEditSession, setIsEnteringEditSession] = useState(false);
  const editSessionActiveRef = useRef(false);
  const activeCanvasVersionIdRef = useRef<string>("");
  // Distinguishes "user deliberately selected Current version from the sidebar"
  // (session stays open) from "landed back on live because the draft was
  // published/discarded" (session must close). Both states look identical in
  // terms of active-version state (no selected version), so we can't derive it.
  const previewingCurrentVersionRef = useRef(false);
  const draftCanvasSpecsRef = useRef<Map<string, CanvasesCanvas["spec"] | null>>(new Map());
  const updateCanvasVersionMutation = useUpdateCanvasVersion(canvasId!);
  const [commitDialogOpen, setCommitDialogOpen] = useState(false);
  const [isCanvasSaveInFlight, setIsCanvasSaveInFlight] = useState(false);
  const [isCanvasSaveQueued, setIsCanvasSaveQueued] = useState(false);
  const [isPreparingVersionAction, setIsPreparingVersionAction] = useState(false);
  const [stagingResetNonce, setStagingResetNonce] = useState(0);
  const flushRepositoryFileStagingRef = useRef<(() => Promise<void>) | null>(null);
  const handleFlushRepositoryFileStagingReady = useCallback((flush: (() => Promise<void>) | null) => {
    flushRepositoryFileStagingRef.current = flush;
  }, []);
  const flushRepositoryFileStaging = useCallback(async () => {
    await flushRepositoryFileStagingRef.current?.();
  }, []);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: widgets = [], isLoading: widgetsLoading } = useWidgets();
  const { data: availableIntegrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();
  const canReadIntegrations = canAct("integrations", "read");
  const canUpdateIntegrations = canAct("integrations", "update");
  const canUseAgents = !factoryEmbed && canAct("agents", "create") && canAct("agents", "read");
  const { data: integrations = [] } = useConnectedIntegrations(organizationId!, { enabled: canReadIntegrations });
  const {
    data: liveCanvas,
    isLoading: canvasLoading,
    isFetching: canvasFetching,
    error: canvasError,
  } = useCanvas(organizationId!, canvasId!, {
    enabled: true,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  });
  const factoryOwnedApp = isFactoryApp(liveCanvas?.metadata?.factoryId);
  const factoryAutoLayout = factoryConfigure && factoryOwnedApp;
  const urlViewFlags = useMemo(
    () => (factoryOwnedApp ? clampWorkflowViewFlagsForFactoryApp(rawUrlViewFlags) : rawUrlViewFlags),
    [factoryOwnedApp, rawUrlViewFlags],
  );
  useEffect(() => {
    if (!factoryOwnedApp) {
      return;
    }

    const view = searchParams.get("view") ?? "";
    if (!isNonCanvasAppViewParam(view)) {
      return;
    }

    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete("view");
        next.delete("file");
        return next;
      },
      { replace: true },
    );
  }, [factoryOwnedApp, searchParams, setSearchParams]);
  const liveCanvasVersionId = liveCanvas?.metadata?.liveVersionId;
  const { data: liveCanvasVersion, isLoading: liveCanvasVersionLoading } = useDescribeCanvasVersion(
    canvasId!,
    liveCanvasVersionId,
  );
  const canvasLiveVersionsQuery = useInfiniteCanvasLiveVersions(organizationId!, canvasId!, true);
  const paginatedVersions = useMemo(
    () => (canvasLiveVersionsQuery.data?.pages || []).flatMap((page) => page?.versions || []),
    [canvasLiveVersionsQuery.data?.pages],
  );
  const liveVersions = useMemo(() => sortVersionsDesc(paginatedVersions), [paginatedVersions]);
  const selectableVersionsById = useMemo(() => {
    const indexedVersions = new Map<string, CanvasVersionListItem>();
    paginatedVersions.forEach((version) => {
      const id = version.id;
      if (!id) return;
      indexedVersions.set(id, version);
    });
    return indexedVersions;
  }, [paginatedVersions]);
  const hasMoreLiveVersions = canvasLiveVersionsQuery.hasNextPage || false;
  const isLoadingMoreLiveVersions = canvasLiveVersionsQuery.isFetchingNextPage;
  const refreshLatestLiveCanvasData = useRefreshLatestLiveCanvasData(organizationId, canvasId, liveCanvasVersionId);
  const {
    activeCanvasVersionId,
    shouldReadStagedCanvasVersionFlag,
    loadedCanvasVersion,
    loadedCanvasVersionLoading,
    loadedCanvasVersionFetching,
    isAwaitingStagedCanvasSpecFlag,
    selectedCanvasVersion,
    isViewingCurrentLiveVersion,
    isViewingLiveVersion,
    isEditing,
    hasEditableVersion,
    showLiveActivity,
  } = useCanvasEditVersionState({
    organizationId: organizationId!,
    canvasId: canvasId!,
    editSessionActive,
    isEnteringEditSession,
    activeCanvasVersion,
    liveCanvasVersionId,
    selectableVersionsById,
    isRunInspectionMode,
    isMemoryMode: urlViewFlags.isMemoryMode,
  });
  const [draftCanvasSpec, setDraftCanvasSpec] = useState<CanvasesCanvas["spec"] | null>(null);
  const draftSpecToRender = draftCanvasSpec ?? selectedCanvasVersion?.spec ?? null;
  const {
    committedBaselinesForEdit,
    isEditBootstrapReady,
    draftSpecForView,
    isDraftCanvasLoading,
    isEditSessionUiReady,
    stableCanvasViewKey,
    canvasRenderKey,
  } = useEditSessionBootstrap({
    canvasId: canvasId!,
    isEditing,
    isEnteringEditSession,
    shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
    activeCanvasVersionId,
    stagingResetNonce,
    draftCanvasSpec,
    draftSpecToRender,
    loadedCanvasVersionLoading,
    loadedCanvasVersionFetching,
    selectedCanvasVersion,
    liveCanvasVersionId,
    isRunInspectionMode,
    includeConsoleBaseline: !factoryOwnedApp,
  });
  // Factory Configure seeds draft up front; never block the shell on baseline/resync fetches.
  const showDraftCanvasLoadingOverlay = isDraftCanvasLoading && !(factoryConfigure && draftCanvasSpec);
  useDraftCanvasSpecSync({
    isEditing,
    isEnteringEditSession,
    shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
    isAwaitingStagedCanvasSpec: isAwaitingStagedCanvasSpecFlag,
    activeCanvasVersionId,
    selectedCanvasVersion,
    draftCanvasSpec,
    setDraftCanvasSpec,
    draftCanvasSpecsRef,
    liveCanvas,
    liveCanvasVersion,
  });

  const canvas = useMemo(
    () =>
      resolveCanvasForView({
        isEditing,
        isViewingCurrentLiveVersion,
        liveCanvas,
        draftSpecToRender: draftSpecForView,
        selectedCanvasVersion,
        canvasId: canvasId!,
      }),
    [liveCanvas, selectedCanvasVersion, isEditing, isViewingCurrentLiveVersion, draftSpecForView, canvasId],
  );
  const versionCanvasLoading = useMemo(
    () =>
      isHistoricalVersionSpecLoading({
        activeCanvasVersionId,
        liveCanvasVersionId,
        shouldReadStagedCanvasVersion: shouldReadStagedCanvasVersionFlag,
        loadedCanvasVersion,
        loadedCanvasVersionLoading,
        loadedCanvasVersionFetching,
      }),
    [
      activeCanvasVersionId,
      liveCanvasVersionId,
      shouldReadStagedCanvasVersionFlag,
      loadedCanvasVersion,
      loadedCanvasVersionLoading,
      loadedCanvasVersionFetching,
    ],
  );
  const canvasForPrep = canvas ?? ((isEditing || isEnteringEditSession) && liveCanvas ? liveCanvas : null);
  const canvasNodes = canvas?.spec?.nodes ?? EMPTY_CANVAS_SPEC_ITEMS;

  const [runStatusFilters, setRunStatusFilters] = useState<RunStatusFilter[]>([]);
  const runApiFilters = useMemo(
    () => (isRunInspectionMode && selectedRunId ? {} : statusFiltersToApiFilters(runStatusFilters)),
    [isRunInspectionMode, selectedRunId, runStatusFilters],
  );
  const infiniteRunsQuery = useInfiniteCanvasRuns(canvasId!, runApiFilters, showLiveActivity);
  const infiniteLogRunsQuery = useInfiniteCanvasRuns(canvasId!, {}, isViewingLiveVersion);
  const infiniteRunningRunsQuery = useInfiniteCanvasRuns(canvasId!, RUNNING_RUNS_FILTERS, isViewingLiveVersion);
  const selectedRunIdIsValid = selectedRunId ? isValidRunId(selectedRunId) : false;
  const describedRunQuery = useDescribeRun(
    canvasId!,
    selectedRunId,
    isRunInspectionMode && !!selectedRunId && selectedRunIdIsValid,
  );
  const runsData = useMemo(() => {
    const pages = infiniteRunsQuery.data?.pages || [];
    const seen = new Set<string>();
    const runs = pages
      .flatMap((page) => page?.runs || [])
      .filter((run): run is CanvasesCanvasRun => {
        if (!run.id || seen.has(run.id)) return false;
        seen.add(run.id);
        return true;
      });
    const totalCount = pages[0]?.totalCount || 0;
    return { runs, totalCount };
  }, [infiniteRunsQuery.data]);
  const logRunsData = useMemo(() => {
    const pages = infiniteLogRunsQuery.data?.pages || [];
    const seen = new Set<string>();
    const runs = pages
      .flatMap((page) => page?.runs || [])
      .filter((run): run is CanvasesCanvasRun => {
        if (!run.id || seen.has(run.id)) return false;
        seen.add(run.id);
        return true;
      });
    return { runs };
  }, [infiniteLogRunsQuery.data]);
  const runningRunsCount = getRunningRunsCount(infiniteRunningRunsQuery.data, isViewingLiveVersion);
  const selectedRunFromList = useMemo(
    () => runsData.runs.find((run) => run.id === selectedRunId) || null,
    [runsData.runs, selectedRunId],
  );
  const selectedRun = useMemo(() => {
    if (!selectedRunId) return null;
    if (selectedRunFromList?.id === selectedRunId) {
      return selectedRunFromList;
    }
    if (isRunInspectionMode) {
      return describedRunQuery.data?.run ?? null;
    }
    return selectedRunFromList;
  }, [describedRunQuery.data?.run, isRunInspectionMode, selectedRunFromList, selectedRunId]);
  const isSelectedRunLoading =
    isRunInspectionMode && !!selectedRunId && selectedRunIdIsValid && !selectedRun && describedRunQuery.isLoading;
  const describeQueryEnabled = isRunInspectionMode && !!selectedRunId && selectedRunIdIsValid;
  const describeRunSettled = !describeQueryEnabled || describedRunQuery.isFetched;
  const selectedRunExecutionsQuery = useEventExecutions(canvasId!, selectedRun?.rootEvent?.id ?? null);
  const selectedRunFullExecutions = selectedRunExecutionsQuery.data?.executions;
  const { selectedRunCanvas, isSelectedRunVersionLoading } = useSelectedRunCanvas({
    organizationId: organizationId!,
    canvasId: canvasId!,
    selectedRun,
    isRunInspectionMode,
    liveCanvasVersionId,
    canvas,
    liveCanvas,
  });
  const componentIconMap = useComponentIconMap(components, triggers);
  const { runFilterState, runNavigation } = useRunSidebarNavigationState({
    runs: runsData.runs,
    selectedRunId,
    hasNextPage: infiniteRunsQuery.hasNextPage,
    workflowNodes: canvasNodes,
    componentIconMap,
    onStatusFiltersChange: setRunStatusFilters,
  });
  const {
    data: canvasMemoryEntries = [],
    isLoading: canvasMemoryLoading,
    error: canvasMemoryError,
  } = useCanvasMemoryEntries(canvasId!, shouldLoadCanvasMemoryEntries(urlViewFlags.isMemoryMode, isViewingLiveVersion));
  const deleteCanvasMemoryEntry = useDeleteCanvasMemoryEntry(canvasId!);
  const createCanvasMemoryNamespace = useCreateCanvasMemoryNamespace(canvasId!);
  const updateCanvasMemoryNamespace = useUpdateCanvasMemoryNamespace(canvasId!);
  const canUpdateCanvas = canAct("canvases", "update");
  const canEditCanvasDraft = canEditCanvasDraftVersion(canUpdateCanvas, canAct);
  usePageTitle([canvas?.metadata?.name || "Canvas"]);
  const [canvasDeletedRemotely, setCanvasDeletedRemotely] = useState(false);
  const [remoteCanvasUpdatePending, setRemoteCanvasUpdatePending] = useState(false);
  const canvasAccess = { canUpdateCanvas, canvasDeletedRemotely };
  const canActOnCanvas = canUpdateCanvas && !canvasDeletedRemotely;
  const canvasVersionEditPermission = getCanvasVersionEditPermissionState({
    canEditCanvasDraft,
    canvasDeletedRemotely,
  });
  const canStageCanvasVersion = canvasVersionEditPermission.canStageCanvasVersion;
  const isReadOnly = !canStageCanvasVersion || !hasEditableVersion;
  /**
   * Track if we've already done the initial fit to view.
   * This ref persists across re-renders to prevent viewport changes on save.
   */
  const hasFitToViewRef = useRef(false);
  const runsHasFitToViewRef = useRef(false);
  // Canvas/version the persisted viewport was fitted for; switching content re-fits the graph.
  const lastFittedContentKeyRef = useRef<string | null>(null);
  const hasSyncedVersionFromURLRef = useRef(false);

  /**
   * Capture the initial node focus from the URL so we only zoom once.
   */
  const initialFocusNodeIdRef = useRef<string | null>(null);
  if (initialFocusNodeIdRef.current === null) {
    initialFocusNodeIdRef.current = searchParams.get("node") || null;
  }

  /**
   * Track if the user has manually toggled the building blocks sidebar.
   * This ref persists across re-renders to preserve user preference.
   */
  const hasUserToggledSidebarRef = useRef(false);

  /**
   * Track the building blocks sidebar state.
   * Initialize based on whether nodes exist (open if no nodes).
   * This ref persists across re-renders to preserve sidebar state.
   */
  const isSidebarOpenRef = useRef<boolean | null>(null);
  if (isSidebarOpenRef.current === null && typeof window !== "undefined") {
    const storedSidebarState = window.localStorage.getItem(CANVAS_SIDEBAR_STORAGE_KEY);
    if (storedSidebarState !== null) {
      try {
        isSidebarOpenRef.current = JSON.parse(storedSidebarState);
        hasUserToggledSidebarRef.current = true;
      } catch (error) {
        console.warn("Failed to parse sidebar state from local storage:", error);
      }
    }
  }
  if (isSidebarOpenRef.current === null && canvas) {
    isSidebarOpenRef.current = canvas.spec?.nodes?.length === 0;
  }

  /**
   * Track the canvas viewport state.
   * This ref persists across re-renders to preserve viewport position and zoom.
   */
  const viewportRef = useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);
  const runsViewportRef = useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);
  const lastRunsViewportKeyRef = useRef<"runs" | null>(null);

  const [isPositionAutoSaveQueued, setIsPositionAutoSaveQueued] = useState(false);
  const [isAnnotationAutoSaveQueued, setIsAnnotationAutoSaveQueued] = useState(false);

  const isAutoSaveQueued = isPositionAutoSaveQueued || isAnnotationAutoSaveQueued;
  const hasLocalSaveActivity = isCanvasSaveInFlight || isCanvasSaveQueued || isAutoSaveQueued;
  const { handleToggleAutoLayoutOnUpdate, isAutoLayoutOnUpdateEnabled } = useAutoLayoutOnUpdatePreference();
  const { handleToggleAutoFocus, isAutoFocusEnabled } = useCanvasAutoFocusPreference();

  const lastSavedWorkflowSignatureRef = useRef("");
  const lastAppliedVersionSnapshotRef = useRef("");
  const canvasRef = useRef<CanvasesCanvas | null>(canvas ?? null);
  // Tracks which version the rendered workflow in `canvasRef` currently
  // represents. `activeCanvasVersionIdRef` flips synchronously when switching
  // drafts, but `canvasRef` only catches up a render later; auto-save must not
  // persist the previous draft's graph onto the newly-active draft while they
  // disagree.
  const canvasContentVersionIdRef = useRef<string>("");
  const queuedCanvasSaveRef = useRef<QueuedCanvasSaveRequest | null>(null);
  const isDrainingCanvasSaveQueueRef = useRef(false);
  const hasTrackedCanvasView = useRef(false);
  const canvasSaveSessionRef = useRef(0);
  const consoleMutationGenerationRef = useRef(0);
  const handleRemoteStagingUpdatedRef = useRef<() => Promise<void>>(async () => {});
  const ignoredCanvasUpdatedEchoReleasesRef = useRef<Array<CanvasEchoRelease>>([]);
  const { registerIgnoredCanvasUpdatedEcho, consumeIgnoredCanvasUpdatedEcho, resetLifecycleEchoGuards } =
    useCanvasEchoReleaseGuards({
      canvasSaveSessionRef,
      ignoredCanvasUpdatedEchoReleasesRef,
    });
  const setLastSavedWorkflowSnapshot = useCallback((workflow: CanvasesCanvas | null) => {
    if (!workflow) {
      lastSavedWorkflowSignatureRef.current = "";
      return;
    }

    lastSavedWorkflowSignatureRef.current = getWorkflowSaveSignature(workflow);
  }, []);
  const clearQueuedAutoSaveFlags = useCallback(() => {
    setIsPositionAutoSaveQueued(false);
    setIsAnnotationAutoSaveQueued(false);
  }, []);
  useEffect(() => {
    canvasRef.current = canvas ?? null;
    // `canvas` and `activeCanvasVersionId` settle together on the render that
    // applies a draft switch, so stamp the content's version atomically here.
    canvasContentVersionIdRef.current = activeCanvasVersionId;
  }, [canvas, activeCanvasVersionId]);
  useEffect(() => {
    activeCanvasVersionIdRef.current = activeCanvasVersionId;
  }, [activeCanvasVersionId]);

  useEffect(() => {
    editSessionActiveRef.current = editSessionActive;
  }, [editSessionActive]);

  const applyLocalWorkflowUpdate = useCallback(
    (updatedWorkflow: CanvasesCanvas) => {
      if (!organizationId || !canvasId) {
        return;
      }

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (!isEditing || !activeCanvasVersionId || !updatedWorkflow.spec) {
        return;
      }

      draftCanvasSpecsRef.current.set(activeCanvasVersionId, updatedWorkflow.spec);
      setDraftCanvasSpec(updatedWorkflow.spec);
      setActiveCanvasVersion((current) =>
        current?.metadata?.id === activeCanvasVersionId ? { ...current, spec: updatedWorkflow.spec } : current,
      );
      queryClient.setQueryData<CanvasesCanvasVersion | undefined>(canvasKeys.stagedCanvasSpec(canvasId), (current) =>
        current
          ? { ...current, spec: updatedWorkflow.spec }
          : {
              metadata: { id: activeCanvasVersionId },
              spec: updatedWorkflow.spec,
            },
      );
    },
    [organizationId, canvasId, queryClient, isEditing, activeCanvasVersionId],
  );

  const getCurrentWorkflowSnapshot = useCallback(() => {
    const renderedWorkflow = canvasRef.current;
    if (!organizationId || !canvasId) {
      return renderedWorkflow;
    }

    const detailWorkflow = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
    return resolveEditableWorkflowSnapshot({
      isEditing,
      renderedWorkflow,
      detailWorkflow,
    });
  }, [organizationId, canvasId, queryClient, isEditing]);

  // Use Zustand store for execution data - extract only the methods to avoid recreating callbacks
  // Subscribe to version to ensure React detects all updates
  const storeVersion = useNodeExecutionStore((state) => state.version);
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);
  const loadNodeDataMethod = useNodeExecutionStore((state) => state.loadNodeData);
  const refetchNodeDataMethod = useNodeExecutionStore((state) => state.refetchNodeData);
  const initializeFromWorkflow = useNodeExecutionStore((state) => state.initializeFromWorkflow);

  // Redirect to home page if workflow is not found (404)
  // Use replace to avoid back button issues and prevent 404 flash
  useEffect(() => {
    if (!canvasError || canvasLoading) {
      return;
    }
    if (isCanvasLoadNotFoundError(canvasError) && organizationId && !canvasDeletedRemotely) {
      navigate(`/${organizationId}`, { replace: true });
    }
  }, [canvasError, canvasLoading, navigate, organizationId, canvasDeletedRemotely]);
  useEffect(() => {
    if (hasTrackedCanvasView.current) return;
    if (!canvas || !canvasId || !organizationId || canvasLoading) return;
    hasTrackedCanvasView.current = true;
    analytics.canvasView(canvasId, canvas.spec?.nodes?.length ?? 0, canvas.spec?.edges?.length ?? 0, organizationId);
  }, [canvas, canvasId, organizationId, canvasLoading]);
  // Initialize store from workflow.status on workflow load.
  // On canvas switch with cached data, the store initializes immediately from the
  // cache (no loading gap) and then re-initializes once when the background refetch
  // completes with fresh data (pendingStoreReinitRef).
  const hasInitializedStoreRef = useRef<string | null>(null);
  const pendingStoreReinitRef = useRef(false);
  useEffect(() => {
    if (!canvas?.metadata?.id) return;

    if (hasInitializedStoreRef.current !== canvas.metadata.id) {
      initializeFromWorkflow(canvas);
      hasInitializedStoreRef.current = canvas.metadata.id;
      if (!canvasFetching) {
        pendingStoreReinitRef.current = false;
      }
      return;
    }

    if (pendingStoreReinitRef.current && !canvasFetching) {
      initializeFromWorkflow(canvas);
      pendingStoreReinitRef.current = false;
    }
  }, [canvas, canvasFetching, initializeFromWorkflow]);

  useEffect(() => {
    if (!canvas || lastSavedWorkflowSignatureRef.current) {
      return;
    }

    setLastSavedWorkflowSnapshot(canvas);
  }, [canvas, setLastSavedWorkflowSnapshot]);

  useEffect(() => {
    canvasSaveSessionRef.current += 1;

    const queuedRequest = queuedCanvasSaveRef.current;
    if (queuedRequest) {
      queuedCanvasSaveRef.current = null;
      queuedRequest.resolve({
        status: "replaced",
        workflow: queuedRequest.workflow,
        savingVersionId: queuedRequest.savingVersionId,
        matchesCurrentCanvas: false,
        hasQueuedFollowUp: false,
      });
    }

    hasTrackedCanvasView.current = false;
    setActiveCanvasVersion(null);
    hasSyncedVersionFromURLRef.current = false;
    setLastSavedWorkflowSnapshot(null);
    resetLifecycleEchoGuards();
    draftCanvasSpecsRef.current.clear();
    isDrainingCanvasSaveQueueRef.current = false;
    setIsCanvasSaveInFlight(false);
    setIsCanvasSaveQueued(false);
    hasInitializedStoreRef.current = null;
    pendingStoreReinitRef.current = true;
  }, [canvasId, resetLifecycleEchoGuards, setLastSavedWorkflowSnapshot]);

  useEffect(() => {
    if (hasSyncedVersionFromURLRef.current || selectableVersionsById.size === 0 || activeCanvasVersionId) {
      return;
    }

    const requestedVersionID = searchParams.get("version");
    if (!requestedVersionID) {
      hasSyncedVersionFromURLRef.current = true;
      return;
    }

    if (selectedCanvasVersion?.metadata?.id === requestedVersionID) {
      return;
    }

    const requestedVersion = selectableVersionsById.get(requestedVersionID);
    if (!requestedVersion) {
      hasSyncedVersionFromURLRef.current = true;
      return;
    }

    const requestedVersionId = requestedVersion.id || "";
    const isCurrentLive = requestedVersionId === liveCanvasVersionId;

    if (isCurrentLive) {
      setActiveCanvasVersion(null);
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        next.delete("version");
        return clearComponentSidebarSearchParams(next);
      });
      hasSyncedVersionFromURLRef.current = true;
      return;
    }

    setActiveCanvasVersion(canvasVersionShell(requestedVersion));
    hasSyncedVersionFromURLRef.current = true;
  }, [
    selectableVersionsById,
    activeCanvasVersionId,
    selectedCanvasVersion?.metadata?.id,
    searchParams,
    currentUserId,
    liveCanvasVersionId,
    setSearchParams,
    queryClient,
    organizationId,
    canvasId,
  ]);

  useEffect(() => {
    syncLoadedVersionToCanvasDetail({
      organizationId,
      canvasId,
      activeCanvasVersionId,
      loadedCanvasVersion,
      hasLocalSaveActivity,
      isEditing,
      isViewingCurrentLiveVersion,
      queryClient,
      lastAppliedVersionSnapshotRef,
    });
  }, [
    organizationId,
    canvasId,
    activeCanvasVersionId,
    loadedCanvasVersion,
    queryClient,
    hasLocalSaveActivity,
    isEditing,
    isViewingCurrentLiveVersion,
  ]);

  useEffect(() => {
    if (!remoteCanvasUpdatePending || hasLocalSaveActivity || canvasDeletedRemotely || !organizationId || !canvasId) {
      return;
    }

    if (isViewingLiveVersion) {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    } else if (activeCanvasVersionId) {
      draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
    }

    setRemoteCanvasUpdatePending(false);
    void handleRemoteStagingUpdatedRef.current();
  }, [
    remoteCanvasUpdatePending,
    hasLocalSaveActivity,
    canvasDeletedRemotely,
    organizationId,
    canvasId,
    queryClient,
    isViewingLiveVersion,
    activeCanvasVersionId,
  ]);

  // Build maps from store for canvas display (using initial data from workflow.status and websocket updates)
  // Rebuild whenever store version changes (indicates data was updated)
  const { nodeExecutionsMap, nodeQueueItemsMap, nodeEventsMap } = useMemo<{
    nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
    nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
    nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  }>(() => {
    void storeVersion;
    const executionsMap: Record<string, CanvasesCanvasNodeExecution[]> = {};
    const queueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]> = {};
    const eventsMap: Record<string, CanvasesCanvasEvent[]> = {};

    // Get current store data
    const storeData = useNodeExecutionStore.getState().data;

    storeData.forEach((data, nodeId) => {
      if (data.executions.length > 0) {
        executionsMap[nodeId] = data.executions;
      }
      if (data.queueItems.length > 0) {
        queueItemsMap[nodeId] = data.queueItems;
      }
      if (data.events.length > 0) {
        eventsMap[nodeId] = data.events;
      }
    });

    return { nodeExecutionsMap: executionsMap, nodeQueueItemsMap: queueItemsMap, nodeEventsMap: eventsMap };
  }, [storeVersion]);
  const { showNodeRuntimeActivity, visibleNodeExecutionsMap, visibleNodeQueueItemsMap, visibleNodeEventsMap } =
    useVisibleNodeRuntimeMaps({
      showLiveActivity,
      factoryEmbed,
      isRunInspectionMode,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      nodeEventsMap,
    });
  const consoleNodeStatuses = useMemo(
    () => deriveConsoleNodeStatuses(visibleNodeExecutionsMap),
    [visibleNodeExecutionsMap],
  );
  const handleConsoleTriggerNode = useConsoleTriggerNode({ canvasId, canvas: canvas ?? undefined, queryClient });

  const {
    stagingStale,
    commitCanvasStagingMutation,
    discardCanvasStagingMutation,
    consoleQuery,
    updateConsoleMutation,
    draftChangeIndicators,
    canvasConsoleVersionDiff,
    handleEffectiveConsoleChange,
    handleLocalFilesStagingChange,
    hasStagingChanges,
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
    hasFilesStagingChanges,
  } = useAppDraftStagingData({
    canvasId: canvasId!,
    activeCanvasVersionId,
    liveCanvasVersionId,
    isEditing,
    hasEditableVersion,
    stagingResetNonce,
    draftSpecToRender,
    canvas,
    getConsoleMutationGeneration: () => consoleMutationGenerationRef.current,
    committedBaselines: committedBaselinesForEdit,
    // Factory Configure seeds draft before baselines finish; still treat as ready so
    // Save can see local graph diffs and open the commit dialog.
    editBootstrapReady: isEditBootstrapReady || (factoryConfigure && Boolean(draftCanvasSpec)),
  });

  useDefaultAppTab({ canvasId, urlViewFlags, searchParams });

  const syncCurrentCanvasWithSavedVersion = useCallback(
    (workflow: CanvasesCanvas, version?: CanvasesCanvasVersion) => {
      if (!organizationId || !canvasId || !version?.spec) {
        return;
      }

      // Mark the saved version as already applied so the version sync effect
      // (which replaces canvas spec with loadedCanvasVersion.spec) doesn't
      // overwrite the merged positions we set below.
      const versionId = version.metadata?.id;
      lastAppliedVersionSnapshotRef.current =
        versionId && versionId === activeCanvasVersionIdRef.current
          ? `${versionId}:${version.metadata?.updatedAt || ""}`
          : lastAppliedVersionSnapshotRef.current;

      queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
        if (!current || getWorkflowSaveSignature(current) !== getWorkflowSaveSignature(workflow)) {
          return current;
        }

        const currentPositionsByNodeId = new Map(
          (current.spec?.nodes ?? [])
            .filter((node) => node.id && node.position)
            .map((node) => [node.id, node.position]),
        );

        const mergedNodes = (version.spec?.nodes ?? []).map((serverNode) => {
          const localPosition = currentPositionsByNodeId.get(serverNode.id);
          if (localPosition) {
            return { ...serverNode, position: localPosition };
          }
          return serverNode;
        });

        return {
          ...current,
          metadata: {
            ...current.metadata,
            name: workflow.metadata?.name ?? current.metadata?.name,
            description: workflow.metadata?.description ?? current.metadata?.description,
          },
          spec: { ...current.spec, ...version.spec, nodes: mergedNodes },
        };
      });
    },
    [organizationId, canvasId, queryClient],
  );

  const saveMatchesCurrentCanvas = useCallback(
    (workflow: CanvasesCanvas) => {
      const currentWorkflow = getCurrentWorkflowSnapshot();
      return getWorkflowSaveSignature(currentWorkflow) === getWorkflowSaveSignature(workflow);
    },
    [getCurrentWorkflowSnapshot],
  );

  const processQueuedCanvasSave = useCallback(
    async (saveSession: number, request: QueuedCanvasSaveRequest) => {
      if (canvasSaveSessionRef.current !== saveSession) {
        request.resolve({
          status: "stale",
          workflow: request.workflow,
          savingVersionId: request.savingVersionId,
          matchesCurrentCanvas: false,
          hasQueuedFollowUp: false,
        });
        return;
      }

      if (activeCanvasVersionIdRef.current !== (request.savingVersionId || "")) {
        request.resolve({
          status: "stale",
          workflow: request.workflow,
          savingVersionId: request.savingVersionId,
          matchesCurrentCanvas: false,
          hasQueuedFollowUp: !!queuedCanvasSaveRef.current,
        });
        return;
      }

      const releaseCanvasUpdatedEcho = registerIgnoredCanvasUpdatedEcho();

      try {
        const response = await updateCanvasVersionMutation.mutateAsync({
          versionId: request.savingVersionId,
          canvasYaml: materializeCanvasSpec(request.workflow),
        });

        if (canvasSaveSessionRef.current !== saveSession) {
          request.resolve({
            status: "stale",
            workflow: request.workflow,
            savingVersionId: request.savingVersionId,
            matchesCurrentCanvas: false,
            hasQueuedFollowUp: false,
          });
          return;
        }

        syncCurrentCanvasWithSavedVersion(request.workflow, response?.data?.version);

        request.resolve({
          status: "saved",
          workflow: request.workflow,
          savingVersionId: request.savingVersionId,
          response,
          matchesCurrentCanvas: saveMatchesCurrentCanvas(request.workflow),
          hasQueuedFollowUp: !!queuedCanvasSaveRef.current,
        });
      } catch (error) {
        releaseCanvasUpdatedEcho();
        request.reject(error);
      }
    },
    [
      registerIgnoredCanvasUpdatedEcho,
      saveMatchesCurrentCanvas,
      syncCurrentCanvasWithSavedVersion,
      updateCanvasVersionMutation,
    ],
  );

  const drainCanvasSaveQueue = useCallback(async () => {
    if (isDrainingCanvasSaveQueueRef.current || !organizationId || !canvasId) {
      return;
    }

    const saveSession = canvasSaveSessionRef.current;
    isDrainingCanvasSaveQueueRef.current = true;
    setIsCanvasSaveInFlight(true);

    try {
      while (queuedCanvasSaveRef.current && canvasSaveSessionRef.current === saveSession) {
        const request = queuedCanvasSaveRef.current;
        queuedCanvasSaveRef.current = null;
        setIsCanvasSaveQueued(false);
        await processQueuedCanvasSave(saveSession, request);
      }
    } finally {
      if (canvasSaveSessionRef.current === saveSession) {
        setIsCanvasSaveInFlight(false);
        isDrainingCanvasSaveQueueRef.current = false;
        if (queuedCanvasSaveRef.current) {
          void drainCanvasSaveQueue();
        }
      }
    }
  }, [organizationId, canvasId, processQueuedCanvasSave]);

  const enqueueCanvasSave = useCallback(
    (workflow: CanvasesCanvas, savingVersionId?: string) =>
      new Promise<CanvasSaveResult>((resolve, reject) => {
        const replacedRequest = queuedCanvasSaveRef.current;
        if (replacedRequest) {
          replacedRequest.resolve({
            status: "replaced",
            workflow: replacedRequest.workflow,
            savingVersionId: replacedRequest.savingVersionId,
            matchesCurrentCanvas: false,
            hasQueuedFollowUp: true,
          });
        }

        queuedCanvasSaveRef.current = { workflow, savingVersionId, resolve, reject };
        setIsCanvasSaveQueued(true);
        void drainCanvasSaveQueue();
      }),
    [drainCanvasSaveQueue],
  );

  const clearQueuedCanvasSave = useCallback(() => {
    const queuedRequest = queuedCanvasSaveRef.current;
    if (!queuedRequest) {
      return;
    }

    queuedCanvasSaveRef.current = null;
    setIsCanvasSaveQueued(false);
    queuedRequest.resolve({
      status: "replaced",
      workflow: queuedRequest.workflow,
      savingVersionId: queuedRequest.savingVersionId,
      matchesCurrentCanvas: false,
      hasQueuedFollowUp: false,
    });
  }, []);

  /**
   * Ref to track pending position updates that need to be auto-saved.
   * Maps node ID to its updated position.
   */
  const pendingPositionUpdatesRef = useRef<Map<string, { x: number; y: number }>>(new Map());
  const pendingAnnotationUpdatesRef = useRef<
    Map<
      string,
      { text?: string; color?: string; width?: number; height?: number; label?: string; description?: string }
    >
  >(new Map());

  /**
   * Debounced auto-save function for node position changes.
   * Uses a short delay while the canvas is editable; a longer delay when read-only
   * so stray timers from a mode switch do not fire too aggressively.
   * Only saves position changes, not structural modifications (deletions, additions, etc).
   * If there are unsaved structural changes, position auto-save is skipped.
   */
  const debouncedAutoSave = useMemo(
    () =>
      debounce(
        () =>
          runPositionAutoSave({
            setIsPositionAutoSaveQueued,
            organizationId,
            canvasId,
            pendingPositionUpdatesRef,
            isReadOnly,
            isEditing,
            canvasRef,
            queryClient,
            activeCanvasVersionIdRef,
            activeCanvasVersionId,
            canvasContentVersionIdRef,
            enqueueCanvasSave,
            setActiveCanvasVersion,
            applyLocalWorkflowUpdate,
            setLastSavedWorkflowSnapshot,
          }),
        isReadOnly ? 2000 : 100,
      ),
    [
      organizationId,
      canvasId,
      activeCanvasVersionId,
      queryClient,
      isReadOnly,
      isEditing,
      enqueueCanvasSave,
      applyLocalWorkflowUpdate,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const queuePositionAutoSave = useCallback(
    (updates: Map<string, { x: number; y: number }>) => {
      if (isReadOnly || updates.size === 0) {
        return;
      }

      updates.forEach((position, nodeId) => {
        pendingPositionUpdatesRef.current.set(nodeId, position);
      });
      setIsPositionAutoSaveQueued(true);
      debouncedAutoSave();
    },
    [debouncedAutoSave, isReadOnly],
  );

  const handleNodeWebsocketEvent = useCallback(
    (nodeId: string, event: string) => {
      if (event.includes("event_created")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeEventHistory(canvasId!, nodeId),
        });
      }

      if (event.startsWith("execution")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeExecutionHistory(canvasId!, nodeId),
        });
      }

      if (event.startsWith("queue_item")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeQueueItemHistory(canvasId!, nodeId),
        });
      }
    },
    [queryClient, canvasId],
  );

  // Warn user before leaving page with unsaved changes
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (hasLocalSaveActivity) {
        e.preventDefault();
        e.returnValue = "Your work isn't saved, unsaved changes will be lost. Are you sure you want to leave?";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasLocalSaveActivity]);

  // Merge triggers and components from applications into the main arrays
  const allTriggers = useMemo(() => {
    const merged = [...triggers];
    availableIntegrations.forEach((integration) => {
      if (integration.capabilities) {
        merged.push(...triggersFromCapabilities(integration.capabilities));
      }
    });
    return merged;
  }, [triggers, availableIntegrations]);

  const allComponents = useMemo(() => {
    const merged = [...components];
    availableIntegrations.forEach((integration) => {
      if (integration.capabilities) {
        merged.push(...actionsFromCapabilities(integration.capabilities));
      }
    });
    return merged;
  }, [components, availableIntegrations]);

  const buildingBlocks = useMemo(
    () =>
      buildBuildingBlockCategories(triggers, components, availableIntegrations, {
        isFactoryApp: Boolean(canvas?.metadata?.factoryId),
      }),
    [triggers, components, availableIntegrations, canvas?.metadata?.factoryId],
  );
  const canvasEdges = canvas?.spec?.edges ?? EMPTY_CANVAS_SPEC_ITEMS;
  const canvasNodesById = useMemo(() => {
    const nodesById = new Map<string, ComponentsNode>();
    canvasNodes.forEach((node) => {
      if (node.id) {
        nodesById.set(node.id, node);
      }
    });
    return nodesById;
  }, [canvasNodes]);
  const incomingNodeIdsByTargetId = useMemo(() => {
    const incomingByTargetId = new Map<string, string[]>();
    canvasEdges.forEach((edge) => {
      if (!edge.targetId || !edge.sourceId) {
        return;
      }

      const incoming = incomingByTargetId.get(edge.targetId) || [];
      incoming.push(edge.sourceId);
      incomingByTargetId.set(edge.targetId, incoming);
    });
    return incomingByTargetId;
  }, [canvasEdges]);
  const allComponentsByName = useMemo(
    () => new Map(allComponents.map((component) => [component.name, component])),
    [allComponents],
  );
  const allTriggersByName = useMemo(
    () => new Map(allTriggers.map((trigger) => [trigger.name, trigger])),
    [allTriggers],
  );
  const widgetsByName = useMemo(() => new Map(widgets.map((widget) => [widget.name, widget])), [widgets]);
  const availableIntegrationsByName = useMemo(
    () => new Map(availableIntegrations.map((integration) => [integration.name, integration])),
    [availableIntegrations],
  );
  const integrationNameByComponentName = useMemo(() => {
    const namesByComponent = new Map<string, string>();
    availableIntegrations.forEach((integration) => {
      integration.capabilities?.forEach((capability) => {
        if (capability.name && integration.name) {
          namesByComponent.set(capability.name, integration.name);
        }
      });
    });
    return namesByComponent;
  }, [availableIntegrations]);
  const readyIntegrationNames = useMemo(() => {
    const names = new Set<string>();
    integrations.forEach((integration) => {
      const integrationName = integration.metadata?.integrationName;
      if (integrationName && integration.status?.state === "ready") {
        names.add(integrationName);
      }
    });
    return names;
  }, [integrations]);
  const nonReadyIntegrationsByName = useMemo(() => {
    const integrationsByName = new Map<string, OrganizationsIntegration>();
    integrations.forEach((integration) => {
      const integrationName = integration.metadata?.integrationName;
      if (integrationName && integration.status?.state !== "ready" && !integrationsByName.has(integrationName)) {
        integrationsByName.set(integrationName, integration);
      }
    });
    return integrationsByName;
  }, [integrations]);
  const canvasMode = hasEditableVersion ? "edit" : "live";
  const triggerModalHostRef = useRef<((modal: TriggerActionModal) => void) | undefined>(undefined);
  const runDisabledRef = useRef(false);
  const runDisabledTooltipRef = useRef<string | undefined>(undefined);
  const registerTriggerModalHost = useCallback((openModal: (modal: TriggerActionModal) => void) => {
    triggerModalHostRef.current = openModal;
  }, []);
  const openTriggerModal = useCallback((modal: TriggerActionModal) => {
    if (runDisabledRef.current) {
      if (runDisabledTooltipRef.current) {
        showErrorToast(runDisabledTooltipRef.current);
      }
      return;
    }
    triggerModalHostRef.current?.(modal);
  }, []);

  const dataLoading = isCanvasPrepLoading(
    canvasForPrep,
    canvasLoading,
    triggersLoading,
    componentsLoading,
    integrationsLoading,
  );
  const { nodes: preparedNodes, edges: preparedEdges } = useMemo(() => {
    if (dataLoading || !canvasForPrep) {
      return { nodes: [], edges: [] };
    }

    return prepareData(
      canvasForPrep,
      allTriggers,
      allComponents,
      visibleNodeEventsMap,
      visibleNodeExecutionsMap,
      visibleNodeQueueItemsMap,
      canvasId!,
      queryClient,
      me,
      canvasMode,
      openTriggerModal,
    );
  }, [
    canvasForPrep,
    allTriggers,
    allComponents,
    visibleNodeEventsMap,
    visibleNodeExecutionsMap,
    visibleNodeQueueItemsMap,
    canvasId,
    queryClient,
    dataLoading,
    me,
    canvasMode,
    openTriggerModal,
  ]);

  const draftVisualDiff = useDraftVisualDiff({
    isViewingDraftVersion: isEditing && isEditBootstrapReady,
    canvas,
    liveCanvasVersion,
    latestDraftVersion: liveCanvasVersion,
    selectedCanvasVersion,
    preparedNodes,
    preparedEdges,
    allTriggers,
    allComponents,
    canvasId,
    queryClient,
    includeRemovedNodeGhosts: !factoryOwnedApp,
  });

  const nodesWithIntegrationStatus = useMemo(
    () => overlayIntegrationWarnings(draftVisualDiff.nodes, integrations, canvasNodes),
    [draftVisualDiff.nodes, integrations, canvasNodes],
  );

  const runCanvasData = useRunCanvasData({
    isRunInspectionMode,
    selectedRun,
    selectedRunCanvas,
    canvasLoading,
    triggersLoading,
    componentsLoading,
    isSelectedRunVersionLoading,
    allTriggers,
    allComponents,
    canvasId,
    queryClient,
    me,
    visibleNodeExecutionsMap,
    selectedRunFullExecutions,
  });

  const {
    nodes,
    edges: renderedEdges,
    runCanvasLoading,
  } = useRunCanvasPresentation({
    isRunInspectionMode,
    selectedRun,
    runCanvasData,
    liveNodes: nodesWithIntegrationStatus,
    liveEdges: draftVisualDiff.edges,
    isSelectedRunLoading,
    isSelectedRunVersionLoading,
    isSelectedRunExecutionsLoading: selectedRunExecutionsQuery.isLoading,
  });
  syncRunInspectionViewportTransition({
    isRunInspectionMode,
    liveViewportRef: viewportRef,
    runsViewportRef,
    liveHasFitToViewRef: hasFitToViewRef,
    runsHasFitToViewRef,
    lastRunsViewportKeyRef,
  });

  useEffect(() => {
    if (
      runDetailNodeId &&
      shouldClearRunDetailNode({
        runDetailNodeId,
        participantNodeIds: runCanvasData?.participantNodeIds ?? [],
        runCanvasLoading,
        runCanvasSettled:
          Boolean(selectedRun) &&
          !canvasLoading &&
          !triggersLoading &&
          !componentsLoading &&
          !isSelectedRunVersionLoading &&
          !selectedRunExecutionsQuery.isLoading,
      })
    ) {
      setRunDetailNodeId(null);
      setSearchParams((current) => clearRunDetailNodeSearchParams(current, runDetailNodeId), { replace: true });
    }
  }, [
    canvasLoading,
    componentsLoading,
    isSelectedRunVersionLoading,
    runCanvasData,
    runCanvasLoading,
    runDetailNodeId,
    selectedRun,
    selectedRunExecutionsQuery.isLoading,
    selectedRunId,
    setRunDetailNodeId,
    setSearchParams,
    triggersLoading,
  ]);

  const getSidebarData = useGetSidebarData({
    canvasNodes,
    canvasNodesById,
    allComponents,
    allTriggers,
    visibleNodeEventsMap,
    showNodeRuntimeActivity,
    getNodeData,
  });

  // Trigger data loading when sidebar opens for a node
  const loadSidebarData = useCallback(
    (nodeId: string) => {
      if (!isViewingLiveVersion) {
        return;
      }

      const node = canvasNodesById.get(nodeId);
      if (!node) return;

      // Set current history node for tracking
      setCurrentHistoryNode({ nodeId, nodeType: node?.type || "TYPE_ACTION" });

      loadNodeDataMethod(canvasId!, nodeId, node.type!, queryClient);
    },
    [canvasNodesById, canvasId, queryClient, loadNodeDataMethod, isViewingLiveVersion],
  );

  const onCancelQueueItem = useOnCancelQueueItemHandler({
    canvasId: canvasId!,
    organizationId,
    canvas,
    loadSidebarData,
  });

  const [currentHistoryNode, setCurrentHistoryNode] = useState<{ nodeId: string; nodeType: string } | null>(null);
  const [focusRequest, setFocusRequest] = useState<CanvasFocusRequest | null>(null);
  const handleSidebarChange = useCallback(
    (open: boolean, nodeId: string | null) => {
      // Use the functional updater so this composes with other concurrent
      // search-param updates (e.g. switching drafts updating `branch`). Building
      // from a stale `searchParams` snapshot would clobber those updates and, for
      // example, re-add the previous `branch` and prevent switching drafts while
      // the node sidebar is open.
      setSearchParams(
        (current) => {
          const hasSidebar = current.get("sidebar") === "1";
          const currentNode = current.get("node");
          if (open) {
            if (hasSidebar && currentNode === (nodeId ?? null)) {
              return current;
            }
            const next = new URLSearchParams(current);
            next.set("sidebar", "1");
            if (nodeId) {
              next.set("node", nodeId);
            } else {
              next.delete("node");
            }
            return next;
          }

          if (!hasSidebar && !currentNode) {
            return current;
          }

          const next = new URLSearchParams(current);
          next.delete("sidebar");
          next.delete("node");
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const handleLogNodeSelect = useCallback(
    (nodeId: string) => {
      handleSidebarChange(true, nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), targetMode: "live", tab: "settings" });
    },
    [handleSidebarChange],
  );

  const handleLogRunNodeSelect = useCallback((nodeId: string) => {
    setFocusRequest({ nodeId, requestId: Date.now(), targetMode: "live", tab: "latest" });
  }, []);

  const handleRunItemOpen = useCallback(
    (nodeId: string | undefined, executionStatus: string, errorMessage?: string) => {
      const node = nodeId ? canvas?.spec?.nodes?.find((n) => n.id === nodeId) : undefined;
      const { nodeRef } = node ? getNodeAnalyticsProps(node, availableIntegrations) : { nodeRef: undefined };
      analytics.canvasRunItemOpen(nodeRef, executionStatus, organizationId ?? "");
      if (errorMessage) {
        analytics.canvasComponentError(nodeRef, errorMessage, organizationId ?? "");
      }
    },
    [canvas, availableIntegrations, organizationId],
  );

  const { resyncStagedEditorState } = useCanvasStagingResync({
    organizationId,
    canvasId,
    activeCanvasVersionIdRef,
    draftCanvasSpecsRef,
    consoleMutationGenerationRef,
    setDraftCanvasSpec,
    setActiveCanvasVersion,
    setLastSavedWorkflowSnapshot,
    setStagingResetNonce,
  });

  const { handleCanvasLifecycleEvent, shouldApplyCanvasUpdate, handleCanvasStagingEvent } =
    useCanvasLifecycleEventHandlers({
      canvasId,
      currentUserId,
      editSessionActiveRef,
      hasLocalSaveActivity,
      isViewingLiveVersion,
      canvasDeletedRemotely,
      consumeIgnoredCanvasUpdatedEcho,
      onRemoteStagingUpdated: () => {
        void handleRemoteStagingUpdatedRef.current();
      },
      setCanvasDeletedRemotely,
      setRemoteCanvasUpdatePending,
    });

  useCanvasWebsocket(
    canvasId!,
    organizationId!,
    handleNodeWebsocketEvent,
    undefined,
    undefined,
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
    isViewingLiveVersion,
    true,
    handleCanvasStagingEvent,
  );
  const rawLogNodes = prepareCanvasLogNodes(canvasNodes, canvasEdges, allComponents, !dataLoading);
  const logNodesSignature = useMemo(() => getCanvasLogNodesSignature(rawLogNodes), [rawLogNodes]);
  const logNodesRef = useRef<{ signature: string; nodes: ComponentsNode[] }>({ signature: "", nodes: [] });
  const logNodes = useMemo(() => {
    if (logNodesRef.current.signature === logNodesSignature) {
      return logNodesRef.current.nodes;
    }
    logNodesRef.current = { signature: logNodesSignature, nodes: rawLogNodes };
    return rawLogNodes;
  }, [rawLogNodes, logNodesSignature]);

  const logEntries = useMemo(
    () => buildCanvasLogEntries(logNodes, canvas?.metadata?.updatedAt || "", handleLogNodeSelect),
    [handleLogNodeSelect, canvas?.metadata?.updatedAt, logNodes],
  );
  const nodeHistoryQuery = useNodeHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION",
    allNodes: canvasNodes,
    enabled: !!currentHistoryNode && !!canvasId && isViewingLiveVersion,
  });

  const queueHistoryQuery = useQueueHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    allNodes: canvasNodes,
    enabled: !!currentHistoryNode && !!canvasId && isViewingLiveVersion,
  });

  const getAllHistoryEvents = useCallback(
    (nodeId: string): SidebarEvent[] => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.getAllHistoryEvents();
      }

      return [];
    },
    [currentHistoryNode, nodeHistoryQuery],
  );
  // Load more history for a specific node
  const handleLoadMoreHistory = useCallback(
    (nodeId: string) => {
      if (!currentHistoryNode || currentHistoryNode.nodeId !== nodeId) {
        setCurrentHistoryNode({ nodeId, nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION" });
      } else {
        nodeHistoryQuery.handleLoadMore();
      }
    },
    [currentHistoryNode, nodeHistoryQuery],
  );

  const getHasMoreHistory = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.hasMoreHistory;
      }
      return false;
    },
    [currentHistoryNode, nodeHistoryQuery.hasMoreHistory],
  );

  const getLoadingMoreHistory = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.isLoadingMore;
      }
      return false;
    },
    [currentHistoryNode, nodeHistoryQuery.isLoadingMore],
  );

  const onLoadMoreQueue = useCallback(
    (nodeId: string) => {
      if (!currentHistoryNode || currentHistoryNode.nodeId !== nodeId) {
        setCurrentHistoryNode({ nodeId, nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION" });
      } else {
        queueHistoryQuery.handleLoadMore();
      }
    },
    [currentHistoryNode, queueHistoryQuery],
  );

  const getAllQueueEvents = useCallback(
    (nodeId: string): SidebarEvent[] => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.getAllHistoryEvents();
      }

      return [];
    },
    [currentHistoryNode, queueHistoryQuery],
  );

  const getHasMoreQueue = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.hasMoreHistory;
      }
      return false;
    },
    [currentHistoryNode, queueHistoryQuery.hasMoreHistory],
  );

  const getLoadingMoreQueue = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.isLoadingMore;
      }
      return false;
    },
    [currentHistoryNode, queueHistoryQuery.isLoadingMore],
  );

  /**
   * Builds a topological path to find all nodes that should execute before the given target node.
   * This follows the directed graph structure of the workflow to determine execution order.
   */

  const getTabData = useCallback(
    (nodeId: string, event: SidebarEvent): TabData | undefined => {
      return buildTabData(nodeId, event, {
        workflowNodes: canvasNodes,
        nodeEventsMap: visibleNodeEventsMap,
        nodeExecutionsMap: visibleNodeExecutionsMap,
        nodeQueueItemsMap: visibleNodeQueueItemsMap,
      });
    },
    [canvasNodes, visibleNodeExecutionsMap, visibleNodeEventsMap, visibleNodeQueueItemsMap],
  );

  const autocompleteExampleContext = useAutocompleteExampleContext({
    canvas,
    canvasNodes,
    canvasNodesById,
    incomingNodeIdsByTargetId,
    visibleNodeExecutionsMap,
    visibleNodeEventsMap,
    allComponentsByName,
    allTriggersByName,
  });
  const getAutocompleteExampleObj = useCallback(
    (nodeId: string) => buildAutocompleteExampleObj(nodeId, autocompleteExampleContext),
    [autocompleteExampleContext],
  );

  const handleSaveWorkflow = useCallback(
    async (workflowToSave?: CanvasesCanvas, options?: { showToast?: boolean; savingVersionId?: string }) => {
      const targetWorkflow = workflowToSave || canvasRef.current;
      if (!targetWorkflow || !organizationId || !canvasId) return;
      if (!canStageCanvasVersion) {
        if (options?.showToast !== false) {
          showErrorToast("You don't have permission to edit this canvas version");
        }
        return;
      }
      // Callers that build the payload from a specific render (e.g. the node
      // config panel, which can flush on unmount after a draft switch) pass the
      // version their content belongs to. This keeps content and target version
      // bound together instead of reading the live ref, which may already point
      // at a different draft and would otherwise cross-write the two drafts.
      const savingVersionID =
        options?.savingVersionId || activeCanvasVersionIdRef.current || activeCanvasVersionId || undefined;
      if (!savingVersionID) {
        if (options?.showToast !== false) {
          showErrorToast("Enable edit mode before saving changes");
        }
        return;
      }
      // When relying on the rendered workflow (no explicit payload), make sure
      // it represents the active version. Right after a draft switch the
      // version id flips before `canvasRef` catches up, and persisting the
      // stale graph would overwrite the newly-active draft with the previous
      // draft's content.
      if (!workflowToSave && !options?.savingVersionId && canvasContentVersionIdRef.current !== savingVersionID) {
        return;
      }
      const shouldRestoreFocus = options?.showToast === false;
      const focusedNoteId = shouldRestoreFocus ? getActiveNoteId() : null;

      try {
        const result = await enqueueCanvasSave(targetWorkflow, savingVersionID);
        if (result.status !== "saved") {
          return result;
        }

        if (result.response?.data?.version && savingVersionID && activeCanvasVersionIdRef.current === savingVersionID) {
          setActiveCanvasVersion(result.response.data.version);
        }
        if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
          return result;
        }

        if (options?.showToast !== false) {
          showSuccessToast("Canvas changes saved");
        }
        setLastSavedWorkflowSnapshot(targetWorkflow);

        return result;
      } catch (error: any) {
        const errorMessage = getApiErrorMessage(error, "Failed to save changes to the canvas");
        const displayMessage = getUsageLimitToastMessage(error, errorMessage);
        showErrorToast(displayMessage);
        return undefined;
      } finally {
        if (focusedNoteId) {
          requestAnimationFrame(() => {
            restoreActiveNoteFocus();
          });
        }
      }
    },
    [
      organizationId,
      canvasId,
      activeCanvasVersionId,
      canStageCanvasVersion,
      enqueueCanvasSave,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const commitTopologyMutation = useTopologyMutationCommit({
    factoryAutoLayout,
    autoLayoutOnUpdate: isAutoLayoutOnUpdateEnabled,
    components,
    getCurrentWorkflow: getCurrentWorkflowSnapshot,
    applyLocalWorkflow: applyLocalWorkflowUpdate,
    saveWorkflow: handleSaveWorkflow,
    saveSessionRef: canvasSaveSessionRef,
    readOnly: isReadOnly,
  });
  const applyInitialFactoryLayout = useCallback(
    () => commitTopologyMutation((workflow) => workflow),
    [commitTopologyMutation],
  );
  useFactoryConfigureInitialLayout({
    factoryAutoLayout,
    isEditing,
    editBootstrapReady: isEditBootstrapReady,
    activeCanvasVersionId,
    applyLayout: applyInitialFactoryLayout,
  });

  const getNodeEditData = useCallback(
    (nodeId: string): NodeEditData | null => {
      const node = canvasNodesById.get(nodeId);
      if (!node) return null;

      // Get configuration fields from metadata based on node type
      let configurationFields: ActionsAction["configuration"] = [];
      let displayLabel: string | undefined = node.name || undefined;
      let integrationName: string | undefined;
      let integrationLabel: string | undefined;
      let blockName: string | undefined;

      if (node.type === "TYPE_ACTION" || node.type === "TYPE_TRIGGER") {
        const metadata =
          node.type === "TYPE_ACTION" ? allComponentsByName.get(node.component) : allTriggersByName.get(node.component);
        configurationFields = metadata?.configuration || [];
        displayLabel = metadata?.label || displayLabel;
        blockName = node.component;
        integrationName = node.component ? integrationNameByComponentName.get(node.component) : undefined;
        integrationLabel = integrationName ? availableIntegrationsByName.get(integrationName)?.label : undefined;
      } else if (node.type === "TYPE_WIDGET") {
        const widget = widgetsByName.get(node.component);
        if (widget) {
          configurationFields = widget.configuration || [];
          displayLabel = widget.label || "Widget";
        }

        return {
          nodeId: node.id!,
          nodeName: node.name!,
          displayLabel,
          configuration: {
            text: node.configuration?.text || "",
            color: node.configuration?.color || "yellow",
          },
          configurationFields,
          integrationName,
          integrationLabel,
          blockName,
          integrationRef: node.integration,
        };
      }

      return {
        nodeId: node.id!,
        nodeName: node.name!,
        displayLabel,
        configuration: node.configuration || {},
        configurationFields,
        integrationName,
        integrationLabel,
        blockName,
        integrationRef: node.integration,
        concurrency: node.concurrency,
        // Merge admits every run's queue item, so it is inherently
        // unbounded and takes no concurrency configuration. Loop only
        // honors max, as its parallel-session cap.
        supportsConcurrency: node.type === "TYPE_ACTION" && node.component !== "merge",
        concurrencyMaxOnly: node.component === "loop",
      };
    },
    [
      canvasNodesById,
      allComponentsByName,
      allTriggersByName,
      availableIntegrationsByName,
      integrationNameByComponentName,
      widgetsByName,
    ],
  );

  const createIntegrationMutation = useCreateIntegration(organizationId ?? "", "node_configuration");
  const [integrationDialogName, setIntegrationDialogName] = useState<string | null>(null);
  const [justConnectedIntegrations, setJustConnectedIntegrations] = useState<Set<string>>(new Set());

  const integrationDialogDefinition = useMemo(
    () => (integrationDialogName ? availableIntegrations.find((d) => d.name === integrationDialogName) : undefined),
    [availableIntegrations, integrationDialogName],
  );

  const integrationDialogPendingInstance = useMemo(() => {
    if (!integrationDialogName) return undefined;
    return integrations.find(
      (i) => i.metadata?.integrationName === integrationDialogName && i.status?.state !== "ready",
    );
  }, [integrationDialogName, integrations]);

  const initialWebhookSetup = useMemo(() => {
    const webhookUrl = getIntegrationWebhookUrl(integrationDialogPendingInstance?.status?.metadata);
    if (!webhookUrl || !integrationDialogPendingInstance?.metadata?.id) return undefined;
    return {
      id: integrationDialogPendingInstance.metadata.id,
      webhookUrl,
      config: { ...(integrationDialogPendingInstance.spec?.configuration ?? {}) },
    };
  }, [integrationDialogPendingInstance]);

  const missingIntegrations: MissingIntegration[] = useMemo(() => {
    if (!canReadIntegrations) return [];

    const missingMap = new Map<
      string,
      {
        count: number;
        definition?: (typeof availableIntegrations)[0];
        state?: "pending" | "error";
        stateDescription?: string;
      }
    >();

    for (const node of canvasNodes) {
      const integrationName = node.component ? integrationNameByComponentName.get(node.component) : undefined;
      if (!integrationName) continue;

      if (readyIntegrationNames.has(integrationName)) continue;

      const existing = missingMap.get(integrationName);
      if (existing) {
        existing.count++;
      } else {
        const nonReadyInstance = nonReadyIntegrationsByName.get(integrationName);
        const rawState = nonReadyInstance?.status?.state;
        missingMap.set(integrationName, {
          count: 1,
          definition: availableIntegrationsByName.get(integrationName),
          state: rawState === "error" ? "error" : rawState === "pending" ? "pending" : undefined,
          stateDescription: nonReadyInstance?.status?.stateDescription,
        });
      }
    }

    return Array.from(missingMap.entries()).map(([name, { count, definition, state, stateDescription }]) => ({
      integrationName: name,
      affectedNodeCount: count,
      definition,
      justConnected: !state && justConnectedIntegrations.has(name),
      state,
      stateDescription,
    }));
  }, [
    canvasNodes,
    integrationNameByComponentName,
    readyIntegrationNames,
    nonReadyIntegrationsByName,
    availableIntegrationsByName,
    canReadIntegrations,
    justConnectedIntegrations,
  ]);

  const handleConnectIntegration = useCallback((integrationName: string) => {
    setIntegrationDialogName(integrationName);
  }, []);

  // Listen for agent sidebar integration button clicks
  const [agentConfigureIntegrationId, setAgentConfigureIntegrationId] = useState<string | null>(null);

  useEffect(() => {
    const handler = (e: Event) => {
      const { integrationName, instanceId } = (e as CustomEvent).detail;
      if (instanceId) {
        // Existing instance — open configure modal
        setAgentConfigureIntegrationId(instanceId);
      } else if (integrationName) {
        // No instance — open create dialog
        setIntegrationDialogName(integrationName);
      }
    };
    window.addEventListener("agent:open-integration", handler);
    return () => window.removeEventListener("agent:open-integration", handler);
  }, []);

  const handleIntegrationCreated = useCallback(
    async (integrationId: string, instanceName: string) => {
      if (!canvas || !organizationId || !canvasId || !integrationDialogName) return;

      setJustConnectedIntegrations((prev) => new Set(prev).add(integrationDialogName));
      setTimeout(() => {
        setJustConnectedIntegrations((prev) => {
          const next = new Set(prev);
          next.delete(integrationDialogName);
          return next;
        });
      }, 2000);

      const integrationRef: ComponentsIntegrationRef = {
        id: integrationId,
        name: instanceName,
      };

      const updatedNodes = canvas.spec?.nodes?.map((node) => {
        const nodeIntegrationName = getNodeIntegrationName(node, availableIntegrations);
        if (nodeIntegrationName === integrationDialogName && !node.integration?.id) {
          return { ...node, integration: integrationRef };
        }
        return node;
      });

      const updatedWorkflow = {
        ...canvas,
        spec: { ...canvas.spec, nodes: updatedNodes },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      integrationDialogName,
      availableIntegrations,
      handleSaveWorkflow,
      isReadOnly,
      applyLocalWorkflowUpdate,
    ],
  );

  const handleNodeConfigurationSave = useCallback(
    async (
      nodeId: string,
      updatedConfiguration: Record<string, any>,
      updatedNodeName: string,
      integrationRef?: ComponentsIntegrationRef,
      concurrency?: ComponentsConcurrencySpec,
    ) => {
      if (!canvas || !organizationId || !canvasId) return;

      const configuringNode = canvasNodesById.get(nodeId);
      if (configuringNode) {
        const fieldCount = Object.values(updatedConfiguration).filter(
          (v) => v !== null && v !== undefined && v !== "",
        ).length;
        const { nodeType, integration } = getNodeAnalyticsProps(configuringNode, availableIntegrations);
        analytics.nodeConfigure(nodeType, integration, fieldCount, organizationId);
      }

      // Save snapshot before making changes

      // Update the node's configuration, name, and app installation ref in local cache only
      const updatedNodes = canvasNodes.map((node) => {
        if (node.id === nodeId) {
          // Handle widget nodes like any other node - store in configuration
          if (node.type === "TYPE_WIDGET") {
            return {
              ...node,
              name: updatedNodeName,
              configuration: { ...node.configuration, ...updatedConfiguration },
            };
          }

          return {
            ...node,
            configuration: updatedConfiguration,
            name: updatedNodeName,
            integration: integrationRef,
            concurrency,
          };
        }
        return node;
      });

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        // Bind the save to the version this closure's `canvas` represents.
        // The config panel can flush a pending auto-save on unmount (e.g. when
        // switching drafts remounts the canvas); without this, the save would
        // target whatever draft is now active and overwrite it with this
        // draft's graph.
        //
        // Do not await auto-save here: repository file commits round-trip through
        // canvas.yaml and would block expression autocomplete on every keystroke.
        void handleSaveWorkflow(updatedWorkflow, { showToast: false, savingVersionId: activeCanvasVersionId });
      }
    },
    [
      canvas,
      canvasNodes,
      canvasNodesById,
      organizationId,
      canvasId,
      activeCanvasVersionId,
      handleSaveWorkflow,
      isReadOnly,
      applyLocalWorkflowUpdate,
      availableIntegrations,
    ],
  );
  const debouncedAnnotationAutoSave = useMemo(
    () =>
      debounce(
        async () => {
          setIsAnnotationAutoSaveQueued(false);
          if (!organizationId || !canvasId) return;

          const annotationUpdates = new Map(pendingAnnotationUpdatesRef.current);
          if (annotationUpdates.size === 0) return;

          if (isReadOnly) {
            return;
          }

          const latestWorkflow = getCurrentWorkflowSnapshot();

          if (!latestWorkflow?.spec?.nodes) return;

          const updatedNodes = latestWorkflow.spec.nodes.map((node) => {
            if (!node.id || node.type !== "TYPE_WIDGET") {
              return node;
            }

            const updates = annotationUpdates.get(node.id);
            if (!updates) {
              return node;
            }

            return {
              ...node,
              configuration: {
                ...node.configuration,
                ...updates,
              },
            };
          });

          const updatedWorkflow = {
            ...latestWorkflow,
            spec: {
              ...latestWorkflow.spec,
              nodes: updatedNodes,
            },
          };
          const saveResult = await handleSaveWorkflow(updatedWorkflow, { showToast: false });
          if (saveResult?.status !== "saved") {
            return;
          }

          annotationUpdates.forEach((updates, nodeId) => {
            if (pendingAnnotationUpdatesRef.current.get(nodeId) === updates) {
              pendingAnnotationUpdatesRef.current.delete(nodeId);
            }
          });
        },
        isReadOnly ? 2000 : 100,
      ),
    [organizationId, canvasId, getCurrentWorkflowSnapshot, handleSaveWorkflow, isReadOnly],
  );

  const handleAnnotationBlur = useCallback(() => {
    if (isReadOnly) {
      return;
    }

    debouncedAnnotationAutoSave.flush();
  }, [isReadOnly, debouncedAnnotationAutoSave]);

  const clearPendingAutoSaveWork = useCallback(() => {
    debouncedAutoSave.cancel();
    debouncedAnnotationAutoSave.cancel();
    pendingPositionUpdatesRef.current.clear();
    pendingAnnotationUpdatesRef.current.clear();
    clearQueuedCanvasSave();
    clearQueuedAutoSaveFlags();
  }, [clearQueuedAutoSaveFlags, clearQueuedCanvasSave, debouncedAnnotationAutoSave, debouncedAutoSave]);

  const hasPendingCanvasSaveWork = useCallback(() => {
    return (
      pendingPositionUpdatesRef.current.size > 0 ||
      pendingAnnotationUpdatesRef.current.size > 0 ||
      !!queuedCanvasSaveRef.current ||
      isDrainingCanvasSaveQueueRef.current
    );
  }, []);

  const hasPendingLocalDraftChanges = useCallback(() => {
    if (!activeCanvasVersionIdRef.current) {
      return false;
    }

    const currentWorkflow = getCurrentWorkflowSnapshot();

    if (!currentWorkflow || !lastSavedWorkflowSignatureRef.current) {
      return false;
    }

    return getWorkflowSaveSignature(currentWorkflow) !== lastSavedWorkflowSignatureRef.current;
  }, [getCurrentWorkflowSnapshot]);

  const waitForLocalCanvasChangesToSettle = useCallback(async () => {
    debouncedAutoSave.flush();
    debouncedAnnotationAutoSave.flush();

    const deadline = Date.now() + VERSION_ACTION_SAVE_SETTLE_TIMEOUT_MS;
    while (Date.now() < deadline) {
      if (!hasPendingCanvasSaveWork()) {
        return !hasPendingLocalDraftChanges();
      }

      await new Promise((resolve) => {
        window.setTimeout(resolve, 50);
      });
    }

    return false;
  }, [debouncedAnnotationAutoSave, debouncedAutoSave, hasPendingCanvasSaveWork, hasPendingLocalDraftChanges]);

  const ensureVersionActionDraftReady = useCallback(
    async (blockedMessage: string) => {
      if (!hasPendingCanvasSaveWork() && !hasPendingLocalDraftChanges()) {
        return true;
      }

      const settled = await waitForLocalCanvasChangesToSettle();
      if (settled || !hasPendingLocalDraftChanges()) {
        return true;
      }

      const currentWorkflow = getCurrentWorkflowSnapshot();
      if (!currentWorkflow) {
        showErrorToast(blockedMessage);
        return false;
      }

      const saveResult = await handleSaveWorkflow(currentWorkflow, { showToast: false });
      if (saveResult?.status === "saved" && !saveResult.hasQueuedFollowUp && !hasPendingLocalDraftChanges()) {
        return true;
      }

      if (hasPendingCanvasSaveWork()) {
        const settledAfterSave = await waitForLocalCanvasChangesToSettle();
        if (settledAfterSave) {
          return true;
        }
      } else if (!hasPendingLocalDraftChanges()) {
        return true;
      }

      showErrorToast(blockedMessage);
      return false;
    },
    [
      getCurrentWorkflowSnapshot,
      handleSaveWorkflow,
      hasPendingCanvasSaveWork,
      hasPendingLocalDraftChanges,
      waitForLocalCanvasChangesToSettle,
    ],
  );

  const handleAnnotationUpdate = useCallback(
    (
      nodeId: string,
      updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
    ) => {
      if (!canvas || !organizationId || !canvasId) return;
      if (Object.keys(updates).length === 0) return;

      const latestWorkflow = getCurrentWorkflowSnapshot();
      if (!latestWorkflow) return;

      // Separate position updates from configuration updates
      const { x, y, ...configurationUpdates } = updates;
      const hasPositionUpdate = x !== undefined || y !== undefined;
      const hasConfigurationUpdate = Object.keys(configurationUpdates).length > 0;
      const updatedNodes = latestWorkflow?.spec?.nodes?.map((node) => {
        if (node.id !== nodeId || node.type !== "TYPE_WIDGET") {
          return node;
        }

        const updatedNode = { ...node };

        // Update position if provided
        if (hasPositionUpdate) {
          updatedNode.position = {
            x: x !== undefined ? x : node.position?.x || 0,
            y: y !== undefined ? y : node.position?.y || 0,
          };
        }

        // Update configuration if provided
        if (hasConfigurationUpdate) {
          updatedNode.configuration = {
            ...node.configuration,
            ...configurationUpdates,
          };
        }

        return updatedNode;
      });

      const updatedWorkflow = {
        ...latestWorkflow,
        spec: {
          ...latestWorkflow.spec,
          nodes: updatedNodes,
        },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (hasConfigurationUpdate && !isReadOnly) {
        const existing = pendingAnnotationUpdatesRef.current.get(nodeId) || {};
        pendingAnnotationUpdatesRef.current.set(nodeId, { ...existing, ...configurationUpdates });
        setIsAnnotationAutoSaveQueued(true);
        debouncedAnnotationAutoSave();
      }

      if (!isReadOnly && hasPositionUpdate) {
        // Queue position updates for auto-save
        pendingPositionUpdatesRef.current.set(nodeId, {
          x: x !== undefined ? x : latestWorkflow?.spec?.nodes?.find((n) => n.id === nodeId)?.position?.x || 0,
          y: y !== undefined ? y : latestWorkflow?.spec?.nodes?.find((n) => n.id === nodeId)?.position?.y || 0,
        });
        setIsPositionAutoSaveQueued(true);
        debouncedAutoSave();
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      getCurrentWorkflowSnapshot,
      debouncedAnnotationAutoSave,
      debouncedAutoSave,
      isReadOnly,
      applyLocalWorkflowUpdate,
    ],
  );

  const handleNodeAdd = useCallback(
    async (newNodeData: NewNodeData): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      const { buildingBlock, configuration, position, sourceConnection, integrationRef } = newNodeData;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, buildingBlock.configuration || []);
      const nameBase = newNodeData.nodeName || buildingBlock.name || "node";
      let newNodeId = "";

      await commitTopologyMutation((workflow) => {
        const existingNodeNames = (workflow.spec?.nodes || []).map((node) => node.name || "").filter(Boolean);
        const uniqueNodeName = generateUniqueNodeName(nameBase, existingNodeNames);
        newNodeId = generateNodeId(buildingBlock.name || "node", uniqueNodeName);

        const newNode: ComponentsNode = {
          id: newNodeId,
          name: uniqueNodeName,
          type:
            buildingBlock.type === "trigger"
              ? "TYPE_TRIGGER"
              : buildingBlock.name === "annotation"
                ? "TYPE_WIDGET"
                : "TYPE_ACTION",
          configuration: filteredConfiguration,
          integration: integrationRef,
          position: position
            ? {
                x: Math.round(position.x),
                y: Math.round(position.y),
              }
            : {
                x: (workflow.spec?.nodes?.length || 0) * 250,
                y: 100,
              },
        };

        if (buildingBlock.name === "annotation") {
          newNode.component = "annotation";
          newNode.configuration = { text: "", color: "yellow" };
        } else if (buildingBlock.type === "component" || buildingBlock.type === "trigger") {
          newNode.component = buildingBlock.name;
        }

        const { nodeType, integration, nodeRef } = getNodeAnalyticsProps(newNode, availableIntegrations);
        analytics.nodeAdd(nodeType, integration, nodeRef, organizationId);

        const newEdges: ComponentsEdge[] = sourceConnection
          ? [
              {
                sourceId: sourceConnection.nodeId,
                targetId: newNodeId,
                channel: sourceConnection.handleId || "default",
              },
            ]
          : [];
        return {
          workflow: appendWorkflowFragment(workflow, [newNode], newEdges),
          options: { addedNodeId: newNodeId },
        };
      });
      return newNodeId;
    },
    [canvas, organizationId, canvasId, availableIntegrations, commitTopologyMutation],
  );

  const handlePlaceholderAdd = useCallback(
    async (data: {
      position: { x: number; y: number };
      sourceNodeId?: string;
      sourceHandleId?: string | null;
    }): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      const latestWorkflow = getCurrentWorkflowSnapshot();
      if (!latestWorkflow) return "";

      const placeholderName = "New Component";
      const newNodeId = generateNodeId("component", "node");

      // Create placeholder node - will fail validation but still be saved
      const newNode: ComponentsNode = {
        id: newNodeId,
        name: placeholderName,
        type: "TYPE_ACTION",
        // NO component/trigger reference - causes validation error
        configuration: {},
        metadata: {},
        position: {
          x: Math.round(data.position.x),
          y: Math.round(data.position.y),
        },
      };

      const newEdge =
        data.sourceNodeId && data.sourceHandleId !== undefined
          ? ({
              sourceId: data.sourceNodeId,
              targetId: newNodeId,
              channel: data.sourceHandleId || "default",
            } as ComponentsEdge)
          : null;

      await commitTopologyMutation(
        (workflow) => appendWorkflowFragment(workflow, [newNode], newEdge ? [newEdge] : []),
        {
          addedNodeId: newNodeId,
        },
      );

      return newNodeId;
    },
    [canvas, organizationId, canvasId, getCurrentWorkflowSnapshot, commitTopologyMutation],
  );

  const handlePlaceholderConfigure = useCallback(
    async (data: {
      placeholderId: string;
      buildingBlock: BuildingBlock;
      nodeName: string;
      configuration: Record<string, unknown>;
      integrationName?: string;
    }): Promise<void> => {
      if (!canvas || !organizationId || !canvasId) {
        return;
      }

      const filteredConfiguration = filterVisibleConfiguration(
        data.configuration,
        data.buildingBlock.configuration || [],
      );
      await commitTopologyMutation((workflow) => {
        const nodes = workflow.spec?.nodes || [];
        const nodeIndex = nodes.findIndex((node) => node.id === data.placeholderId);
        if (nodeIndex === -1) {
          return workflow;
        }

        const existingNodeNames = nodes
          .filter((node) => node.id !== data.placeholderId)
          .map((node) => node.name || "")
          .filter(Boolean);
        const uniqueNodeName = generateUniqueNodeName(data.buildingBlock.name || "node", existingNodeNames);
        const updatedNode: ComponentsNode = {
          ...nodes[nodeIndex],
          name: uniqueNodeName,
          type: data.buildingBlock.type === "trigger" ? "TYPE_TRIGGER" : "TYPE_ACTION",
          component: data.buildingBlock.name,
          configuration: filteredConfiguration,
        };
        const updatedNodes = [...nodes];
        updatedNodes[nodeIndex] = updatedNode;

        const declaredChannels = (data.buildingBlock.outputChannels || [])
          .map((channel) => channel.name)
          .filter((channel): channel is string => Boolean(channel));
        const validChannels = declaredChannels.length > 0 ? declaredChannels : ["default"];
        const updatedEdges = (workflow.spec?.edges || []).map((edge) => {
          if (edge.sourceId !== data.placeholderId || validChannels.includes(edge.channel || "default")) {
            return edge;
          }
          return { ...edge, channel: validChannels[0] };
        });

        return {
          ...workflow,
          spec: { ...workflow.spec, nodes: updatedNodes, edges: updatedEdges },
        };
      });
    },
    [canvas, organizationId, canvasId, commitTopologyMutation],
  );

  const handleEdgeCreate = useCallback(
    async (sourceId: string, targetId: string, sourceHandle?: string | null) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes

      // Create the new edge
      const newEdge: ComponentsEdge = {
        sourceId,
        targetId,
        channel: sourceHandle || "default",
      };

      analytics.edgeCreate(organizationId);
      await commitTopologyMutation((workflow) => appendWorkflowFragment(workflow, [], [newEdge]));
    },
    [canvas, organizationId, canvasId, commitTopologyMutation],
  );
  const handleNodeDelete = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes

      const specNodes = canvas.spec?.nodes || [];
      const nodeBeingDeleted = specNodes.find((n) => n.id === nodeId);
      if (nodeBeingDeleted) {
        const { nodeType, integration, nodeRef } = getNodeAnalyticsProps(nodeBeingDeleted, availableIntegrations);
        analytics.nodeRemove(nodeType, integration, nodeRef, organizationId);
      }

      await commitTopologyMutation((workflow) => removeWorkflowNodes(workflow, new Set([nodeId])));
    },
    [canvas, organizationId, canvasId, availableIntegrations, commitTopologyMutation],
  );
  const handleNodesDelete = useCallback(
    async (nodeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      const nodeIdSet = new Set(nodeIds);
      const specNodes = canvas.spec?.nodes || [];
      specNodes
        .filter((n) => nodeIds.includes(n.id || ""))
        .forEach((node) => {
          const { nodeType, integration, nodeRef } = getNodeAnalyticsProps(node, availableIntegrations);
          analytics.nodeRemove(nodeType, integration, nodeRef, organizationId);
        });
      await commitTopologyMutation((workflow) => removeWorkflowNodes(workflow, nodeIdSet));
    },
    [canvas, organizationId, canvasId, availableIntegrations, commitTopologyMutation],
  );
  const handleAutoLayoutNodes = useCallback(
    async (nodeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      const updatedWorkflow = await DefaultLayoutEngine.apply(canvas, {
        nodeIds,
        scope: "connected-component",
        components,
        direction: resolveCanvasFlowDirection(canvas.metadata?.factoryId),
      });

      analytics.autoLayout(updatedWorkflow.spec?.nodes?.length ?? 0, organizationId);

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, components, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
  );

  const handleNodesDuplicate = useCallback(
    async (nodeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      await commitTopologyMutation((workflow) => {
        const { newNodes, nodeIdMap } = buildDuplicatedNodes(workflow.spec?.nodes || [], nodeIds);
        if (newNodes.length === 0) return { workflow, changed: false };

        const duplicatedNodeIds = new Set(nodeIds);
        const newEdges = buildDuplicatedEdges(workflow.spec?.edges || [], duplicatedNodeIds, nodeIdMap);
        return appendWorkflowFragment(workflow, newNodes, newEdges);
      });
    },
    [canvas, organizationId, canvasId, commitTopologyMutation],
  );

  const handleEdgeDelete = useCallback(
    async (edgeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes

      // Parse edge IDs to extract sourceId, targetId, and channel
      // Edge IDs are formatted as: `${sourceId}--${targetId}--${channel}`
      const edgesToRemove = edgeIds.map((edgeId) => {
        let parts = edgeId?.split("-targets->") || [];
        parts = parts.flatMap((part) => part.split("-using->"));
        return {
          sourceId: parts[0],
          targetId: parts[1],
          channel: parts[2],
        };
      });

      analytics.edgeRemove(organizationId);
      await commitTopologyMutation((workflow) => removeWorkflowEdges(workflow, edgesToRemove));
    },
    [canvas, organizationId, canvasId, commitTopologyMutation],
  );

  /**
   * Updates the position of a node in the local cache.
   * Called when a node is dragged in the CanvasPage.
   *
   * @param nodeId - The ID of the node to update.
   * @param position - The new position of the node.
   */
  const handleNodePositionChange = useCallback(
    (nodeId: string, position: { x: number; y: number }) => {
      if (!organizationId || !canvasId) return;

      const roundedPosition = {
        x: Math.round(position.x),
        y: Math.round(position.y),
      };

      queuePositionAutoSave(new Map([[nodeId, roundedPosition]]));
    },
    [organizationId, canvasId, queuePositionAutoSave],
  );

  const handleNodesPositionChange = useCallback(
    (updates: Array<{ nodeId: string; position: { x: number; y: number } }>) => {
      if (!organizationId || !canvasId || updates.length === 0) return;

      // Create a map of nodeId -> rounded position for efficient lookup
      const positionMap = new Map(
        updates.map((update) => [
          update.nodeId,
          {
            x: Math.round(update.position.x),
            y: Math.round(update.position.y),
          },
        ]),
      );

      queuePositionAutoSave(positionMap);
    },
    [organizationId, canvasId, queuePositionAutoSave],
  );

  const handleNodeCollapseChange = useCallback(
    async (nodeId: string, collapsed: boolean) => {
      if (!canvas || !organizationId || !canvasId) return;

      const currentNode = canvas.spec?.nodes?.find((node) => node.id === nodeId);
      if (!currentNode) return;

      if (currentNode.isCollapsed === collapsed) {
        return;
      }

      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              isCollapsed: collapsed,
            }
          : node,
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
  );

  const handleReEmit = useCallback(
    async (nodeId: string, eventOrExecutionId: string) => {
      if (!canvasId) return;

      try {
        await canvasesReemitTriggerEvent(
          withOrganizationHeader({
            path: {
              canvasId,
              nodeId,
              eventId: eventOrExecutionId,
            },
          }),
        );

        const node = canvasNodesById.get(nodeId);
        if (node && organizationId) {
          const { nodeType, integration } = getNodeAnalyticsProps(node, availableIntegrations);
          analytics.eventEmit(nodeType, integration, organizationId);
        }
      } catch (error) {
        showErrorToast("Failed to re-emit event");
        throw error;
      }
    },
    [canvasId, canvasNodesById, availableIntegrations, organizationId],
  );

  const handleNodeDuplicate = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      await commitTopologyMutation((workflow) => {
        const nodeToDuplicate = workflow.spec?.nodes?.find((node) => node.id === nodeId);
        if (!nodeToDuplicate) return { workflow, changed: false };

        const existingNodeNames = (workflow.spec?.nodes || []).map((node) => node.name || "").filter(Boolean);
        let baseName = nodeToDuplicate.name?.trim() || "";
        if (!baseName) {
          baseName =
            (nodeToDuplicate.type === "TYPE_TRIGGER" || nodeToDuplicate.type === "TYPE_ACTION") &&
            nodeToDuplicate.component
              ? nodeToDuplicate.component
              : "node";
        }

        const uniqueNodeName = generateUniqueNodeName(baseName, existingNodeNames);
        const newNodeId = generateNodeId(baseName, uniqueNodeName);
        const duplicateNode: ComponentsNode = {
          ...nodeToDuplicate,
          id: newNodeId,
          name: uniqueNodeName,
          position: {
            x: (nodeToDuplicate.position?.x || 0) + 50,
            y: (nodeToDuplicate.position?.y || 0) + 50,
          },
          isCollapsed: false,
        };

        return {
          workflow: appendWorkflowFragment(workflow, [duplicateNode]),
          options: { addedNodeId: newNodeId },
        };
      });
    },
    [canvas, organizationId, canvasId, commitTopologyMutation],
  );

  const cancelPendingCanvasSaves = useCallback(() => {
    canvasSaveSessionRef.current += 1;
    clearQueuedCanvasSave();
  }, [clearQueuedCanvasSave]);

  const handleCanvasDraftRestoredToCommitted = useCallback(
    (_version: CanvasesCanvasVersion) => {
      const restoredWorkflow = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId!, canvasId!));
      if (restoredWorkflow) {
        setLastSavedWorkflowSnapshot(restoredWorkflow);
      }
    },
    [canvasId, organizationId, queryClient, setLastSavedWorkflowSnapshot],
  );

  const handleCommittedVersionId = useCallback(
    (_versionId: string) => {
      flushSync(() => {
        setEditSessionActive(false);
        setActiveCanvasVersion(null);
        setDraftCanvasSpec(null);
        draftCanvasSpecsRef.current.clear();

        setSearchParams((current) => {
          const next = new URLSearchParams(current);
          next.delete("version");
          next.delete("branch");
          return clearComponentSidebarSearchParams(next);
        });
      });

      editSessionActiveRef.current = false;
      previewingCurrentVersionRef.current = false;
      activeCanvasVersionIdRef.current = "";
    },
    [setSearchParams],
  );

  const { handleCommitStaging, handleResetStaging, commitStagingPending, resetStagingPending } = useDraftStagingActions(
    {
      organizationId,
      canvasId,
      activeCanvasVersionId,
      hasEditableVersion,
      ensureVersionActionDraftReady,
      commitCanvasStagingMutation,
      discardCanvasStagingMutation,
      draftCanvasSpecsRef,
      setDraftCanvasSpec,
      setActiveCanvasVersion,
      setStagingResetNonce,
      consoleMutationGenerationRef,
      setIsPreparingVersionAction,
      flushRepositoryFileStaging,
      cancelPendingCanvasSaves,
      onCanvasDraftRestoredToCommitted: handleCanvasDraftRestoredToCommitted,
      onCommittedVersionId: handleCommittedVersionId,
      registerIgnoredCanvasUpdatedEcho,
    },
  );

  const handleOpenCommitDialog = useCallback(() => {
    setCommitDialogOpen(true);
  }, []);

  const handleConfirmCommitStaging = useCallback(
    async (commitMessage: string) => {
      await handleCommitStaging(commitMessage);
      setCommitDialogOpen(false);
      if (factoryConfigure) {
        onFactoryConfigureDone?.();
      }
    },
    [factoryConfigure, handleCommitStaging, onFactoryConfigureDone],
  );

  const handleAgentSidebarStagingCommit = useCallback(
    async (commitMessage: string) => {
      if (!organizationId || !canvasId) {
        return false;
      }

      const versionId = activeCanvasVersionId || liveCanvasVersionId || "";
      if (!versionId) {
        return false;
      }

      const trimmedMessage = commitMessage.trim();
      if (!trimmedMessage) {
        return false;
      }

      try {
        return await executeCommitStaging({
          organizationId,
          canvasId,
          activeCanvasVersionId: versionId,
          commitMessage: trimmedMessage,
          queryClient,
          commitCanvasStagingMutation,
          consoleMutationGenerationRef,
          draftCanvasSpecsRef,
          setDraftCanvasSpec,
          setStagingResetNonce,
          ensureVersionActionDraftReady,
          flushRepositoryFileStaging,
          registerIgnoredCanvasUpdatedEcho,
          onCommittedVersionId: handleCommittedVersionId,
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to commit changes"));
        return false;
      }
    },
    [
      organizationId,
      canvasId,
      activeCanvasVersionId,
      liveCanvasVersionId,
      queryClient,
      commitCanvasStagingMutation,
      ensureVersionActionDraftReady,
      flushRepositoryFileStaging,
      handleCommittedVersionId,
      registerIgnoredCanvasUpdatedEcho,
    ],
  );

  const handleDiscardStaleStaging = useCallback(async () => {
    await handleResetStaging();
  }, [handleResetStaging]);

  const activateCanvasVersionForEditing = useActivateCanvasVersionForEditing({
    organizationId,
    canvasId,
    liveCanvasVersionId,
    queryClient,
    draftCanvasSpec,
    draftCanvasSpecsRef,
    activeCanvasVersionIdRef,
    lastAppliedVersionSnapshotRef,
    liveCanvasVersion,
    liveCanvas,
    clearPendingAutoSaveWork,
    setDraftCanvasSpec,
    setActiveCanvasVersion,
    setLastSavedWorkflowSnapshot,
    setSearchParams,
    initializeFromWorkflow,
  });

  const handleUseVersion = useCallback(
    (versionID: string, options?: { preserveStagedLayer?: boolean }) => {
      if (!organizationId || !canvasId) {
        return;
      }

      const version = selectableVersionsById.get(versionID);
      if (!version) {
        showErrorToast("Version not found");
        return;
      }

      const versionShell = canvasVersionShell(version);
      if (!versionShell) {
        showErrorToast("Version not found");
        return;
      }

      activateCanvasVersionForEditing(versionID, versionShell, options);
    },
    [activateCanvasVersionForEditing, canvasId, organizationId, selectableVersionsById],
  );

  const resyncLiveVersionDraftAfterSwitch = useCallback(
    async (versionId: string, options?: { preserveStagedLayer?: boolean }) => {
      handleUseVersion(versionId, options);
      await resyncStagedEditorState(versionId, { bumpResetNonce: false });
    },
    [handleUseVersion, resyncStagedEditorState],
  );

  const handleSeeCurrentVersion = useCallback(() => {
    if (!liveCanvasVersionId) {
      showErrorToast("No live version available");
      return;
    }
    // Deliberate preview of the current version keeps the edit session open.
    previewingCurrentVersionRef.current = true;
    if (editSessionActive) {
      void resyncLiveVersionDraftAfterSwitch(liveCanvasVersionId);
      return;
    }
    handleUseVersion(liveCanvasVersionId);
  }, [editSessionActive, liveCanvasVersionId, handleUseVersion, resyncLiveVersionDraftAfterSwitch]);

  const handleUseVersionFromVersionPanel = useCallback(
    (versionID: string) => {
      if (hasEditableVersion && hasLocalSaveActivity && versionID !== activeCanvasVersionIdRef.current) {
        const shouldSwitch = window.confirm(
          "You have unsaved changes in the current draft. Switch versions and discard those unsaved changes?",
        );
        if (!shouldSwitch) {
          return;
        }
      }

      const isSelectingLiveVersion = isActiveCanvasVersionCurrentLive({
        activeCanvasVersionId: versionID,
        liveCanvasVersionId,
      });

      // Track when the user deliberately selects the current/live version from
      // the sidebar so the edit session stays open (vs. internal navigation back
      // to live after publish/discard, which must close it).
      previewingCurrentVersionRef.current = isSelectingLiveVersion;

      if (editSessionActive && isSelectingLiveVersion) {
        void resyncLiveVersionDraftAfterSwitch(versionID);
        return;
      }

      handleUseVersion(versionID);
    },
    [
      handleUseVersion,
      resyncLiveVersionDraftAfterSwitch,
      hasEditableVersion,
      hasLocalSaveActivity,
      editSessionActive,
      liveCanvasVersionId,
    ],
  );

  const runInspectionChromeActive = isRunInspectionMode && !editSessionActive && !isEnteringEditSession;
  useAgentNodeFocusRequest(setFocusRequest, runInspectionChromeActive);
  const runParticipantFit = useRunParticipantFitRequest({
    isRunInspectionMode: runInspectionChromeActive,
    selectedRunId,
    runCanvasLoading,
    runCanvasData,
  });
  const requestRunFitRef = useRef(runParticipantFit.requestParticipantFit);
  requestRunFitRef.current = runParticipantFit.requestParticipantFit;
  const { headerMode, canvasStateMode, showBottomStatusControls, hideAddControls, readOnlyViewModes } =
    getWorkflowViewPresentation({
      ...urlViewFlags,
      isRunInspectionMode: runInspectionChromeActive,
      hasEditableVersion: hasEditableVersion || isEnteringEditSession,
      isViewingCurrentLiveVersion,
    });

  const { enterLiveEditSession } = useEnterLiveEditSession({
    organizationId,
    canvasId,
    canUpdateCanvas: canStageCanvasVersion,
    liveCanvasVersionId,
    selectableVersionsById,
    handleUseVersion,
    resyncStagedEditorState,
    previewingCurrentVersionRef,
    setEditSessionActive,
    setIsEnteringEditSession,
  });

  handleRemoteStagingUpdatedRef.current = async () => {
    const targetVersionId = liveCanvasVersionId;
    if (!targetVersionId || !canvasId) {
      return;
    }

    if (editSessionActiveRef.current && activeCanvasVersionIdRef.current === targetVersionId) {
      await resyncStagedEditorState(targetVersionId, { bumpResetNonce: false });
      return;
    }

    const stagingSummary = await queryClient.fetchQuery({
      queryKey: canvasKeys.canvasStaging(canvasId),
      queryFn: async () => {
        const summary = await fetchCanvasStagingSummary(canvasId);
        return summary ?? { hasStaging: false, stagedPaths: [] };
      },
      staleTime: 0,
    });
    if (!stagingSummary.hasStaging) {
      return;
    }

    await enterLiveEditSession();
  };

  const handleAgentStagingReady = useCallback(async (): Promise<boolean> => {
    if (!liveCanvasVersionId) {
      return false;
    }

    if (editSessionActive && isViewingCurrentLiveVersion) {
      await resyncStagedEditorState(liveCanvasVersionId, { bumpResetNonce: false });
      return true;
    }

    return enterLiveEditSession();
  }, [
    editSessionActive,
    liveCanvasVersionId,
    enterLiveEditSession,
    isViewingCurrentLiveVersion,
    resyncStagedEditorState,
  ]);

  const handleToggleEditMode = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    if (!canStageCanvasVersion) {
      showErrorToast("You don't have permission to edit this canvas");
      return;
    }

    if (editSessionActive) {
      if (organizationId && canvasId) {
        resetCommittedLiveCanvasDetail({
          queryClient,
          organizationId,
          canvasId,
          liveCanvasVersion,
        });
      }
      clearLiveEditSessionDraftState({
        setEditSessionActive,
        setActiveCanvasVersion,
        setDraftCanvasSpec,
        draftCanvasSpecsRef,
        activeCanvasVersionIdRef,
        previewingCurrentVersionRef,
      });
      setIsEnteringEditSession(false);
      setSearchParams(clearLiveEditSessionSearchParams);
      void refreshLatestLiveCanvasData();
      return;
    }

    if (!liveCanvasVersionId || !liveCanvasVersion) {
      if (canvasLoading || liveCanvasVersionLoading) {
        return;
      }
      showErrorToast("No live version available");
      return;
    }

    if (searchParams.get("run")) {
      setRunDetailNodeId(null);
      setFocusRequest(null);
      setSearchParams((current) => clearRunInspectionSearchParams(current), { replace: true });
      await Promise.resolve();
    }

    await enterLiveEditSession();
  }, [
    organizationId,
    canvasId,
    canStageCanvasVersion,
    editSessionActive,
    liveCanvasVersionId,
    liveCanvasVersion,
    canvasLoading,
    liveCanvasVersionLoading,
    enterLiveEditSession,
    refreshLatestLiveCanvasData,
    setSearchParams,
    queryClient,
    searchParams,
    setRunDetailNodeId,
    setFocusRequest,
  ]);

  const {
    runLookupEnabled,
    liveSidebarRunLookupEnabled,
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
    handleSelectRun,
    handleSelectRunFromSidebarEvent,
    handleLogRunExecutionSelect,
    handleNavigateRun,
    handleClearRunInspection,
    handleSelectLiveCanvas,
    handleRunNodeDetailSelection,
    handleRunNodeDetailNavigate,
    handleRunCanvasNodeClick,
    handleLiveCanvasNodeClick,
  } = useRunInspectionNavigation({
    hasEditableVersion,
    liveCanvasVersionId,
    handleUseVersion,
    isViewingLiveVersion,
    isEditing,
    isRunInspectionMode,
    clearDismissedRunDetail,
    setRunDetailNodeId,
    setFocusRequest,
    setSearchParams,
    requestRunFitRef,
    preserveRunDetailNodeOnNextRunChangeRef,
    searchParams,
    runDetailNodeId,
    clearParticipantFit: runParticipantFit.clearParticipantFit,
    selectedRunId,
    selectedRun,
    isSelectedRunLoading,
    describeRunSettled,
    canvasId,
    organizationId,
    queryClient,
    runs: runsData.runs,
    infiniteRunsPages: infiniteRunsQuery.data?.pages,
    runCanvasData,
    canvasNodesById,
    refetchNodeDataMethod,
  });

  const { handleSelectConsoleMode, handleExitConsoleMode } = useConsoleModeActions({
    setIsConsoleAddPanelOpen,
    setIsConsoleYamlOpen,
    setSearchParams,
  });

  const { handleSelectMemoryMode, handleExitMemoryMode } = useMemoryModeActions({
    setIsConsoleAddPanelOpen,
    setIsConsoleYamlOpen,
    setSearchParams,
  });

  const { handleSelectFilesMode, handleExitFilesMode } = useFilesModeActions({
    setIsConsoleAddPanelOpen,
    setIsConsoleYamlOpen,
    setSearchParams,
  });

  const {
    handleSelectCanvasView,
    handleConsoleAddPanelDialogOpenChange,
    onConsoleAddPanel,
    onConsoleOpenYaml,
    consoleYamlReadOnly,
  } = useWorkflowViewModeActions({
    ...urlViewFlags,
    hasEditableVersion,
    canUpdateCanvas: canStageCanvasVersion,
    canvasDeletedRemotely,
    handleExitConsoleMode,
    handleExitMemoryMode,
    handleExitFilesMode,
    handleClearRunInspection,
    handleToggleEditMode,
    setIsConsoleAddPanelOpen,
    setIsConsoleYamlOpen,
  });

  const { handleEnterEditModeFromHeader, clearRunInspectionForEdit } = useWorkflowHeaderEditActions({
    isRunInspectionMode,
    handleClearRunInspection,
    handleToggleEditMode,
    setRunDetailNodeId,
    setSearchParams,
    // Factory view embed is read-only. Configure enters via a dedicated effect
    // (does not await staged-spec resync — that was leaving "Loading canvas..." forever).
    startup:
      factoryViewOnly || factoryConfigure
        ? undefined
        : {
            hasEditableVersion,
            canUpdateCanvas: canStageCanvasVersion,
            canvas,
            liveVersionLoading: canvasLoading || liveCanvasVersionLoading,
            handlePlaceholderAdd,
            searchParams,
          },
  });

  // Ends the edit session: closes the versions sidebar and returns to the live
  // canvas (which restores events/runs). Works whether editing a draft or
  // previewing a version from the sidebar.
  const handleExitEditSession = useCallback(() => {
    setEditSessionActive(false);
    clearRunInspectionForEdit();
    if (liveCanvasVersionId) {
      handleUseVersion(liveCanvasVersionId);
    }
  }, [clearRunInspectionForEdit, liveCanvasVersionId, handleUseVersion]);

  useFactoryConfigureSession({
    factoryConfigure,
    factoryConfigureActionsRef,
    onFactoryConfigureBusyChange,
    onFactoryConfigureDone,
    editSessionActive,
    setEditSessionActive,
    canStageCanvasVersion,
    canvasLoading,
    liveCanvasVersionLoading,
    liveCanvasVersionId,
    liveCanvasVersion,
    liveCanvas,
    previewingCurrentVersionRef,
    activateCanvasVersionForEditing,
    draftCanvasSpecsRef,
    setDraftCanvasSpec,
    resyncStagedEditorState,
    setLastSavedWorkflowSnapshot,
    commitStagingPending,
    resetStagingPending,
    activeCanvasVersionIdRef,
    activeCanvasVersionId,
    getCurrentWorkflowSnapshot,
    updateCanvasVersionMutation,
    handleCommitStaging,
    handleResetStaging,
    handleExitEditSession,
    hasStagingChanges,
    hasUncommittedCanvasDraftChanges,
  });

  const buildYamlExportPayload = useCallback(
    (workflow: CanvasesCanvas | null | undefined, canvasNodes?: CanvasNode[]) =>
      buildCanvasYamlExportPayload(workflow, canvasNodes),
    [],
  );

  const { onCancelExecution } = useCancelExecutionHandler({
    canvasId: canvasId!,
    canvas,
  });

  const [isResolvingErrors, setIsResolvingErrors] = useState(false);

  const handleAcknowledgeErrors = useCallback(
    async (executionIds: string[]) => {
      if (!canvasId || executionIds.length === 0 || isResolvingErrors) {
        return;
      }

      setIsResolvingErrors(true);
      try {
        await resolveExecutionErrors(canvasId, executionIds);
        await queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteRuns(canvasId) });
        await queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecutions() });
        showSuccessToast("Errors acknowledged");
      } catch {
        showErrorToast("Failed to acknowledge errors");
      } finally {
        setIsResolvingErrors(false);
      }
    },
    [canvasId, isResolvingErrors, queryClient],
  );

  // Provide state function based on component type
  const getExecutionState = useCallback(
    (nodeId: string, execution: CanvasesCanvasNodeExecution): { map: EventStateMap; state: EventState } => {
      const node = canvasNodesById.get(nodeId);
      if (!node) {
        return {
          map: getStateMap("default"),
          state: getState("default")(buildExecutionInfo(execution)),
        };
      }

      let componentName = "default";
      if (node.type === "TYPE_ACTION" && node.component) {
        componentName = node.component;
      } else if (node.type === "TYPE_TRIGGER" && node.component) {
        componentName = node.component;
      }

      return {
        map: getStateMap(componentName),
        state: getState(componentName)(buildExecutionInfo(execution)),
      };
    },
    [canvasNodesById],
  );

  const getCustomField = useCallback(
    (nodeId: string, integration?: OrganizationsIntegration) => {
      const node = canvasNodesById.get(nodeId);
      if (!node) return null;

      let componentName = "";
      if (node.type === "TYPE_TRIGGER" && node.component) {
        componentName = node.component;
      } else if (node.type === "TYPE_ACTION" && node.component) {
        componentName = node.component;
      }

      const renderer = getCustomFieldRenderer(componentName);
      if (!renderer) return null;

      const context: {
        integration?: OrganizationsIntegration;
      } = {};
      if (integration) {
        context.integration = integration;
      }

      // Return a function that takes the current configuration
      return (configuration?: Record<string, unknown>) => {
        return renderCanvasNodeCustomField({
          renderer,
          node,
          configuration,
          context: Object.keys(context).length > 0 ? context : undefined,
        });
      };
    },
    [canvasNodesById],
  );

  const appFiles = useMemo(
    () =>
      buildAppFiles({
        canvas,
        canvasNodes: nodes,
        panels: consoleQuery.data?.panels,
        layout: consoleQuery.data?.layout,
        canvasId,
        canvasName: canvas?.metadata?.name,
        consoleLoading: consoleQuery.isLoading,
        consoleError: consoleQuery.error,
      }),
    [
      canvas,
      nodes,
      consoleQuery.data?.panels,
      consoleQuery.data?.layout,
      canvasId,
      consoleQuery.isLoading,
      consoleQuery.error,
    ],
  );
  const { onSpecFileChange } = useSpecFileAutosave({
    canvas,
    isReadOnly,
    applyLocalWorkflowUpdate,
    handleSaveWorkflow,
    updateConsoleMutation,
    onEffectiveConsoleChange: handleEffectiveConsoleChange,
  });
  const { onShowDiff, onShowNodeDiff, yamlDiffModal } = useCanvasYamlDiffModal({
    hasUnpublishedDraftChanges: hasStagingChanges,
    liveCanvas,
    liveCanvasVersion,
    draftCanvasVersion: liveCanvasVersion,
    draftCanvas: canvas,
    draftNodes: nodes,
    activeCanvasVersionId,
    buildYamlExportPayload,
  });

  const isInitialCanvasBootstrapLoading =
    !canvas && (canvasLoading || triggersLoading || componentsLoading || widgetsLoading);

  useReportPageReady(!isInitialCanvasBootstrapLoading && !!canvas, {
    failed: !canvas && !canvasLoading && !showDraftCanvasLoadingOverlay,
  });

  // Keep full-screen loading only for initial bootstrap.
  // Version switches should not unmount the page.
  if (isInitialCanvasBootstrapLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
          <p className="text-sm text-gray-500">Loading canvas...</p>
        </div>
      </div>
    );
  }

  if (!canvas && !canvasLoading && !showDraftCanvasLoadingOverlay) {
    // Workflow not found after loading - could be deleted or doesn't exist
    // Show a brief message then redirect (handled by the error useEffect above)
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-4">
          <h1 className="text-4xl font-bold text-gray-700">404</h1>
          <p className="text-sm text-gray-500">Canvas not found</p>
          <p className="text-sm text-gray-400">
            This canvas may have been deleted or you may not have permission to view it.
          </p>
        </div>
      </div>
    );
  }

  const handleReloadRemoteCanvas = async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    clearPendingAutoSaveWork();
    setRemoteCanvasUpdatePending(false);
    setLastSavedWorkflowSnapshot(null);

    if (isViewingLiveVersion) {
      await queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      await queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      return;
    }
  };

  const backToAppId = searchParams.get("appId") ?? undefined;
  const appBanner = backToAppId ? (
    <div className="bg-blue-50 border-b border-blue-200 px-4 py-2 flex items-center gap-2">
      <Link
        to={appPath(organizationId!, backToAppId)}
        className="flex items-center gap-1 text-sm text-blue-700 hover:text-blue-900 transition-colors"
      >
        <ArrowLeft size={14} />
        <span>Back to App</span>
      </Link>
    </div>
  ) : null;

  const remoteUpdateBanner =
    remoteCanvasUpdatePending && !hasLocalSaveActivity ? (
      <div className="bg-amber-100 px-4 py-2.5 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-medium text-gray-900">Canvas updated elsewhere</p>
          <p className="text-[13px] text-black/60">
            A newer canvas version is available. Reloading will discard your unsaved local changes.
          </p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={handleReloadRemoteCanvas}>
            Reload remote
          </Button>
        </div>
      </div>
    ) : null;
  const headerBanners = [appBanner, remoteUpdateBanner].filter(Boolean);
  const headerBanner = headerBanners.length > 0 ? <div className="flex flex-col">{headerBanners}</div> : null;
  const enterEditModeDisabled = !canStageCanvasVersion;
  const enterEditModeDisabledTooltip = canvasVersionEditPermission.tooltip;
  const exitEditModeDisabled = !canStageCanvasVersion;
  const exitEditModeDisabledTooltip = getExitEditModeDisabledTooltip({
    canUpdateCanvas: canStageCanvasVersion,
    canvasDeletedRemotely,
    hasEditableVersion,
  });
  const { disabled: runDisabled, tooltip: runDisabledTooltip } = getRunActionState({
    ...canvasAccess,
    isViewingDraftVersion: isEditing,
    isViewingCurrentLiveVersion,
  });
  runDisabledRef.current = runDisabled;
  runDisabledTooltipRef.current = runDisabledTooltip;

  const selectedRunDetailDismissed = isRunDetailDismissed(detailDismissedForRunId, selectedRunId);
  const { showRunsSidebar, showVersionsSidebar } = resolveFactoryEmbedSidebars({
    factoryEmbed,
    headerModeAllowsRuns: allowsRunsSidebar(headerMode),
    editSessionActive,
    isMemoryMode: urlViewFlags.isMemoryMode,
    isFilesMode: urlViewFlags.isFilesMode,
    runInspectionChromeActive,
  });
  const factoryEmbedCanvasChrome = resolveFactoryEmbedCanvasChrome({
    factoryEmbed,
    factoryViewOnly,
    factoryOwnedApp,
    routeFactoryId,
    canvas,
    headerBanner,
    canUpdateCanvas,
    showBottomStatusControls,
    hideAddControls,
    editSessionActive,
    runInspectionChromeActive,
    handleSelectMemoryMode,
    handleSelectConsoleMode,
    handleSelectFilesMode,
    handleEnterEditModeFromHeader,
    handleExitEditSession,
    handleSelectLiveCanvas,
    handleBackToRunList,
    filesHeaderActionsSlotId,
    runsHasFitToViewRef,
    hasFitToViewRef,
    runsViewportRef,
    viewportRef,
    selectedRunId,
    selectedRunDetailDismissed,
    runCanvasLoading,
    selectedRun,
  });

  const toolSidebarRunsContent = renderCanvasRunsSidebarPanel({
    isOpen: showRunsSidebar,
    canvasId: canvasId!,
    runs: runsData.runs,
    selectedRunId,
    selectedRun,
    isSelectedRunLoading,
    onSelectRun: handleSelectRun,
    onSelectLiveCanvas: handleSelectLiveCanvas,
    onBackToRunList: handleBackToRunList,
    initialOpenDetail: openRunDetailOnMount,
    detailDismissedForRunId,
    selectedNodeId: runDetailNodeId,
    onSelectNode: handleRunNodeDetailNavigate,
    hasNextPage: !!infiniteRunsQuery.hasNextPage,
    isFetchingNextPage: infiniteRunsQuery.isFetchingNextPage,
    onLoadMore: () => infiniteRunsQuery.fetchNextPage(),
    isLoading: infiniteRunsQuery.isLoading,
    isError: infiniteRunsQuery.isError,
    onRetry: () => infiniteRunsQuery.refetch(),
    componentIconMap,
    filterState: runFilterState,
  });
  const toolSidebarVersionsContent = renderCanvasVersionsSidebarPanel({
    isOpen: showVersionsSidebar,
    scrollPersistenceKey: canvasId,
    liveCanvasVersionId: liveCanvasVersionId,
    liveCanvasVersion,
    selectedCanvasVersion,
    liveVersions,
    canEditCanvasVersion: canStageCanvasVersion,
    canvasDeletedRemotely,
    onUseVersion: handleUseVersionFromVersionPanel,
    onLoadMoreLiveVersions: hasMoreLiveVersions ? () => canvasLiveVersionsQuery.fetchNextPage() : undefined,
    loadMoreLiveVersionsDisabled: !hasMoreLiveVersions || isLoadingMoreLiveVersions,
    loadMoreLiveVersionsPending: isLoadingMoreLiveVersions,
  });
  const factoryLayoutCanvasProps = resolveFactoryAutoCanvasProps(factoryAutoLayout, isReadOnly, {
    onAutoLayoutNodes: handleAutoLayoutNodes,
    onNodePositionChange: handleNodePositionChange,
    onNodesPositionChange: handleNodesPositionChange,
    onPlaceholderAdd: handlePlaceholderAdd,
    onPlaceholderConfigure: handlePlaceholderConfigure,
  });

  return (
    <>
      <div className="relative h-full w-full">
        <WorkflowPageModeOverlays
          key={`overlays:${stableCanvasViewKey}:reset-${stagingResetNonce}`}
          urlViewFlags={urlViewFlags}
          console={{
            canActOnCanvas,
            editSessionUiReady: isEditSessionUiReady,
            hasUncommittedCanvasDraftChanges,
            editLocked: isReadOnly,
            showConsoleEditControls: isEditing,
            onConsoleAddPanel,
            onConsoleOpenYaml,
            consoleYamlReadOnly,
            consoleQuery,
            updateConsoleMutation,
            addPanelDialogOpen: isConsoleAddPanelOpen,
            onAddPanelDialogOpenChange: handleConsoleAddPanelDialogOpenChange,
            yamlModalOpen: isConsoleYamlOpen,
            onYamlModalOpenChange: setIsConsoleYamlOpen,
            canvasId: canvasId || undefined,
            canvasName: canvas?.metadata?.name || undefined,
            organizationId: organizationId || undefined,
            canvasNodes,
            canvasNodesLoading: canvasLoading,
            nodeStatuses: consoleNodeStatuses,
            onTriggerNode: handleConsoleTriggerNode,
            visualDiff: {
              enabled: draftVisualDiff.visualDiffEnabled && isEditSessionUiReady,
              summary: canvasConsoleVersionDiff.draftConsoleDiffSummary,
            },
            onEffectiveConsoleChange: handleEffectiveConsoleChange,
          }}
          memory={{
            canEdit: canEditCanvasMemory({
              ...canvasAccess,
              hasEditableVersion,
            }),
            entries: canvasMemoryEntries,
            isLoading: canvasMemoryLoading,
            error: canvasMemoryError,
            deleteCanvasMemoryEntry,
            createCanvasMemoryNamespace,
            updateCanvasMemoryNamespace,
          }}
          files={{
            isEditing,
            canvasId: canvasId || undefined,
            organizationId,
            versionId: activeCanvasVersionId || undefined,
            canWrite: canStageCanvasVersion,
            files: appFiles,
            headerActionsSlotId: filesHeaderActionsSlotId,
            stagingResetNonce,
            suspendRepositoryFileStaging: isPreparingVersionAction,
            onSpecFileChange,
            onLocalFilesStagingChange: handleLocalFilesStagingChange,
            onFlushRepositoryFileStagingReady: handleFlushRepositoryFileStagingReady,
          }}
        />
        <CanvasPage
          key={canvasRenderKey}
          // In run inspection, sidebar/node params restore the run detail pane,
          // not the live node inspector.
          initialSidebar={
            runInspectionChromeActive
              ? { isOpen: false, nodeId: null }
              : {
                  isOpen: searchParams.get("sidebar") === "1",
                  nodeId: searchParams.get("node") || null,
                }
          }
          onSidebarChange={handleSidebarChange}
          onTriggerModalHostReady={registerTriggerModalHost}
          title={canvas?.metadata?.name || liveCanvas?.metadata?.name || "Canvas"}
          {...factoryEmbedCanvasChrome}
          canvasStateMode={canvasStateMode}
          onSeeCurrentVersion={handleSeeCurrentVersion}
          nodes={nodes}
          edges={renderedEdges}
          {...factoryLayoutCanvasProps}
          organizationId={organizationId}
          canvasId={canvasId}
          getSidebarData={getSidebarData}
          loadSidebarData={loadSidebarData}
          getTabData={getTabData}
          getNodeEditData={getNodeEditData}
          getAutocompleteExampleObj={getAutocompleteExampleObj}
          getCustomField={getCustomField}
          onNodeConfigurationSave={!isReadOnly ? handleNodeConfigurationSave : undefined}
          onAnnotationUpdate={!isReadOnly ? handleAnnotationUpdate : undefined}
          onAnnotationBlur={!isReadOnly ? handleAnnotationBlur : undefined}
          onEdgeCreate={!isReadOnly ? handleEdgeCreate : undefined}
          onNodeDelete={!isReadOnly ? handleNodeDelete : undefined}
          onNodesDelete={!isReadOnly ? handleNodesDelete : undefined}
          onDuplicateNodes={!isReadOnly ? handleNodesDuplicate : undefined}
          onEdgeDelete={!isReadOnly ? handleEdgeDelete : undefined}
          isAutoLayoutOnUpdateEnabled={isAutoLayoutOnUpdateEnabled && !isReadOnly}
          onToggleAutoLayoutOnUpdate={!isReadOnly ? handleToggleAutoLayoutOnUpdate : undefined}
          onToggleView={resolveFactoryAwareToggleView(isReadOnly, factoryOwnedApp, handleNodeCollapseChange)}
          onDuplicate={!isReadOnly ? handleNodeDuplicate : undefined}
          buildingBlocks={buildingBlocks}
          isEditing={isEditing}
          activeCanvasVersionId={activeCanvasVersionId}
          liveCanvasVersionId={liveCanvasVersionId}
          onAgentStagingReady={handleAgentStagingReady}
          onAgentStagingCommit={whenAllowed(canUpdateCanvas, handleAgentSidebarStagingCommit)}
          onNodeAdd={!isReadOnly ? handleNodeAdd : undefined}
          integrations={canReadIntegrations ? integrations : []}
          canReadIntegrations={canReadIntegrations}
          canCreateIntegrations={canAct("integrations", "create")}
          canUpdateIntegrations={canUpdateIntegrations}
          canUseAgents={canUseAgents}
          agentSuggestions={agentSuggestions}
          missingIntegrations={missingIntegrations}
          onConnectIntegration={!isReadOnly ? handleConnectIntegration : undefined}
          readOnly={isReadOnly || readOnlyViewModes}
          hasUserToggledSidebarRef={hasUserToggledSidebarRef}
          isSidebarOpenRef={isSidebarOpenRef}
          fitViewContentKey={`${canvasId}:${resolveFitViewVersionId({ liveCanvasVersionId, activeCanvasVersionId, isViewingDraftVersion: isEditing, draftSpec: draftSpecToRender, selectedVersion: selectedCanvasVersion })}`}
          lastFittedContentKeyRef={lastFittedContentKeyRef}
          initialFocusNodeId={initialFocusNodeIdRef.current}
          {...runParticipantFit.canvasFitProps}
          runNodeDetailNodeId={runDetailNodeId}
          runNodeDetailCanvasId={canvasId}
          runNodeDetailEdges={selectedRunCanvas?.spec?.edges}
          runNavigation={runNavigation}
          onRunNodeDetailClear={() => handleRunNodeDetailSelection(null)}
          onRunNodeDetailNavigate={handleRunNodeDetailNavigate}
          onRunNavigate={handleNavigateRun}
          onRunNavigateOlder={() => {
            void infiniteRunsQuery.fetchNextPage();
          }}
          onShowDiff={onShowDiff}
          {...canvasConsoleVersionDiff.consoleDiffHeaderProps}
          visualDiffEnabled={draftVisualDiff.visualDiffEnabled && isEditSessionUiReady}
          draftVisualDiff={isEditSessionUiReady ? draftVisualDiff : undefined}
          onToggleVisualDiff={draftVisualDiff.toggleVisualDiff}
          onShowNodeDiff={onShowNodeDiff}
          headerMode={headerMode}
          onSelectCanvasView={handleSelectCanvasView}
          enterEditModeDisabled={enterEditModeDisabled}
          enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
          exitEditModeDisabled={exitEditModeDisabled}
          exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
          {...draftChangeIndicators}
          hasStagingChanges={isEditSessionUiReady && hasStagingChanges}
          stagingStale={isEditing && stagingStale}
          hasUncommittedCanvasDraftChanges={isEditSessionUiReady && hasUncommittedCanvasDraftChanges}
          hasUncommittedConsoleDraftChanges={isEditSessionUiReady && hasUncommittedConsoleDraftChanges}
          hasUncommittedFilesDraftChanges={isEditSessionUiReady && hasUncommittedFilesDraftChanges}
          hasCommittedCanvasDraftChanges={hasCommittedCanvasDraftChanges}
          hasCommittedConsoleDraftChanges={hasCommittedConsoleDraftChanges}
          hasFilesStagingChanges={isEditSessionUiReady && hasFilesStagingChanges}
          onCommitStaging={whenAllowed(canUpdateCanvas, handleOpenCommitDialog)}
          commitStagingPending={commitStagingPending}
          resetStagingPending={resetStagingPending}
          onResetStaging={whenAllowed(canStageCanvasVersion, handleResetStaging)}
          onDiscardStaleStaging={whenAllowed(canStageCanvasVersion, handleDiscardStaleStaging)}
          discardStaleStagingPending={resetStagingPending}
          autoLayoutOnUpdateDisabled={isReadOnly}
          autoLayoutOnUpdateDisabledTooltip={isReadOnly ? "You don't have permission to edit this canvas." : undefined}
          isAutoFocusEnabled={isAutoFocusEnabled}
          onToggleAutoFocus={handleToggleAutoFocus}
          onCancelQueueItem={onCancelQueueItem}
          onCancelExecution={showLiveActivity ? onCancelExecution : undefined}
          getAllHistoryEvents={getAllHistoryEvents}
          onLoadMoreHistory={handleLoadMoreHistory}
          getHasMoreHistory={getHasMoreHistory}
          getLoadingMoreHistory={getLoadingMoreHistory}
          onLoadMoreQueue={onLoadMoreQueue}
          getAllQueueEvents={getAllQueueEvents}
          getHasMoreQueue={getHasMoreQueue}
          getLoadingMoreQueue={getLoadingMoreQueue}
          onReEmit={canUpdateCanvas && showLiveActivity ? handleReEmit : undefined}
          onRunItemOpen={showLiveActivity ? handleRunItemOpen : undefined}
          resolveRunIdForSidebarEvent={liveSidebarRunLookupEnabled ? resolveRunIdForSidebarEvent : undefined}
          fetchRunIdForSidebarEvent={runLookupEnabled ? fetchRunIdForSidebarEvent : undefined}
          onSelectRunFromSidebarEvent={runLookupEnabled ? handleSelectRunFromSidebarEvent : undefined}
          getExecutionState={getExecutionState}
          workflowNodes={canvasNodes}
          components={allComponents}
          triggers={allTriggers}
          logEntries={logEntries}
          logRuns={isViewingLiveVersion ? logRunsData.runs : []}
          runningRunsCount={runningRunsCount}
          runsNodes={canvasNodes}
          runsComponentIconMap={componentIconMap}
          onRunNodeSelect={handleLogRunNodeSelect}
          onRunExecutionSelect={handleLogRunExecutionSelect}
          onAcknowledgeErrors={canUpdateCanvas && showLiveActivity ? handleAcknowledgeErrors : undefined}
          onNodeClick={
            runInspectionChromeActive ? handleRunCanvasNodeClick : !isEditing ? handleLiveCanvasNodeClick : undefined
          }
          toolSidebarRunsContent={toolSidebarRunsContent}
          toolSidebarVersionsContent={toolSidebarVersionsContent}
          focusRequest={focusRequest}
        />
        {showDraftCanvasLoadingOverlay ? <CanvasPageLoadingOverlay message="Loading canvas..." /> : null}
        {versionCanvasLoading && !runInspectionChromeActive ? (
          <CanvasPageLoadingOverlay message="Loading version..." testId="canvas-version-loading" />
        ) : null}
      </div>
      {yamlDiffModal}
      {canvasConsoleVersionDiff.consoleYamlDiffModal}
      <CanvasPageModals
        canvasDeletedRemotely={canvasDeletedRemotely}
        onGoToCanvases={() => {
          if (organizationId) {
            navigate(`/${organizationId}`, { replace: true });
          }
        }}
      />
      <CommitStagingDialog
        open={commitDialogOpen}
        pending={commitStagingPending}
        onOpenChange={setCommitDialogOpen}
        onCommit={handleConfirmCommitStaging}
      />
      <IntegrationCreateDialog
        open={!!integrationDialogName}
        onOpenChange={(open) => !open && setIntegrationDialogName(null)}
        integrationDefinition={integrationDialogDefinition ?? null}
        organizationId={organizationId ?? ""}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={integrationDialogPendingInstance?.metadata?.name ?? integrationDialogDefinition?.name ?? ""}
        onCreated={(integrationId, instanceName) => void handleIntegrationCreated(integrationId, instanceName)}
        initialBrowserAction={integrationDialogPendingInstance?.status?.browserAction}
        initialCreatedIntegrationId={integrationDialogPendingInstance?.metadata?.id}
        initialWebhookSetup={initialWebhookSetup}
        initialConfiguration={
          integrationDialogPendingInstance?.spec?.configuration as Record<string, unknown> | undefined
        }
      />
      <ConfigureIntegrationDialog
        integrationId={agentConfigureIntegrationId}
        organizationId={organizationId ?? ""}
        onClose={() => setAgentConfigureIntegrationId(null)}
      />
    </>
  );
}
