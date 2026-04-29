import { Badge } from "@/components/ui/badge";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { countNodesByType, extractIntegrations, getTemplateTags } from "@/pages/canvas/templateMetadata";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { QueryClient } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";
import * as yaml from "js-yaml";
import debounce from "lodash.debounce";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";

import type {
  CanvasChangesetChange,
  CanvasesCanvas,
  CanvasesCanvasChangeRequest,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasVersion,
  CanvasesListEventExecutionsResponse,
  SuperplaneActionsAction,
  SuperplaneComponentsEdge as ComponentsEdge,
  ComponentsIntegrationRef,
  SuperplaneComponentsNode as ComponentsNode,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import { canvasesApplyCanvasVersionChangeset, canvasesEmitNodeEvent, canvasesUpdateNodePause } from "@/api-client";
import { useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";

import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useComponents } from "@/hooks/useComponentData";
import {
  canvasKeys,
  eventExecutionsQueryOptions,
  useActOnCanvasChangeRequest,
  useCanvas,
  useCanvasChangeRequests,
  useCanvasMemoryEntries,
  useCanvasVersion,
  useCanvasVersions,
  useCreateCanvas,
  useCreateCanvasChangeRequest,
  useCreateCanvasVersion,
  useDeleteCanvasMemoryEntry,
  useDeleteCanvasVersion,
  usePublishCanvasVersion,
  useInfiniteCanvasEvents,
  useInfiniteCanvasLiveVersions,
  useResolveCanvasChangeRequest,
  useTriggers,
  useUpdateCanvasVersion,
  useWidgets,
} from "@/hooks/useCanvasData";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import { analytics } from "@/lib/analytics";
import { getColorClass } from "@/lib/colors";
import { filterVisibleConfiguration } from "@/lib/components";
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { DefaultLayoutEngine } from "@/lib/layout";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";
import type { LogEntry, LogRunItem } from "@/ui/CanvasLogSidebar";
import type { CanvasEdge, CanvasNode, NewNodeData, NodeEditData, SidebarData } from "@/ui/CanvasPage";
import { CANVAS_SIDEBAR_STORAGE_KEY, CanvasPage, type MissingIntegration } from "@/ui/CanvasPage";
import type { EventState, EventStateMap } from "@/ui/componentBase";
import type { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { CanvasChangeRequestConflictResolver } from "./CanvasChangeRequestConflictResolver";
import { CanvasMemoryModal } from "./CanvasMemoryModal";
import { CanvasPageModals } from "./CanvasPageModals";
import { CanvasVersionControlSidebar } from "./CanvasVersionControlSidebar";
import { CanvasVersionNodeDiffDialog, type CanvasVersionNodeDiffContext } from "./CanvasVersionNodeDiffDialog";
import { CanvasYamlModal } from "./CanvasYamlModal";
import { getChangeRequestReviewPhase } from "./changeRequestReviewActions";
import { buildDraftNodeDiffSummary, hasDraftVersusLiveGraphDiff } from "./draftNodeDiff";
import { prepareAnnotationNode } from "./lib/canvas-annotation-node";
import { shouldPreserveDraftSpec } from "./lib/draft-canvas-sync";
import { prepareComponentNode, prepareTriggerNode } from "./lib/canvas-node-preparation";
import {
  isDraftVersion,
  isPublishedVersion,
  sortDraftVersionsDesc,
  sortPublishedVersionsDesc,
  versionSortValue,
} from "./lib/canvas-versions";
import { buildChangeRequestVersionRowsForStatus } from "./lib/change-requests";
import { getNodeIntegrationName, overlayIntegrationWarnings } from "./lib/node-integrations";
import { renderCanvasNodeCustomField } from "./lib/render-canvas-node-custom-field";
import { getVersionActionAvailability } from "./lib/version-action-state";
import { getCustomFieldRenderer, getState, getStateMap } from "./mappers";
import { resolveExecutionErrors } from "./mappers/dash0";
import type { User } from "./mappers/types";
import { useCancelExecutionHandler } from "./useCancelExecutionHandler";
import { useCanvasYaml } from "./useCanvasYaml";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import {
  buildCanvasStatusLogEntry,
  buildExecutionInfo,
  buildRunEntryFromEvent,
  buildRunItemFromExecution,
  buildTabData,
  buildUserInfo,
  generateNodeId,
  generateUniqueNodeName,
  mapCanvasNodesToLogEntries,
  mapExecutionsToSidebarEvents,
  mergeWorkflowLogEntries,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
  mapWorkflowEventsToRunLogEntries,
  getWorkflowSaveSignature,
  summarizeWorkflowChanges,
} from "./utils";
function getNodeAnalyticsProps(
  node: ComponentsNode,
  availableIntegrations: IntegrationsIntegrationDefinition[],
): { nodeType: string; integration: string | undefined; nodeRef: string | undefined } {
  return {
    nodeType: node.type === "TYPE_TRIGGER" ? "trigger" : node.type === "TYPE_WIDGET" ? "annotation" : "action",
    integration: getNodeIntegrationName(node, availableIntegrations),
    nodeRef: node.component,
  };
}

const CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY = "canvas-auto-layout-on-update-enabled";
const CANVAS_VERSION_CONTROL_STORAGE_KEY = "canvas-version-control-open";
const LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS = 5000;
const VERSION_ACTION_SAVE_SETTLE_TIMEOUT_MS = 5000;

type ChangeRequestAction = "ACTION_APPROVE" | "ACTION_UNAPPROVE" | "ACTION_PUBLISH" | "ACTION_REJECT" | "ACTION_REOPEN";

type CanvasSaveResult = {
  status: "saved" | "replaced" | "stale";
  workflow: CanvasesCanvas;
  savingVersionId?: string;
  matchesCurrentCanvas: boolean;
  hasQueuedFollowUp: boolean;
  response?: {
    data?: {
      version?: CanvasesCanvasVersion;
    };
  };
};

type QueuedCanvasSaveRequest = {
  workflow: CanvasesCanvas;
  savingVersionId?: string;
  resolve: (result: CanvasSaveResult) => void;
  reject: (error: unknown) => void;
};

type CanvasEchoRelease = () => void;

export function WorkflowPageV2() {
  const { organizationId, canvasId } = useParams<{
    organizationId: string;
    canvasId: string;
  }>();

  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { data: me } = useMe();
  const currentUserId = me?.id;
  const { canAct } = usePermissions();
  const [activeCanvasVersion, setActiveCanvasVersion] = useState<CanvasesCanvasVersion | null>(null);
  const [selectedChangeRequestId, setSelectedChangeRequestId] = useState("");
  const [resolvingConflictChangeRequestId, setResolvingConflictChangeRequestId] = useState("");
  const [isCreateChangeRequestMode, setIsCreateChangeRequestMode] = useState(false);
  const [createChangeRequestTitle, setCreateChangeRequestTitle] = useState("");
  const [createChangeRequestDescription, setCreateChangeRequestDescription] = useState("");
  const hasInitializedCreateChangeRequestFormRef = useRef(false);
  const [isResetDraftPending, setIsResetDraftPending] = useState(false);
  const createCanvasVersionMutation = useCreateCanvasVersion(organizationId!, canvasId!);
  const deleteCanvasVersionMutation = useDeleteCanvasVersion(organizationId!, canvasId!);
  const publishCanvasVersionMutation = usePublishCanvasVersion(organizationId!, canvasId!);
  const updateCanvasVersionMutation = useUpdateCanvasVersion(organizationId!, canvasId!);
  const [isCanvasSaveInFlight, setIsCanvasSaveInFlight] = useState(false);
  const [isCanvasSaveQueued, setIsCanvasSaveQueued] = useState(false);
  const [isPreparingVersionAction, setIsPreparingVersionAction] = useState(false);
  const createCanvasChangeRequestMutation = useCreateCanvasChangeRequest(organizationId!, canvasId!);
  const actOnCanvasChangeRequestMutation = useActOnCanvasChangeRequest(organizationId!, canvasId!);
  const resolveCanvasChangeRequestMutation = useResolveCanvasChangeRequest(organizationId!, canvasId!);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: widgets = [], isLoading: widgetsLoading } = useWidgets();
  const { data: availableIntegrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();
  const canReadIntegrations = canAct("integrations", "read");
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");
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
  const { data: organizationUsers = [], isLoading: usersLoading } = useOrganizationUsers(organizationId!);
  const { data: canvasVersions = [] } = useCanvasVersions(organizationId!, canvasId!);
  const canvasLiveVersionsQuery = useInfiniteCanvasLiveVersions(organizationId!, canvasId!, true, 10);
  const { data: canvasChangeRequests = [] } = useCanvasChangeRequests(organizationId!, canvasId!);
  const paginatedVersionPages = canvasLiveVersionsQuery.data?.pages || [];
  const paginatedVersions = useMemo(
    () => paginatedVersionPages.flatMap((page) => page?.versions || []),
    [paginatedVersionPages],
  );
  const liveCanvasVersion = useMemo(() => {
    const publishedVersions = paginatedVersions.filter(isPublishedVersion);
    if (publishedVersions.length > 0) {
      return publishedVersions[0];
    }

    return canvasVersions.filter(isPublishedVersion)[0];
  }, [paginatedVersions, canvasVersions]);
  const visibleCanvasVersions = useMemo(() => {
    const versionMap = new Map<string, CanvasesCanvasVersion>();
    const addVersion = (version: CanvasesCanvasVersion) => {
      const versionID = version.metadata?.id;
      if (!versionID || versionMap.has(versionID)) {
        return;
      }
      versionMap.set(versionID, version);
    };

    canvasVersions.forEach(addVersion);
    paginatedVersions.forEach(addVersion);

    return Array.from(versionMap.values()).filter((version) => {
      if (isPublishedVersion(version)) {
        return true;
      }

      return version.metadata?.owner?.id === currentUserId;
    });
  }, [canvasVersions, paginatedVersions, currentUserId]);
  const liveVersions = useMemo(() => sortPublishedVersionsDesc(visibleCanvasVersions), [visibleCanvasVersions]);
  const liveVersionChangeRequestsByVersionId = useMemo(() => {
    const publishedChangeRequests = canvasChangeRequests.filter(
      (changeRequest) => changeRequest.metadata?.status === "STATUS_PUBLISHED",
    );

    const pickNewestChangeRequest = (items: CanvasesCanvasChangeRequest[]): CanvasesCanvasChangeRequest | undefined => {
      return items
        .slice()
        .sort(
          (left, right) =>
            versionSortValue(right.metadata?.publishedAt || right.metadata?.updatedAt || right.metadata?.createdAt) -
            versionSortValue(left.metadata?.publishedAt || left.metadata?.updatedAt || left.metadata?.createdAt),
        )[0];
    };

    const indexedByVersionID = new Map<string, CanvasesCanvasChangeRequest[]>();
    const pushIndexed = (versionID: string | undefined, changeRequest: CanvasesCanvasChangeRequest) => {
      if (!versionID) {
        return;
      }

      const current = indexedByVersionID.get(versionID) || [];
      current.push(changeRequest);
      indexedByVersionID.set(versionID, current);
    };

    publishedChangeRequests.forEach((changeRequest) => {
      pushIndexed(changeRequest.metadata?.versionId, changeRequest);
      pushIndexed(changeRequest.version?.metadata?.id, changeRequest);
    });

    const result = new Map<string, CanvasesCanvasChangeRequest>();
    liveVersions.forEach((version) => {
      const versionID = version.metadata?.id;
      if (!versionID) {
        return;
      }

      const directMatch = pickNewestChangeRequest(indexedByVersionID.get(versionID) || []);
      if (directMatch) {
        result.set(versionID, directMatch);
        return;
      }

      const versionPublishedAt = versionSortValue(version.metadata?.publishedAt);
      if (!versionPublishedAt) {
        return;
      }

      const timestampFallback = publishedChangeRequests
        .map((changeRequest) => {
          const requestPublishedAt = versionSortValue(changeRequest.metadata?.publishedAt);
          if (!requestPublishedAt) {
            return null;
          }
          return {
            changeRequest,
            distance: Math.abs(requestPublishedAt - versionPublishedAt),
            score: requestPublishedAt,
          };
        })
        .filter(
          (item): item is { changeRequest: CanvasesCanvasChangeRequest; distance: number; score: number } => !!item,
        )
        .filter((item) => item.distance <= 3_000)
        .sort((left, right) => left.distance - right.distance || right.score - left.score)[0];

      if (timestampFallback) {
        result.set(versionID, timestampFallback.changeRequest);
      }
    });

    return result;
  }, [canvasChangeRequests, liveVersions]);
  const liveVersionOwnerProfilesById = useMemo(() => {
    const profilesByID = new Map<string, { name: string; avatarUrl?: string }>();
    organizationUsers.forEach((user) => {
      const userID = user.metadata?.id;
      if (!userID) {
        return;
      }

      profilesByID.set(userID, {
        name: user.spec?.displayName || user.metadata?.email || userID,
        avatarUrl: user.status?.accountProviders?.[0]?.avatarUrl || undefined,
      });
    });
    return profilesByID;
  }, [organizationUsers]);
  const pendingApprovalVersions = useMemo(
    () => buildChangeRequestVersionRowsForStatus(canvasChangeRequests, visibleCanvasVersions, "open"),
    [canvasChangeRequests, visibleCanvasVersions],
  );
  const rejectedVersions = useMemo(
    () => buildChangeRequestVersionRowsForStatus(canvasChangeRequests, visibleCanvasVersions, "reject"),
    [canvasChangeRequests, visibleCanvasVersions],
  );
  const pendingApprovalVersionIds = useMemo(() => {
    const ids = new Set<string>();
    pendingApprovalVersions.forEach((item) => {
      const id = item.version.metadata?.id;
      if (!id) {
        return;
      }
      ids.add(id);
    });
    return ids;
  }, [pendingApprovalVersions]);
  const selectableVersionsById = useMemo(() => {
    const indexedVersions = new Map<string, CanvasesCanvasVersion>();
    visibleCanvasVersions.forEach((version) => {
      const id = version.metadata?.id;
      if (!id) {
        return;
      }
      indexedVersions.set(id, version);
    });
    pendingApprovalVersions.forEach((item) => {
      const id = item.version.metadata?.id;
      if (!id || indexedVersions.has(id)) {
        return;
      }
      indexedVersions.set(id, item.version);
    });
    return indexedVersions;
  }, [visibleCanvasVersions, pendingApprovalVersions]);
  const draftVersions = useMemo(() => sortDraftVersionsDesc(visibleCanvasVersions), [visibleCanvasVersions]);
  const hasMoreLiveVersions = canvasLiveVersionsQuery.hasNextPage || false;
  const isLoadingMoreLiveVersions = canvasLiveVersionsQuery.isFetchingNextPage;
  const liveCanvasVersionId = liveCanvasVersion?.metadata?.id;
  const activeCanvasVersionId = activeCanvasVersion?.metadata?.id || "";
  const {
    data: loadedCanvasVersion,
    isLoading: loadedCanvasVersionLoading,
    isFetching: loadedCanvasVersionFetching,
  } = useCanvasVersion(organizationId!, canvasId!, activeCanvasVersionId, !!activeCanvasVersionId);
  const selectedCanvasVersion = activeCanvasVersionId ? loadedCanvasVersion || activeCanvasVersion : null;
  const createChangeRequestVersion = useMemo(() => {
    const selectedVersionID = selectedCanvasVersion?.metadata?.id || "";
    const isPendingApprovalVersion = pendingApprovalVersionIds.has(selectedVersionID);
    if (
      activeCanvasVersionId &&
      selectedCanvasVersion &&
      isDraftVersion(selectedCanvasVersion) &&
      !isPendingApprovalVersion
    ) {
      return selectedCanvasVersion;
    }

    return draftVersions[0];
  }, [activeCanvasVersionId, selectedCanvasVersion, draftVersions, pendingApprovalVersionIds]);
  const latestDraftVersion = draftVersions[0];
  const createChangeRequestNodeDiffSummary = useMemo(
    () => buildDraftNodeDiffSummary(liveCanvasVersion, createChangeRequestVersion),
    [liveCanvasVersion, createChangeRequestVersion],
  );
  const isCreateChangeRequestDraftOutdated = useMemo(() => {
    const liveCreatedAt = versionSortValue(liveCanvasVersion?.metadata?.createdAt);
    const draftCreatedAt = versionSortValue(createChangeRequestVersion?.metadata?.createdAt);
    if (!liveCreatedAt || !draftCreatedAt) {
      return false;
    }
    return liveCreatedAt > draftCreatedAt;
  }, [liveCanvasVersion?.metadata?.createdAt, createChangeRequestVersion?.metadata?.createdAt]);
  const hasDraftGraphDiffVersusLive = useMemo(
    () => hasDraftVersusLiveGraphDiff(liveCanvasVersion, latestDraftVersion),
    [liveCanvasVersion, latestDraftVersion],
  );
  const selectedCanvasVersionID = selectedCanvasVersion?.metadata?.id || "";
  const isViewingPendingApprovalVersion =
    !!selectedCanvasVersionID && pendingApprovalVersionIds.has(selectedCanvasVersionID);
  const isViewingDraftVersion =
    !!selectedCanvasVersion && isDraftVersion(selectedCanvasVersion) && !isViewingPendingApprovalVersion;
  const isViewingCurrentLiveVersion =
    !selectedCanvasVersion || selectedCanvasVersion.metadata?.id === liveCanvasVersionId;
  const isViewingLiveVersion = isViewingCurrentLiveVersion;
  const [draftCanvasSpec, setDraftCanvasSpec] = useState<CanvasesCanvas["spec"] | null>(null);
  const draftSpecToRender = draftCanvasSpec ?? selectedCanvasVersion?.spec ?? null;

  useEffect(() => {
    if (!isViewingDraftVersion || !activeCanvasVersionId) {
      return;
    }

    const preservedDraftSpec = draftCanvasSpecsRef.current.get(activeCanvasVersionId);
    if (preservedDraftSpec) {
      setDraftCanvasSpec(preservedDraftSpec);
      return;
    }

    const nextDraftSpec = selectedCanvasVersion?.spec ?? null;
    if (nextDraftSpec) {
      draftCanvasSpecsRef.current.set(activeCanvasVersionId, nextDraftSpec);
    }
    setDraftCanvasSpec(nextDraftSpec);
  }, [isViewingDraftVersion, activeCanvasVersionId, selectedCanvasVersion?.metadata?.id, selectedCanvasVersion?.spec]);

  useEffect(() => {
    if (!isViewingDraftVersion || !activeCanvasVersionId || !liveCanvas?.spec) {
      return;
    }

    if (!draftCanvasSpec && !selectedCanvasVersion?.spec) {
      return;
    }

    if (
      shouldPreserveDraftSpec({
        incomingSpec: liveCanvas.spec,
        draftSpec: draftCanvasSpec,
        selectedDraftVersionSpec: selectedCanvasVersion?.spec,
        liveVersionSpec: liveCanvasVersion?.spec,
      })
    ) {
      return;
    }

    setDraftCanvasSpec((currentDraftSpec) => {
      draftCanvasSpecsRef.current.set(activeCanvasVersionId, liveCanvas.spec);
      if (currentDraftSpec === liveCanvas.spec) {
        return currentDraftSpec;
      }

      return liveCanvas.spec;
    });
  }, [
    isViewingDraftVersion,
    activeCanvasVersionId,
    liveCanvas?.spec,
    liveCanvasVersion?.spec,
    selectedCanvasVersion?.spec,
    draftCanvasSpec,
  ]);

  const canvas = useMemo(() => {
    if (!liveCanvas) {
      return liveCanvas;
    }

    // Draft editing uses the local query cache as source of truth so
    // optimistic/local edits are not overwritten by slower version fetches.
    if (isViewingDraftVersion) {
      if (draftSpecToRender) {
        return {
          ...liveCanvas,
          spec: draftSpecToRender,
        };
      }

      return null;
    }

    const versionSpec = selectedCanvasVersion?.spec;
    if (!versionSpec) {
      return liveCanvas;
    }

    return {
      ...liveCanvas,
      spec: versionSpec,
    };
  }, [liveCanvas, selectedCanvasVersion, isViewingDraftVersion, draftSpecToRender]);
  const isChangeManagementDisabled = !(
    liveCanvas?.spec?.changeManagement?.enabled ??
    liveCanvasVersion?.spec?.changeManagement?.enabled ??
    false
  );
  const isEditing = !!activeCanvasVersionId && isViewingDraftVersion;
  const hasEditableVersion = !!activeCanvasVersionId && isViewingDraftVersion;
  const infiniteEventsQuery = useInfiniteCanvasEvents(canvasId!, isViewingLiveVersion);
  const runsEventsData = useMemo(() => {
    const pages = infiniteEventsQuery.data?.pages || [];
    const seen = new Set<string>();
    const events = pages
      .flatMap((page) => page?.events || [])
      .filter((e) => {
        if (!e.id || seen.has(e.id)) return false;
        seen.add(e.id);
        return true;
      });
    const totalCount = pages[0]?.totalCount || 0;
    return { events, totalCount };
  }, [infiniteEventsQuery.data]);
  const canvasEventsResponse = infiniteEventsQuery.data?.pages?.[0];
  const componentIconMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const c of components) {
      if (c.name && c.icon) map[c.name] = c.icon;
    }
    for (const t of triggers) {
      if (t.name && t.icon) map[t.name] = t.icon;
    }
    return map;
  }, [components, triggers]);
  const {
    data: canvasMemoryEntries = [],
    isLoading: canvasMemoryLoading,
    error: canvasMemoryError,
  } = useCanvasMemoryEntries(canvasId!, isViewingLiveVersion);
  const deleteCanvasMemoryEntry = useDeleteCanvasMemoryEntry(canvasId!);
  const canUpdateCanvas = canAct("canvases", "update");
  usePageTitle([canvas?.metadata?.name || "Canvas"]);

  const isTemplate = liveCanvas?.metadata?.isTemplate ?? false;
  const [canvasDeletedRemotely, setCanvasDeletedRemotely] = useState(false);
  const [remoteCanvasUpdatePending, setRemoteCanvasUpdatePending] = useState(false);
  const isReadOnly = isTemplate || !canUpdateCanvas || canvasDeletedRemotely || !hasEditableVersion;
  const [isUseTemplateOpen, setIsUseTemplateOpen] = useState(false);
  const [isYamlViewModalOpen, setIsYamlViewModalOpen] = useState(false);
  const [isMemoryViewModalOpen, setIsMemoryViewModalOpen] = useState(false);
  const [isVersionControlOpen, setIsVersionControlOpen] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }

    const stored = window.localStorage.getItem(CANVAS_VERSION_CONTROL_STORAGE_KEY);
    if (stored === null) {
      return false;
    }

    try {
      return JSON.parse(stored) as boolean;
    } catch {
      return false;
    }
  });
  /** After creating a change request, hide draft Discard until the user enters edit mode again. */
  const [suppressUnpublishedDraftDiscard, setSuppressUnpublishedDraftDiscard] = useState(false);
  const [versionNodeDiffContext, setVersionNodeDiffContext] = useState<CanvasVersionNodeDiffContext | null>(null);
  const versionNodeDiffLiveChangeRequest = useMemo(() => {
    const fallback = versionNodeDiffContext?.changeRequest;
    const id = fallback?.metadata?.id;
    if (!id) {
      return fallback;
    }
    return canvasChangeRequests.find((c) => c.metadata?.id === id) ?? fallback;
  }, [canvasChangeRequests, versionNodeDiffContext?.changeRequest]);
  const resolvingConflictChangeRequest = useMemo(() => {
    if (!resolvingConflictChangeRequestId) {
      return undefined;
    }
    return canvasChangeRequests.find((c) => c.metadata?.id === resolvingConflictChangeRequestId);
  }, [canvasChangeRequests, resolvingConflictChangeRequestId]);
  const createWorkflowMutation = useCreateCanvas(organizationId!);

  // Warm up org users and roles cache so approval specs can pretty-print
  // user IDs as emails and role names as display names.
  // We don't use the values directly here; loading them populates the
  // react-query cache which prepareApprovalNode reads from.
  useOrganizationRoles(organizationId!);

  /**
   * Track if we've already done the initial fit to view.
   * This ref persists across re-renders to prevent viewport changes on save.
   */
  const hasFitToViewRef = useRef(false);
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
    // Initialize on first render
    isSidebarOpenRef.current = canvas.spec?.nodes?.length === 0;
  }

  /**
   * Track the canvas viewport state.
   * This ref persists across re-renders to preserve viewport position and zoom.
   */
  const viewportRef = useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);

  // Track unsaved changes on the canvas
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [hasNonPositionalUnsavedChanges, setHasNonPositionalUnsavedChanges] = useState(false);
  const [isPositionAutoSaveQueued, setIsPositionAutoSaveQueued] = useState(false);
  const [isAnnotationAutoSaveQueued, setIsAnnotationAutoSaveQueued] = useState(false);

  const isAutoSaveQueued = isPositionAutoSaveQueued || isAnnotationAutoSaveQueued;
  const hasLocalSaveActivity = isCanvasSaveInFlight || isCanvasSaveQueued || isAutoSaveQueued;
  const hasPendingLocalCanvasState = hasUnsavedChanges || hasLocalSaveActivity;
  const [isAutoLayoutOnUpdateEnabled, setIsAutoLayoutOnUpdateEnabled] = useState(() => {
    if (typeof window !== "undefined") {
      const stored = window.localStorage.getItem(CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY);
      return stored !== null ? JSON.parse(stored) : false;
    }
    return false;
  });

  useEffect(() => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_VERSION_CONTROL_STORAGE_KEY, JSON.stringify(isVersionControlOpen));
    }
  }, [isVersionControlOpen]);

  useEffect(() => {
    if (!isCreateChangeRequestMode) {
      hasInitializedCreateChangeRequestFormRef.current = false;
      return;
    }

    if (hasInitializedCreateChangeRequestFormRef.current) {
      return;
    }

    hasInitializedCreateChangeRequestFormRef.current = true;
    const nextVersionNumber = canvasChangeRequests.length + 1;
    setCreateChangeRequestTitle(`v${nextVersionNumber}`);
    setCreateChangeRequestDescription("");
  }, [isCreateChangeRequestMode, canvasChangeRequests.length]);

  useEffect(() => {
    if (!hasEditableVersion || !isVersionControlOpen) {
      return;
    }

    setIsVersionControlOpen(false);
  }, [hasEditableVersion, isVersionControlOpen]);

  const lastSavedWorkflowRef = useRef<CanvasesCanvas | null>(null);
  const lastSavedWorkflowSignatureRef = useRef("");
  const lastAppliedVersionSnapshotRef = useRef("");
  const canvasRef = useRef<CanvasesCanvas | null>(canvas ?? null);
  const activeCanvasVersionIdRef = useRef<string>(activeCanvasVersionId);
  const draftCanvasSpecsRef = useRef<Map<string, CanvasesCanvas["spec"] | null>>(new Map());
  const queuedCanvasSaveRef = useRef<QueuedCanvasSaveRequest | null>(null);
  const isDrainingCanvasSaveQueueRef = useRef(false);
  const hasTrackedCanvasView = useRef(false);
  const canvasSaveSessionRef = useRef(0);
  const ignoredCanvasUpdatedEchoReleasesRef = useRef<Array<CanvasEchoRelease>>([]);
  const ignoredCanvasVersionUpdatedEchoReleasesRef = useRef<Map<string, Array<CanvasEchoRelease>>>(new Map());
  const setLastSavedWorkflowSnapshot = useCallback((workflow: CanvasesCanvas | null) => {
    if (!workflow) {
      lastSavedWorkflowRef.current = null;
      lastSavedWorkflowSignatureRef.current = "";
      return;
    }

    const snapshot = JSON.parse(JSON.stringify(workflow)) as CanvasesCanvas;
    lastSavedWorkflowRef.current = snapshot;
    lastSavedWorkflowSignatureRef.current = getWorkflowSaveSignature(snapshot);
  }, []);
  const clearQueuedAutoSaveFlags = useCallback(() => {
    setIsPositionAutoSaveQueued(false);
    setIsAnnotationAutoSaveQueued(false);
  }, []);
  useEffect(() => {
    canvasRef.current = canvas ?? null;
  }, [canvas]);
  useEffect(() => {
    activeCanvasVersionIdRef.current = activeCanvasVersionId;
  }, [activeCanvasVersionId]);

  const applyLocalWorkflowUpdate = useCallback(
    (updatedWorkflow: CanvasesCanvas) => {
      if (!organizationId || !canvasId) {
        return;
      }

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (!isViewingDraftVersion || !activeCanvasVersionId || !updatedWorkflow.spec) {
        return;
      }

      draftCanvasSpecsRef.current.set(activeCanvasVersionId, updatedWorkflow.spec);
      setDraftCanvasSpec(updatedWorkflow.spec);
      setActiveCanvasVersion((current) =>
        current?.metadata?.id === activeCanvasVersionId ? { ...current, spec: updatedWorkflow.spec } : current,
      );
      queryClient.setQueryData<CanvasesCanvasVersion | undefined>(
        canvasKeys.versionDetail(canvasId, activeCanvasVersionId),
        (current) => (current ? { ...current, spec: updatedWorkflow.spec } : current),
      );
    },
    [organizationId, canvasId, queryClient, isViewingDraftVersion, activeCanvasVersionId],
  );

  // Use Zustand store for execution data - extract only the methods to avoid recreating callbacks
  // Subscribe to version to ensure React detects all updates
  const storeVersion = useNodeExecutionStore((state) => state.version);
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);
  const loadNodeDataMethod = useNodeExecutionStore((state) => state.loadNodeData);
  const initializeFromWorkflow = useNodeExecutionStore((state) => state.initializeFromWorkflow);

  // Redirect to home page if workflow is not found (404)
  // Use replace to avoid back button issues and prevent 404 flash
  useEffect(() => {
    if (canvasError && !canvasLoading) {
      // Check if it's a 404 error
      const is404 =
        (canvasError as any)?.status === 404 ||
        (canvasError as any)?.response?.status === 404 ||
        (canvasError as any)?.code === "NOT_FOUND" ||
        (canvasError as any)?.message?.includes("not found") ||
        (canvasError as any)?.message?.includes("404");

      if (is404 && organizationId && !canvasDeletedRemotely) {
        navigate(`/${organizationId}`, { replace: true });
      }
    }
  }, [canvasError, canvasLoading, navigate, organizationId, canvasDeletedRemotely]);
  useEffect(() => {
    if (hasTrackedCanvasView.current) return;
    if (!canvas || !canvasId || !organizationId) return;
    if (canvasLoading) return;
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
    if (!canvas) {
      return;
    }

    if (!lastSavedWorkflowRef.current) {
      setLastSavedWorkflowSnapshot(canvas);
    }
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
    setHasUnsavedChanges(false);
    setHasNonPositionalUnsavedChanges(false);
    setActiveCanvasVersion(null);
    setSelectedChangeRequestId("");
    hasSyncedVersionFromURLRef.current = false;
    setLastSavedWorkflowSnapshot(null);
    ignoredCanvasUpdatedEchoReleasesRef.current = [];
    ignoredCanvasVersionUpdatedEchoReleasesRef.current.clear();
    draftCanvasSpecsRef.current.clear();
    isDrainingCanvasSaveQueueRef.current = false;
    setIsCanvasSaveInFlight(false);
    setIsCanvasSaveQueued(false);
    hasInitializedStoreRef.current = null;
    pendingStoreReinitRef.current = true;
  }, [canvasId, setLastSavedWorkflowSnapshot]);

  useEffect(() => {
    if (isTemplate) {
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
    }
  }, [isTemplate, canvasId]);

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

    const isPublished = isPublishedVersion(requestedVersion);
    const isOwnedDraft = !isPublished && requestedVersion.metadata?.owner?.id === currentUserId;
    const isPendingApprovalVersion = pendingApprovalVersionIds.has(requestedVersion.metadata?.id || "");
    const isCurrentLive = requestedVersion.metadata?.id === liveCanvasVersionId;
    if (!isOwnedDraft && !isPublished && !isPendingApprovalVersion) {
      hasSyncedVersionFromURLRef.current = true;
      return;
    }

    if (isCurrentLive) {
      setActiveCanvasVersion(null);
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        next.delete("version");
        return next;
      });
      hasSyncedVersionFromURLRef.current = true;
      return;
    }

    setActiveCanvasVersion(requestedVersion);
    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId!, canvasId!), (current) => {
      if (!current || !requestedVersion.spec) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...requestedVersion.spec },
      };
    });
    hasSyncedVersionFromURLRef.current = true;
  }, [
    selectableVersionsById,
    activeCanvasVersionId,
    selectedCanvasVersion?.metadata?.id,
    searchParams,
    currentUserId,
    pendingApprovalVersionIds,
    liveCanvasVersionId,
    setSearchParams,
    queryClient,
    organizationId,
    canvasId,
  ]);

  useEffect(() => {
    if (!canvasChangeRequests.length) {
      if (selectedChangeRequestId) {
        setSelectedChangeRequestId("");
      }
      return;
    }

    if (!selectedChangeRequestId) {
      return;
    }

    const hasSelected = canvasChangeRequests.some(
      (changeRequest) => changeRequest.metadata?.id === selectedChangeRequestId,
    );
    if (!hasSelected) {
      setSelectedChangeRequestId("");
    }
  }, [canvasChangeRequests, selectedChangeRequestId]);

  useEffect(() => {
    if (!resolvingConflictChangeRequestId) {
      return;
    }
    const stillExists = canvasChangeRequests.some(
      (changeRequest) => changeRequest.metadata?.id === resolvingConflictChangeRequestId,
    );
    if (!stillExists) {
      setResolvingConflictChangeRequestId("");
    }
  }, [canvasChangeRequests, resolvingConflictChangeRequestId]);

  useEffect(() => {
    if (
      !organizationId ||
      !canvasId ||
      !activeCanvasVersionId ||
      !loadedCanvasVersion?.spec ||
      hasPendingLocalCanvasState
    ) {
      return;
    }

    const loadedVersionID = loadedCanvasVersion.metadata?.id;
    if (!loadedVersionID || loadedVersionID !== activeCanvasVersionId) {
      return;
    }

    const snapshotKey = `${loadedVersionID}:${loadedCanvasVersion.metadata?.updatedAt || ""}`;
    if (lastAppliedVersionSnapshotRef.current === snapshotKey) {
      return;
    }

    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...loadedCanvasVersion.spec },
      };
    });

    lastAppliedVersionSnapshotRef.current = snapshotKey;
  }, [organizationId, canvasId, activeCanvasVersionId, loadedCanvasVersion, queryClient, hasPendingLocalCanvasState]);

  useEffect(() => {
    if (
      !remoteCanvasUpdatePending ||
      hasPendingLocalCanvasState ||
      canvasDeletedRemotely ||
      !organizationId ||
      !canvasId
    ) {
      return;
    }

    queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
    if (isViewingLiveVersion) {
      queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    } else if (activeCanvasVersionId) {
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(canvasId, activeCanvasVersionId) });
    }

    setRemoteCanvasUpdatePending(false);
  }, [
    remoteCanvasUpdatePending,
    hasPendingLocalCanvasState,
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
  const visibleNodeExecutionsMap = isViewingLiveVersion ? nodeExecutionsMap : {};
  const visibleNodeQueueItemsMap = isViewingLiveVersion ? nodeQueueItemsMap : {};
  const visibleNodeEventsMap = isViewingLiveVersion ? nodeEventsMap : {};

  // Execution chain data utilities for lazy loading
  const { loadExecutionChain } = useExecutionChainData(canvasId!, queryClient, canvas ?? undefined);

  const registerIgnoredCanvasUpdatedEcho = useCallback(() => {
    const saveSession = canvasSaveSessionRef.current;
    let released = false;
    let timeoutId = 0;
    const release = () => {
      if (released) {
        return;
      }

      released = true;
      window.clearTimeout(timeoutId);
      const releaseIndex = ignoredCanvasUpdatedEchoReleasesRef.current.indexOf(release);
      if (releaseIndex >= 0) {
        ignoredCanvasUpdatedEchoReleasesRef.current.splice(releaseIndex, 1);
      }

      if (canvasSaveSessionRef.current !== saveSession) {
        return;
      }
    };

    ignoredCanvasUpdatedEchoReleasesRef.current.push(release);
    timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

    return release;
  }, []);

  const registerIgnoredCanvasVersionUpdatedEcho = useCallback((savingVersionId?: string) => {
    if (!savingVersionId) {
      return () => undefined;
    }

    const saveSession = canvasSaveSessionRef.current;
    const currentReleases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(savingVersionId) || [];
    let released = false;
    let timeoutId = 0;
    const release = () => {
      if (released) {
        return;
      }

      released = true;
      window.clearTimeout(timeoutId);
      const releases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(savingVersionId);
      if (releases) {
        const releaseIndex = releases.indexOf(release);
        if (releaseIndex >= 0) {
          releases.splice(releaseIndex, 1);
        }
        if (releases.length === 0) {
          ignoredCanvasVersionUpdatedEchoReleasesRef.current.delete(savingVersionId);
        }
      }

      if (canvasSaveSessionRef.current !== saveSession) {
        return;
      }
    };

    currentReleases.push(release);
    ignoredCanvasVersionUpdatedEchoReleasesRef.current.set(savingVersionId, currentReleases);
    timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

    return release;
  }, []);

  const consumeIgnoredCanvasUpdatedEcho = useCallback(() => {
    const release = ignoredCanvasUpdatedEchoReleasesRef.current.pop();
    if (!release) {
      return false;
    }

    release();
    return true;
  }, []);

  const consumeIgnoredCanvasVersionUpdatedEcho = useCallback((versionId?: string) => {
    if (!versionId) {
      return false;
    }

    const releases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(versionId);
    if (!releases) {
      return false;
    }

    const release = releases.pop();
    if (!release) {
      return false;
    }

    if (releases.length === 0) {
      ignoredCanvasVersionUpdatedEchoReleasesRef.current.delete(versionId);
    }

    release();
    return true;
  }, []);

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
      const currentWorkflow = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId!, canvasId!));
      return getWorkflowSaveSignature(currentWorkflow) === getWorkflowSaveSignature(workflow);
    },
    [organizationId, canvasId, queryClient],
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

      const expectedVersionId = request.savingVersionId || liveCanvasVersionId || undefined;
      const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho(expectedVersionId);
      const releaseCanvasUpdatedEcho = registerIgnoredCanvasUpdatedEcho();

      try {
        const response = await updateCanvasVersionMutation.mutateAsync({
          versionId: request.savingVersionId,
          name: request.workflow.metadata?.name ?? "",
          description: request.workflow.metadata?.description,
          nodes: request.workflow.spec?.nodes,
          edges: request.workflow.spec?.edges,
          preserveLocalCanvasState: true,
          invalidateRelatedQueries: false,
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
        releaseCanvasVersionUpdatedEcho();
        request.reject(error);
      }
    },
    [
      liveCanvasVersionId,
      registerIgnoredCanvasUpdatedEcho,
      registerIgnoredCanvasVersionUpdatedEcho,
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

  const handleCreateVersion = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    if (!canUpdateCanvas) {
      showErrorToast("You don't have permission to edit this canvas");
      return;
    }

    if (isTemplate) {
      showErrorToast("Template canvases are read-only");
      return;
    }

    if (hasEditableVersion && hasUnsavedChanges) {
      const shouldCreate = window.confirm(
        "You have unsaved changes in the current draft. Create a new draft from live anyway?",
      );
      if (!shouldCreate) {
        return;
      }
    }

    try {
      const response = await createCanvasVersionMutation.mutateAsync();
      const version = response?.data?.version;
      if (!version) {
        showErrorToast("Failed to create canvas version");
        return;
      }

      activeCanvasVersionIdRef.current = version.metadata?.id || "";
      setActiveCanvasVersion(version);
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
      setLastSavedWorkflowSnapshot(null);
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (version.metadata?.id) {
          next.set("version", version.metadata.id);
        }
        return next;
      });

      queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
        if (!current || !version.spec) {
          return current;
        }

        return {
          ...current,
          metadata: {
            ...current.metadata,
            name: version.metadata?.name ?? current.metadata?.name,
            description: version.metadata?.description ?? current.metadata?.description,
          },
          spec: { ...current.spec, ...version.spec },
        };
      });
    } catch (error) {
      const errorMessage =
        (error as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        (error as { message?: string })?.message ||
        "Failed to create version";
      showErrorToast(getUsageLimitToastMessage(error, errorMessage));
    }
  }, [
    organizationId,
    canvasId,
    canUpdateCanvas,
    isTemplate,
    hasEditableVersion,
    hasUnsavedChanges,
    createCanvasVersionMutation,
    queryClient,
    setSearchParams,
    setLastSavedWorkflowSnapshot,
  ]);

  const handleToggleAutoLayoutOnUpdate = useCallback(() => {
    const newValue = !isAutoLayoutOnUpdateEnabled;
    setIsAutoLayoutOnUpdateEnabled(newValue);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY, JSON.stringify(newValue));
    }
  }, [isAutoLayoutOnUpdateEnabled]);

  const applyAutoLayoutOnAddedNode = useCallback(
    async (workflow: CanvasesCanvas, nodeID?: string): Promise<CanvasesCanvas> => {
      if (!isAutoLayoutOnUpdateEnabled || !nodeID) {
        return workflow;
      }

      const node = workflow.spec?.nodes?.find((candidate) => candidate.id === nodeID);
      if (!node || node.type === "TYPE_WIDGET") {
        return workflow;
      }

      return DefaultLayoutEngine.apply(workflow, {
        scope: "connected-component",
        nodeIds: [nodeID],
        components,
      });
    },
    [isAutoLayoutOnUpdateEnabled, components],
  );

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
  const logNodeSelectRef = useRef<(nodeId: string) => void>(() => {});

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
        async () => {
          setIsPositionAutoSaveQueued(false);
          if (!organizationId || !canvasId) return;

          const positionUpdates = new Map(pendingPositionUpdatesRef.current);
          if (positionUpdates.size === 0) return;
          const focusedNoteId = getActiveNoteId();

          try {
            if (isReadOnly) {
              return;
            }

            // Check if there are unsaved structural changes
            // If so, skip auto-save to avoid saving those changes accidentally
            if (hasNonPositionalUnsavedChanges) {
              return;
            }

            // Fetch the latest workflow from the cache
            const latestWorkflow = queryClient.getQueryData<CanvasesCanvas>(
              canvasKeys.detail(organizationId, canvasId),
            );

            if (!latestWorkflow?.spec?.nodes) return;

            // Apply only position updates to the current state
            const updatedNodes = latestWorkflow.spec.nodes.map((node) => {
              if (!node.id) return node;

              const positionUpdate = positionUpdates.get(node.id);
              if (positionUpdate) {
                return {
                  ...node,
                  position: positionUpdate,
                };
              }
              return node;
            });

            const updatedWorkflow = {
              ...latestWorkflow,
              spec: {
                ...latestWorkflow.spec,
                nodes: updatedNodes,
              },
            };

            const changeSummary = summarizeWorkflowChanges({
              before: lastSavedWorkflowRef.current,
              after: updatedWorkflow,
              onNodeSelect: (nodeId: string) => logNodeSelectRef.current(nodeId),
            });
            const changeMessage = changeSummary.changeCount
              ? `${changeSummary.changeCount} Canvas changes saved`
              : "Canvas changes saved";

            // Save the workflow with updated positions
            if (!activeCanvasVersionId) {
              return;
            }
            const savingVersionID = activeCanvasVersionId || undefined;

            const saveResult = await enqueueCanvasSave(updatedWorkflow, savingVersionID);
            if (saveResult.status !== "saved") {
              return;
            }
            if (
              saveResult.response?.data?.version &&
              savingVersionID &&
              activeCanvasVersionIdRef.current === savingVersionID
            ) {
              setActiveCanvasVersion(saveResult.response.data.version);
            }
            if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
              return;
            }

            if (changeSummary.detail) {
              setLiveCanvasEntries((prev) => [
                buildCanvasStatusLogEntry({
                  id: `canvas-save-${Date.now()}`,
                  message: changeMessage,
                  type: "success",
                  timestamp: new Date().toISOString(),
                  detail: changeSummary.detail,
                  searchText: changeSummary.searchText,
                }),
                ...prev,
              ]);
            }

            setLastSavedWorkflowSnapshot(updatedWorkflow);

            // Clear the saved position updates after successful save
            // Keep any new updates that came in during the save
            positionUpdates.forEach((_, nodeId) => {
              if (pendingPositionUpdatesRef.current.get(nodeId) === positionUpdates.get(nodeId)) {
                pendingPositionUpdatesRef.current.delete(nodeId);
              }
            });

            // After save, merge any new pending updates into the cache
            // This prevents the server response from overwriting newer local changes
            const currentWorkflow = queryClient.getQueryData<CanvasesCanvas>(
              canvasKeys.detail(organizationId, canvasId),
            );

            if (currentWorkflow?.spec?.nodes && pendingPositionUpdatesRef.current.size > 0) {
              const mergedNodes = currentWorkflow.spec.nodes.map((node) => {
                if (!node.id) return node;

                const pendingUpdate = pendingPositionUpdatesRef.current.get(node.id);
                if (pendingUpdate) {
                  return {
                    ...node,
                    position: pendingUpdate,
                  };
                }
                return node;
              });

              queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), {
                ...currentWorkflow,
                spec: {
                  ...currentWorkflow.spec,
                  nodes: mergedNodes,
                },
              });
            }

            // Auto-save completed silently (no toast or state changes)
          } catch (error) {
            console.error("Failed to auto-save", error);
          } finally {
            if (focusedNoteId) {
              requestAnimationFrame(() => {
                restoreActiveNoteFocus();
              });
            }
          }
        },
        isReadOnly ? 2000 : 100,
      ),
    [
      organizationId,
      canvasId,
      activeCanvasVersionId,
      queryClient,
      hasNonPositionalUnsavedChanges,
      isReadOnly,
      enqueueCanvasSave,
      setLastSavedWorkflowSnapshot,
    ],
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
      if (hasPendingLocalCanvasState) {
        e.preventDefault();
        e.returnValue = "Your work isn't saved, unsaved changes will be lost. Are you sure you want to leave?";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasPendingLocalCanvasState]);

  // Merge triggers and components from applications into the main arrays
  const allTriggers = useMemo(() => {
    const merged = [...triggers];
    availableIntegrations.forEach((integration) => {
      if (integration.triggers) {
        merged.push(...integration.triggers);
      }
    });
    return merged;
  }, [triggers, availableIntegrations]);

  const allComponents = useMemo(() => {
    const merged = [...components];
    availableIntegrations.forEach((integration) => {
      if (integration.actions) {
        merged.push(...integration.actions);
      }
    });
    return merged;
  }, [components, availableIntegrations]);

  const buildingBlocks = useMemo(
    () => buildBuildingBlockCategories(triggers, components, availableIntegrations),
    [triggers, components, availableIntegrations],
  );
  const canvasMode = hasEditableVersion ? "edit" : "live";

  const { nodes: preparedNodes, edges } = useMemo(() => {
    if (!canvas || canvasLoading || triggersLoading || componentsLoading || integrationsLoading) {
      return { nodes: [], edges: [] };
    }

    return prepareData(
      canvas,
      allTriggers,
      allComponents,
      visibleNodeEventsMap,
      visibleNodeExecutionsMap,
      visibleNodeQueueItemsMap,
      canvasId!,
      queryClient,
      me,
      canvasMode,
    );
  }, [
    canvas,
    allTriggers,
    allComponents,
    visibleNodeEventsMap,
    visibleNodeExecutionsMap,
    visibleNodeQueueItemsMap,
    canvasId,
    queryClient,
    canvasLoading,
    triggersLoading,
    componentsLoading,
    integrationsLoading,
    organizationId,
    me,
    canvasMode,
  ]);

  const nodesWithIntegrationStatus = useMemo(
    () => overlayIntegrationWarnings(preparedNodes, integrations, canvas?.spec?.nodes),
    [preparedNodes, integrations, canvas?.spec?.nodes],
  );

  const nodes = nodesWithIntegrationStatus;

  const getSidebarData = useCallback(
    (nodeId: string): SidebarData | null => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      // Get current data from store (don't trigger load here - that's done in useEffect)
      const nodeData = getNodeData(nodeId);

      // Build maps with current node data for sidebar
      const executionsMap =
        !isViewingLiveVersion || nodeData.executions.length === 0 ? {} : { [nodeId]: nodeData.executions };
      const queueItemsMap =
        !isViewingLiveVersion || nodeData.queueItems.length === 0 ? {} : { [nodeId]: nodeData.queueItems.reverse() };
      const eventsMapForSidebar =
        !isViewingLiveVersion || nodeData.events.length === 0
          ? {}
          : { [nodeId]: nodeData.events.length > 0 ? nodeData.events : visibleNodeEventsMap[nodeId] || [] };
      const totalHistoryCount = !isViewingLiveVersion ? 0 : nodeData.totalInHistoryCount;
      const totalQueueCount = !isViewingLiveVersion ? 0 : nodeData.totalInQueueCount;

      const sidebarData = prepareSidebarData(
        node,
        canvas?.spec?.nodes || [],
        allComponents,
        allTriggers,
        executionsMap,
        queueItemsMap,
        eventsMapForSidebar,
        totalHistoryCount,
        totalQueueCount,
      );

      // Add loading state to sidebar data
      return {
        ...sidebarData,
        isLoading: nodeData.isLoading,
      };
    },
    [canvas, allComponents, allTriggers, visibleNodeEventsMap, isViewingLiveVersion, getNodeData],
  );

  // Trigger data loading when sidebar opens for a node
  const loadSidebarData = useCallback(
    (nodeId: string) => {
      if (!isViewingLiveVersion) {
        return;
      }

      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      // Set current history node for tracking
      setCurrentHistoryNode({ nodeId, nodeType: node?.type || "TYPE_ACTION" });

      loadNodeDataMethod(canvasId!, nodeId, node.type!, queryClient);
    },
    [canvas, canvasId, queryClient, loadNodeDataMethod, isViewingLiveVersion],
  );

  const onCancelQueueItem = useOnCancelQueueItemHandler({
    canvasId: canvasId!,
    organizationId,
    canvas,
    loadSidebarData,
  });

  const [currentHistoryNode, setCurrentHistoryNode] = useState<{ nodeId: string; nodeType: string } | null>(null);
  const [focusRequest, setFocusRequest] = useState<{
    nodeId: string;
    requestId: number;
    tab?: "latest" | "settings" | "execution-chain";
    executionChain?: {
      eventId: string;
      executionId?: string | null;
      triggerEvent?: SidebarEvent | null;
    };
  } | null>(null);
  const [liveRunEntries, setLiveRunEntries] = useState<LogEntry[]>([]);
  const [liveCanvasEntries, setLiveCanvasEntries] = useState<LogEntry[]>([]);
  const [resolvedExecutionIds, setResolvedExecutionIds] = useState<Set<string>>(new Set());
  const handleExecutionChainHandled = useCallback(() => setFocusRequest(null), []);

  const handleSidebarChange = useCallback(
    (open: boolean, nodeId: string | null) => {
      const next = new URLSearchParams(searchParams);
      if (open) {
        next.set("sidebar", "1");
        if (nodeId) {
          next.set("node", nodeId);
        } else {
          next.delete("node");
        }
      } else {
        next.delete("sidebar");
        next.delete("node");
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  const handleLogNodeSelect = useCallback(
    (nodeId: string) => {
      handleSidebarChange(true, nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), tab: "settings" });
    },
    [handleSidebarChange],
  );

  useEffect(() => {
    logNodeSelectRef.current = handleLogNodeSelect;
  }, [handleLogNodeSelect]);

  const handleLogRunNodeSelect = useCallback(
    (nodeId: string) => {
      handleSidebarChange(true, nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), tab: "latest" });
    },
    [handleSidebarChange],
  );

  const handleLogRunExecutionSelect = useCallback(
    (options: { nodeId: string; eventId: string; executionId: string; triggerEvent?: SidebarEvent }) => {
      handleSidebarChange(true, options.nodeId);
      setFocusRequest({
        nodeId: options.nodeId,
        requestId: Date.now(),
        tab: "execution-chain",
        executionChain: {
          eventId: options.eventId,
          executionId: options.executionId,
          triggerEvent: options.triggerEvent,
        },
      });
    },
    [handleSidebarChange],
  );

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

  const handleLogView = useCallback(() => {
    analytics.canvasLogView(organizationId ?? "");
  }, [organizationId]);

  const buildLiveRunItemFromExecution = useCallback(
    (execution: CanvasesCanvasNodeExecution): LogRunItem => {
      return buildRunItemFromExecution({
        execution,
        nodes: canvas?.spec?.nodes || [],
        onNodeSelect: handleLogRunNodeSelect,
        onExecutionSelect: handleLogRunExecutionSelect,
        event: execution.rootEvent || undefined,
      });
    },
    [handleLogRunExecutionSelect, handleLogRunNodeSelect, canvas?.spec?.nodes],
  );

  const buildLiveRunEntryFromEvent = useCallback(
    (event: CanvasesCanvasEvent, runItems: LogRunItem[] = []): LogEntry => {
      return buildRunEntryFromEvent({
        event,
        nodes: canvas?.spec?.nodes || [],
        runItems,
      });
    },
    [canvas?.spec?.nodes],
  );

  const handleWorkflowEventCreated = useCallback(
    (event: CanvasesCanvasEvent) => {
      if (!event.id) {
        return;
      }

      const nodes = canvas?.spec?.nodes || [];
      const node = nodes.find((item) => item.id === event.nodeId);
      if (!node || node.type !== "TYPE_TRIGGER") {
        return;
      }

      setLiveRunEntries((prev) => {
        const entry = buildLiveRunEntryFromEvent(event, []);
        const next = [entry, ...prev.filter((item) => item.id !== entry.id)];
        return next.sort((a, b) => {
          const aTime = Date.parse(a.timestamp || "") || 0;
          const bTime = Date.parse(b.timestamp || "") || 0;
          return bTime - aTime;
        });
      });
    },
    [buildLiveRunEntryFromEvent, canvas?.spec?.nodes],
  );

  const handleExecutionEvent = useCallback(
    (execution: CanvasesCanvasNodeExecution) => {
      if (!execution.rootEvent?.id) {
        return;
      }

      setLiveRunEntries((prev) => {
        const runItem = buildLiveRunItemFromExecution(execution);
        const existing = prev.find((item) => item.id === execution.rootEvent?.id);
        const existingRunItems = existing?.runItems || [];
        const runItemsMap = new Map(existingRunItems.map((item) => [item.id, item]));
        runItemsMap.set(runItem.id, runItem);
        const runItems = Array.from(runItemsMap.values());
        const entry = buildLiveRunEntryFromEvent(execution.rootEvent as CanvasesCanvasEvent, runItems);
        const next = [entry, ...prev.filter((item) => item.id !== entry.id)];
        return next.sort((a, b) => {
          const aTime = Date.parse(a.timestamp || "") || 0;
          const bTime = Date.parse(b.timestamp || "") || 0;
          return bTime - aTime;
        });
      });
    },
    [buildLiveRunEntryFromEvent, buildLiveRunItemFromExecution],
  );

  const invalidateCanvasVersionData = useCallback(
    (targetCanvasId: string, targetVersionId?: string) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(targetCanvasId) });
      if (targetVersionId) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(targetCanvasId, targetVersionId) });
      }
    },
    [queryClient],
  );

  const handleCanvasLifecycleEvent = useCallback(
    (payload: { canvasId: string; versionId?: string }, eventName: string) => {
      if (eventName === "canvas_deleted") {
        setCanvasDeletedRemotely(true);
        return true;
      }

      if (eventName === "canvas_updated" && consumeIgnoredCanvasUpdatedEcho()) {
        return false;
      }

      if (eventName === "canvas_version_updated" && consumeIgnoredCanvasVersionUpdatedEcho(payload.versionId)) {
        return false;
      }

      if (!canvasId) {
        return true;
      }

      if (eventName === "canvas_version_updated") {
        invalidateCanvasVersionData(canvasId);
        if (activeCanvasVersionId && payload.versionId === activeCanvasVersionId) {
          if (hasPendingLocalCanvasState) {
            setRemoteCanvasUpdatePending(true);
            return true;
          }
          invalidateCanvasVersionData(canvasId, activeCanvasVersionId);
        }
        return true;
      }

      if (eventName !== "canvas_updated") {
        return true;
      }

      if (hasPendingLocalCanvasState) {
        setRemoteCanvasUpdatePending(true);
        return true;
      }

      invalidateCanvasVersionData(canvasId, activeCanvasVersionId);
      return true;
    },
    [
      activeCanvasVersionId,
      canvasId,
      consumeIgnoredCanvasUpdatedEcho,
      consumeIgnoredCanvasVersionUpdatedEcho,
      hasPendingLocalCanvasState,
      invalidateCanvasVersionData,
    ],
  );

  const shouldApplyCanvasUpdate = useCallback(
    () => isViewingLiveVersion && !hasPendingLocalCanvasState && !canvasDeletedRemotely,
    [isViewingLiveVersion, hasPendingLocalCanvasState, canvasDeletedRemotely],
  );

  useCanvasWebsocket(
    canvasId!,
    organizationId!,
    handleNodeWebsocketEvent,
    handleWorkflowEventCreated,
    handleExecutionEvent,
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
    isViewingLiveVersion,
    true,
  );

  const logEntries = useMemo(() => {
    const nodes = canvas?.spec?.nodes || [];
    const canvasEntries = mapCanvasNodesToLogEntries({
      nodes,
      workflowUpdatedAt: canvas?.metadata?.updatedAt || "",
      onNodeSelect: handleLogNodeSelect,
    });

    const rootEvents = canvasEventsResponse?.events || [];
    const runEntries = mapWorkflowEventsToRunLogEntries({
      events: rootEvents,
      nodes,
      onNodeSelect: handleLogRunNodeSelect,
      onExecutionSelect: handleLogRunExecutionSelect,
    });
    return mergeWorkflowLogEntries({
      isViewingLiveVersion,
      runEntries,
      liveRunEntries,
      canvasEntries,
      liveCanvasEntries,
      resolvedExecutionIds,
    });
  }, [
    isViewingLiveVersion,
    handleLogNodeSelect,
    handleLogRunNodeSelect,
    handleLogRunExecutionSelect,
    liveCanvasEntries,
    liveRunEntries,
    resolvedExecutionIds,
    canvas?.metadata?.updatedAt,
    canvas?.spec?.nodes,
    canvasEventsResponse?.events,
  ]);

  const nodeHistoryQuery = useNodeHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION",
    allNodes: canvas?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!canvasId && isViewingLiveVersion,
  });

  const queueHistoryQuery = useQueueHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    allNodes: canvas?.spec?.nodes || [],
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
        workflowNodes: canvas?.spec?.nodes || [],
        nodeEventsMap: visibleNodeEventsMap,
        nodeExecutionsMap: visibleNodeExecutionsMap,
        nodeQueueItemsMap: visibleNodeQueueItemsMap,
      });
    },
    [canvas, visibleNodeExecutionsMap, visibleNodeEventsMap, visibleNodeQueueItemsMap],
  );

  const getAutocompleteExampleObj = useCallback(
    (nodeId: string): Record<string, unknown> | null => {
      const workflowNodes = canvas?.spec?.nodes || [];
      const workflowEdges = canvas?.spec?.edges || [];

      const currentNode = workflowNodes.find((node) => node.id === nodeId);
      const chainNodeIds = new Set<string>();

      if (currentNode?.type === "TYPE_TRIGGER") {
        chainNodeIds.add(nodeId);
      }

      const stack = workflowEdges
        .filter((edge) => edge.targetId === nodeId && edge.sourceId)
        .map((edge) => edge.sourceId as string);

      while (stack.length > 0) {
        const nextId = stack.pop();
        if (!nextId || chainNodeIds.has(nextId)) continue;
        chainNodeIds.add(nextId);
        workflowEdges
          .filter((edge) => edge.targetId === nextId && edge.sourceId)
          .forEach((edge) => {
            if (edge.sourceId) {
              stack.push(edge.sourceId);
            }
          });
      }

      if (chainNodeIds.size === 0) {
        return null;
      }

      const exampleObj: Record<string, unknown> = {};
      const nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }> = {};
      const nodeNamesById: Record<string, string> = {};

      chainNodeIds.forEach((chainNodeId) => {
        const chainNode = workflowNodes.find((node) => node.id === chainNodeId);
        if (!chainNode) return;

        const nodeName = (chainNode.name || "").trim();
        if (nodeName) {
          nodeNamesById[chainNodeId] = nodeName;
        }

        if (chainNode.type === "TYPE_TRIGGER") {
          const triggerMetadata = allTriggers.find((trigger) => trigger.name === chainNode.component);

          // Store node metadata with trigger info
          nodeMetadata[chainNodeId] = {
            name: nodeName || undefined,
            componentType: triggerMetadata?.label || "Trigger",
            description: triggerMetadata?.description,
          };

          const latestEvent = visibleNodeEventsMap[chainNodeId]?.[0];
          if (latestEvent?.data) {
            exampleObj[chainNodeId] = { ...(latestEvent.data || {}) } as Record<string, unknown>;
          }
          if (exampleObj[chainNodeId]) {
            return;
          }

          const exampleData = triggerMetadata?.exampleData;
          if (exampleData && typeof exampleData === "object") {
            exampleObj[chainNodeId] = Array.isArray(exampleData)
              ? [...exampleData]
              : ({ ...exampleData } as Record<string, unknown>);
          }
          return;
        }

        // For components (non-triggers)
        const componentMetadata = allComponents.find((component) => component.name === chainNode.component);

        // Store node metadata with component info
        nodeMetadata[chainNodeId] = {
          name: nodeName || undefined,
          componentType: componentMetadata?.label || "Component",
          description: componentMetadata?.description,
        };

        const latestExecution = visibleNodeExecutionsMap[chainNodeId]?.find(
          (execution) => execution.state === "STATE_FINISHED" && execution.resultReason !== "RESULT_REASON_ERROR",
        );
        if (!latestExecution?.outputs) {
          const exampleOutput = componentMetadata?.exampleOutput;
          if (exampleOutput && typeof exampleOutput === "object") {
            exampleObj[chainNodeId] = Array.isArray(exampleOutput)
              ? [...exampleOutput]
              : ({ ...exampleOutput } as Record<string, unknown>);
          }
          return;
        }

        const outputData: unknown[] = Object.values(latestExecution.outputs)?.find((output) => {
          return Array.isArray(output) && output.length > 0;
        }) as unknown[];

        if (outputData?.length > 0) {
          exampleObj[chainNodeId] = { ...(outputData?.[0] || {}) } as Record<string, unknown>;
          return;
        }

        const exampleOutput = componentMetadata?.exampleOutput;
        if (exampleOutput && typeof exampleOutput === "object" && Object.keys(exampleOutput).length > 0) {
          exampleObj[chainNodeId] = { ...exampleOutput } as Record<string, unknown>;
        }
      });

      // Inject config key into component nodes' example objects for autocomplete
      chainNodeIds.forEach((chainNodeId) => {
        const chainNode = workflowNodes.find((node) => node.id === chainNodeId);
        if (!chainNode || chainNode.type !== "TYPE_ACTION") return;

        const obj = exampleObj[chainNodeId];
        if (!obj || typeof obj !== "object" || Array.isArray(obj)) return;

        const latestExecution = visibleNodeExecutionsMap[chainNodeId]?.find(
          (execution) => execution.state === "STATE_FINISHED" && execution.resultReason !== "RESULT_REASON_ERROR",
        );
        if ("config" in (obj as Record<string, unknown>)) return;

        const configData = latestExecution?.configuration || chainNode.configuration;
        if (configData && typeof configData === "object" && Object.keys(configData).length > 0) {
          (obj as Record<string, unknown>).config = configData;
        }
      });

      const getIncomingNodes = (targetId: string): string[] => {
        return workflowEdges
          .filter((edge) => edge.targetId === targetId && edge.sourceId)
          .map((edge) => edge.sourceId as string);
      };

      const previousByDepth: Record<string, unknown> = {};
      let frontier = [nodeId];
      const visited = new Set<string>([nodeId]);
      let depth = 0;

      while (frontier.length > 0) {
        const next: string[] = [];
        frontier.forEach((current) => {
          getIncomingNodes(current).forEach((sourceId) => {
            if (visited.has(sourceId)) return;
            visited.add(sourceId);
            next.push(sourceId);
          });
        });

        if (next.length === 0) {
          break;
        }

        depth += 1;
        const firstAtDepth = next[0];
        if (firstAtDepth && exampleObj[firstAtDepth]) {
          previousByDepth[String(depth)] = exampleObj[firstAtDepth];
        }

        frontier = next;
      }

      const rootNodeId = workflowNodes.find((node) => {
        if (!node.id || !chainNodeIds.has(node.id)) return false;
        return !workflowEdges.some(
          (edge) => edge.targetId === node.id && edge.sourceId && chainNodeIds.has(edge.sourceId as string),
        );
      })?.id;

      if (rootNodeId && exampleObj[rootNodeId]) {
        exampleObj.__root = exampleObj[rootNodeId];
      }

      if (Object.keys(previousByDepth).length > 0) {
        exampleObj.__previousByDepth = previousByDepth;
      }

      // Build name -> nodeId map, keeping the FIRST (closest) node when names are duplicated
      // chainNodeIds is ordered from closest to farthest, so the first occurrence wins
      const nameToNodeId = new Map<string, string>();
      for (const [nId, nodeName] of Object.entries(nodeNamesById)) {
        if (!nodeName || nodeName === "__nodeNames") {
          continue;
        }

        // Only add if we haven't seen this name yet (keep the closest one)
        if (!nameToNodeId.has(nodeName)) {
          nameToNodeId.set(nodeName, nId);
        }
      }

      const namedExampleObj: Record<string, unknown> = {};
      for (const [nodeName, nodeId] of nameToNodeId.entries()) {
        if (nodeName === nodeId || namedExampleObj[nodeName] !== undefined) {
          continue;
        }

        const value = exampleObj[nodeId];
        if (value === undefined) {
          continue;
        }

        namedExampleObj[nodeName] = value;
      }

      if (Object.keys(namedExampleObj).length === 0) {
        return null;
      }

      if (exampleObj.__root) {
        namedExampleObj.__root = exampleObj.__root;
      }

      if (exampleObj.__previousByDepth) {
        namedExampleObj.__previousByDepth = exampleObj.__previousByDepth;
      }

      // Remove the current node from suggestions - you can't reference your own output
      const currentNodeName = currentNode?.name?.trim();
      const currentNodeId = currentNode?.id;
      if (currentNodeName) {
        delete namedExampleObj[currentNodeName];
      }
      if (currentNodeId) {
        delete nodeMetadata[currentNodeId];
      }

      if (Object.keys(nodeMetadata).length > 0) {
        namedExampleObj.__nodeNames = nodeMetadata;
        Object.entries(nodeMetadata).forEach(([, metadata]) => {
          const value = namedExampleObj[metadata.name ?? ""];
          if (value && typeof value === "object" && !Array.isArray(value)) {
            if (metadata.name) {
              (value as Record<string, unknown>).__nodeName = metadata.name;
            }
          }
        });
      }

      return namedExampleObj;
    },
    [canvas, visibleNodeExecutionsMap, visibleNodeEventsMap, allComponents, allTriggers],
  );

  const handleSaveWorkflow = useCallback(
    async (workflowToSave?: CanvasesCanvas, options?: { showToast?: boolean }) => {
      const targetWorkflow = workflowToSave || canvasRef.current;
      if (!targetWorkflow || !organizationId || !canvasId) return;
      if (!canUpdateCanvas) {
        if (options?.showToast !== false) {
          showErrorToast("You don't have permission to update this canvas");
        }
        return;
      }
      if (isTemplate) {
        if (options?.showToast !== false) {
          showErrorToast("Template canvases are read-only");
        }
        return;
      }
      if (!activeCanvasVersionId) {
        if (options?.showToast !== false) {
          showErrorToast("Enable edit mode before saving changes");
        }
        return;
      }
      const shouldRestoreFocus = options?.showToast === false;
      const focusedNoteId = shouldRestoreFocus ? getActiveNoteId() : null;

      try {
        const savingVersionID = activeCanvasVersionId || undefined;
        const result = await enqueueCanvasSave(targetWorkflow, savingVersionID);
        if (result.status !== "saved") {
          return result;
        }

        const changeSummary = summarizeWorkflowChanges({
          before: lastSavedWorkflowRef.current,
          after: targetWorkflow,
          onNodeSelect: handleLogNodeSelect,
        });
        const changeMessage = changeSummary.changeCount
          ? `${changeSummary.changeCount} Canvas changes saved`
          : "Canvas changes saved";

        if (result.response?.data?.version && savingVersionID && activeCanvasVersionIdRef.current === savingVersionID) {
          setActiveCanvasVersion(result.response.data.version);
        }
        if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
          return result;
        }

        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-${Date.now()}`,
            message: changeMessage,
            type: "success",
            timestamp: new Date().toISOString(),
            detail: changeSummary.detail,
            searchText: changeSummary.searchText,
          }),
          ...prev,
        ]);
        if (options?.showToast !== false) {
          showSuccessToast("Canvas changes saved");
        }
        setLastSavedWorkflowSnapshot(targetWorkflow);

        if (result.matchesCurrentCanvas && !result.hasQueuedFollowUp) {
          setHasUnsavedChanges(false);
          setHasNonPositionalUnsavedChanges(false);
        }

        return result;
      } catch (error: any) {
        const errorMessage = getApiErrorMessage(error, "Failed to save changes to the canvas");
        const displayMessage = getUsageLimitToastMessage(error, errorMessage);
        showErrorToast(displayMessage);
        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-error-${Date.now()}`,
            message: errorMessage,
            type: "error",
            timestamp: new Date().toISOString(),
          }),
          ...prev,
        ]);
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
      isTemplate,
      canUpdateCanvas,
      enqueueCanvasSave,
      handleLogNodeSelect,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const getNodeEditData = useCallback(
    (nodeId: string): NodeEditData | null => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      // Get configuration fields from metadata based on node type
      let configurationFields: SuperplaneActionsAction["configuration"] = [];
      let displayLabel: string | undefined = node.name || undefined;
      let integrationName: string | undefined;
      let integrationLabel: string | undefined;
      let blockName: string | undefined;

      if (node.type === "TYPE_ACTION") {
        const componentMetadata = allComponents.find((c) => c.name === node.component);
        configurationFields = componentMetadata?.configuration || [];
        displayLabel = componentMetadata?.label || displayLabel;
        blockName = node.component;
        integrationName = getNodeIntegrationName(node, availableIntegrations);
        integrationLabel = integrationName
          ? availableIntegrations.find((i) => i.name === integrationName)?.label
          : undefined;
      } else if (node.type === "TYPE_TRIGGER") {
        const triggerMetadata = allTriggers.find((t) => t.name === node.component);
        configurationFields = triggerMetadata?.configuration || [];
        displayLabel = triggerMetadata?.label || displayLabel;
        blockName = node.component;
        integrationName = getNodeIntegrationName(node, availableIntegrations);
        integrationLabel = integrationName
          ? availableIntegrations.find((i) => i.name === integrationName)?.label
          : undefined;
      } else if (node.type === "TYPE_WIDGET") {
        const widget = widgets.find((w) => w.name === node.component);
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
      };
    },
    [canvas, allComponents, allTriggers, availableIntegrations, widgets],
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
    return integrations.find((i) => i.spec?.integrationName === integrationDialogName && i.status?.state !== "ready");
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
    if (!canvas?.spec?.nodes || !canReadIntegrations) return [];

    const missingMap = new Map<
      string,
      {
        count: number;
        definition?: (typeof availableIntegrations)[0];
        state?: "pending" | "error";
        stateDescription?: string;
      }
    >();

    for (const node of canvas.spec.nodes) {
      const integrationName = getNodeIntegrationName(node, availableIntegrations);
      if (!integrationName) continue;

      const hasReadyInstance = integrations.some(
        (i) => i.spec?.integrationName === integrationName && i.status?.state === "ready",
      );
      if (hasReadyInstance) continue;

      const existing = missingMap.get(integrationName);
      if (existing) {
        existing.count++;
      } else {
        const nonReadyInstance = integrations.find(
          (i) => i.spec?.integrationName === integrationName && i.status?.state !== "ready",
        );
        const rawState = nonReadyInstance?.status?.state;
        missingMap.set(integrationName, {
          count: 1,
          definition: availableIntegrations.find((d) => d.name === integrationName),
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
  }, [canvas?.spec?.nodes, availableIntegrations, integrations, canReadIntegrations, justConnectedIntegrations]);

  const handleConnectIntegration = useCallback((integrationName: string) => {
    setIntegrationDialogName(integrationName);
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
    ) => {
      if (!canvas || !organizationId || !canvasId) return;

      const configuringNode = canvas.spec?.nodes?.find((n) => n.id === nodeId);
      if (configuringNode) {
        const fieldCount = Object.values(updatedConfiguration).filter(
          (v) => v !== null && v !== undefined && v !== "",
        ).length;
        const { nodeType, integration } = getNodeAnalyticsProps(configuringNode, availableIntegrations);
        analytics.nodeConfigure(nodeType, integration, fieldCount, organizationId);
      }

      // Save snapshot before making changes

      // Update the node's configuration, name, and app installation ref in local cache only
      const updatedNodes = canvas?.spec?.nodes?.map((node) => {
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
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate, availableIntegrations],
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

          if (hasNonPositionalUnsavedChanges) {
            return;
          }

          const latestWorkflow = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));

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
    [organizationId, canvasId, queryClient, handleSaveWorkflow, hasNonPositionalUnsavedChanges, isReadOnly],
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

  const getCurrentWorkflowSnapshot = useCallback(() => {
    if (!organizationId || !canvasId) {
      return canvasRef.current;
    }

    return queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvasRef.current;
  }, [organizationId, canvasId, queryClient]);

  const hasPendingLocalDraftChanges = useCallback(() => {
    if (!activeCanvasVersionIdRef.current) {
      return false;
    }

    const currentWorkflow = getCurrentWorkflowSnapshot();

    if (!currentWorkflow || !lastSavedWorkflowSignatureRef.current) {
      return hasUnsavedChanges;
    }

    return getWorkflowSaveSignature(currentWorkflow) !== lastSavedWorkflowSignatureRef.current;
  }, [getCurrentWorkflowSnapshot, hasUnsavedChanges]);

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

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;

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
      queryClient,
      debouncedAnnotationAutoSave,
      debouncedAutoSave,
      isReadOnly,
      applyLocalWorkflowUpdate,
    ],
  );

  const handleNodeAdd = useCallback(
    async (newNodeData: NewNodeData): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;
      if (!latestWorkflow) return "";

      // Save snapshot before making changes

      const { buildingBlock, configuration, position, sourceConnection, integrationRef } = newNodeData;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, buildingBlock.configuration || []);

      // Get existing node names for unique name generation
      const existingNodeNames = (latestWorkflow.spec?.nodes || []).map((n) => n.name || "").filter(Boolean);

      // Generate unique node name based on component name + ordinal
      const nameBase = newNodeData.nodeName || buildingBlock.name || "node";
      const uniqueNodeName = generateUniqueNodeName(nameBase, existingNodeNames);

      // Generate a unique node ID
      const newNodeId = generateNodeId(buildingBlock.name || "node", uniqueNodeName);

      // Create the new node
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
              x: (latestWorkflow?.spec?.nodes?.length || 0) * 250,
              y: 100,
            },
      };

      // Add type-specific component reference
      if (buildingBlock.name === "annotation") {
        // Annotation nodes are now widgets
        newNode.component = "annotation";
        newNode.configuration = { text: "", color: "yellow" };
      } else if (buildingBlock.type === "component") {
        newNode.component = buildingBlock.name;
      } else if (buildingBlock.type === "trigger") {
        newNode.component = buildingBlock.name;
      }

      // Track node addition
      const { nodeType, integration, nodeRef } = getNodeAnalyticsProps(newNode, availableIntegrations);
      analytics.nodeAdd(nodeType, integration, nodeRef, organizationId);

      // Add the new node to the workflow
      const updatedNodes = [...(latestWorkflow.spec?.nodes || []), newNode];

      // If there's a source connection, create the edge
      let updatedEdges = latestWorkflow.spec?.edges || [];
      if (sourceConnection) {
        const newEdge: ComponentsEdge = {
          sourceId: sourceConnection.nodeId,
          targetId: newNodeId,
          channel: sourceConnection.handleId || "default",
        };
        updatedEdges = [...updatedEdges, newEdge];
      }

      const updatedWorkflow = {
        ...latestWorkflow,
        spec: {
          ...latestWorkflow.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      const finalWorkflow = await applyAutoLayoutOnAddedNode(updatedWorkflow, newNodeId);

      // Update local cache
      applyLocalWorkflowUpdate(finalWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(finalWorkflow, { showToast: false });
      }

      // Return the new node ID
      return newNodeId;
    },
    [
      canvas,
      organizationId,
      canvasId,
      handleSaveWorkflow,
      applyAutoLayoutOnAddedNode,
      isReadOnly,
      applyLocalWorkflowUpdate,
      availableIntegrations,
    ],
  );

  const handleApplyAiOperations = useCallback(
    async (operations: CanvasChangesetChange[]) => {
      if (!operations.length || !organizationId || !canvasId) {
        return;
      }

      const versionId = activeCanvasVersionIdRef.current || activeCanvasVersionId;
      if (!versionId) {
        throw new Error("Enable edit mode before applying AI changes.");
      }

      const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho(versionId);
      const releaseCanvasUpdatedEcho = registerIgnoredCanvasUpdatedEcho();
      const autoLayoutNodeIds = Array.from(
        new Set(
          operations
            .filter((operation) => operation.type === "ADD_NODE")
            .map((operation) => operation.node?.id)
            .filter((id): id is string => Boolean(id)),
        ),
      );

      try {
        const response = await canvasesApplyCanvasVersionChangeset(
          withOrganizationHeader({
            path: {
              canvasId,
              versionId,
            },
            body: {
              changeset: {
                changes: operations,
              },
              ...(autoLayoutNodeIds.length > 0
                ? {
                    autoLayout: {
                      algorithm: "ALGORITHM_HORIZONTAL",
                      scope: "SCOPE_CONNECTED_COMPONENT",
                      nodeIds: autoLayoutNodeIds,
                    },
                  }
                : {}),
            },
          }),
        );

        const version = response.data?.version;
        if (!version) {
          throw new Error("Failed to apply AI changes.");
        }

        queryClient.setQueryData(canvasKeys.versionDetail(canvasId, versionId), version);
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

          next.sort(
            (left, right) =>
              versionSortValue(right.metadata?.publishedAt || right.metadata?.updatedAt || right.metadata?.createdAt) -
              versionSortValue(left.metadata?.publishedAt || left.metadata?.updatedAt || left.metadata?.createdAt),
          );
          return next;
        });

        queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
          if (!current || !version.spec) {
            return current;
          }

          return {
            ...current,
            spec: {
              ...current.spec,
              ...version.spec,
            },
          };
        });

        setActiveCanvasVersion(version);
        setLastSavedWorkflowSnapshot(
          queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) ?? null,
        );
        setHasUnsavedChanges(false);
        setHasNonPositionalUnsavedChanges(false);
      } catch (error) {
        releaseCanvasUpdatedEcho();
        releaseCanvasVersionUpdatedEcho();
        throw error;
      }
    },
    [
      activeCanvasVersionId,
      canvasId,
      organizationId,
      queryClient,
      registerIgnoredCanvasUpdatedEcho,
      registerIgnoredCanvasVersionUpdatedEcho,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const handlePlaceholderAdd = useCallback(
    async (data: {
      position: { x: number; y: number };
      sourceNodeId: string;
      sourceHandleId: string | null;
    }): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;
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

      const newEdge: ComponentsEdge = {
        sourceId: data.sourceNodeId,
        targetId: newNodeId,
        channel: data.sourceHandleId || "default",
      };

      const updatedWorkflow = {
        ...latestWorkflow,
        spec: {
          ...latestWorkflow.spec,
          nodes: [...(latestWorkflow.spec?.nodes || []), newNode],
          edges: [...(latestWorkflow.spec?.edges || []), newEdge],
        },
      };

      const finalWorkflow = await applyAutoLayoutOnAddedNode(updatedWorkflow, newNodeId);

      applyLocalWorkflowUpdate(finalWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(finalWorkflow, { showToast: false });
      }

      return newNodeId;
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      handleSaveWorkflow,
      applyAutoLayoutOnAddedNode,
      isReadOnly,
      applyLocalWorkflowUpdate,
    ],
  );

  const handlePlaceholderConfigure = useCallback(
    async (data: {
      placeholderId: string;
      buildingBlock: any;
      nodeName: string;
      configuration: Record<string, any>;
      appName?: string;
    }): Promise<void> => {
      if (!canvas || !organizationId || !canvasId) {
        return;
      }

      const nodeIndex = canvas.spec?.nodes?.findIndex((n) => n.id === data.placeholderId);
      if (nodeIndex === undefined || nodeIndex === -1) {
        return;
      }

      const filteredConfiguration = filterVisibleConfiguration(
        data.configuration,
        data.buildingBlock.configuration || [],
      );

      // Get existing node names for unique name generation (exclude the placeholder being configured)
      const existingNodeNames = (canvas.spec?.nodes || [])
        .filter((n) => n.id !== data.placeholderId)
        .map((n) => n.name || "")
        .filter(Boolean);

      // Generate unique node name based on component name + ordinal
      const uniqueNodeName = generateUniqueNodeName(data.buildingBlock.name || "node", existingNodeNames);

      // Update placeholder with real component data
      const updatedNode: ComponentsNode = {
        ...canvas.spec!.nodes![nodeIndex],
        name: uniqueNodeName,
        type: data.buildingBlock.type === "trigger" ? "TYPE_TRIGGER" : "TYPE_ACTION",
        configuration: filteredConfiguration,
      };

      // Add the component reference that was missing
      if (data.buildingBlock.type === "component") {
        updatedNode.component = data.buildingBlock.name;
      } else if (data.buildingBlock.type === "trigger") {
        updatedNode.component = data.buildingBlock.name;
      }

      const updatedNodes = [...(canvas.spec?.nodes || [])];
      updatedNodes[nodeIndex] = updatedNode;

      // Update outgoing edges from this node to use valid channels
      // Find edges where this node is the source
      const outgoingEdges = canvas.spec?.edges?.filter((edge) => edge.sourceId === data.placeholderId) || [];

      let updatedEdges = [...(canvas.spec?.edges || [])];

      if (outgoingEdges.length > 0) {
        // Get the valid output channels for the new component
        const validChannels = data.buildingBlock.outputChannels?.map((ch: any) => ch.name).filter(Boolean) || [
          "default",
        ];

        // Update each outgoing edge to use a valid channel
        updatedEdges = updatedEdges.map((edge) => {
          if (edge.sourceId === data.placeholderId) {
            // If the current channel is not valid for the new component, use the first valid channel
            const newChannel = validChannels.includes(edge.channel) ? edge.channel : validChannels[0];
            return {
              ...edge,
              channel: newChannel,
            };
          }
          return edge;
        });
      }

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
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

      // Add the new edge to the workflow
      const updatedEdges = [...(canvas.spec?.edges || []), newEdge];

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
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
      const updatedNodes = specNodes.filter((node) => node.id !== nodeId);
      const survivingNodeIds = new Set(updatedNodes.map((node) => node.id).filter(Boolean));

      const updatedEdges = canvas.spec?.edges?.filter(
        (edge) =>
          (!edge.sourceId || survivingNodeIds.has(edge.sourceId)) &&
          (!edge.targetId || survivingNodeIds.has(edge.targetId)),
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      // Update local cache
      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate, availableIntegrations],
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
      const updatedNodes = specNodes.filter((node) => !node.id || !nodeIdSet.has(node.id));
      const survivingNodeIds = new Set(updatedNodes.map((node) => node.id).filter(Boolean));
      const updatedEdges = canvas.spec?.edges?.filter(
        (edge) =>
          (!edge.sourceId || survivingNodeIds.has(edge.sourceId)) &&
          (!edge.targetId || survivingNodeIds.has(edge.targetId)),
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate, availableIntegrations],
  );
  const handleAutoLayoutNodes = useCallback(
    async (nodeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      const updatedWorkflow = await DefaultLayoutEngine.apply(canvas, {
        nodeIds,
        scope: "connected-component",
        components,
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

      const specNodes = canvas.spec?.nodes || [];

      const existingNodeNames = specNodes.map((n) => n.name || "").filter(Boolean);
      const newNodes: ComponentsNode[] = [];
      const nodeIdMap: Record<string, string> = {};

      for (const nodeId of nodeIds) {
        const nodeToDuplicate = specNodes.find((node) => node.id === nodeId);
        if (!nodeToDuplicate) continue;

        let baseName = nodeToDuplicate.name?.trim() || "";
        if (!baseName) {
          if (nodeToDuplicate.type === "TYPE_TRIGGER" && nodeToDuplicate.component) {
            baseName = nodeToDuplicate.component;
          } else if (nodeToDuplicate.type === "TYPE_ACTION" && nodeToDuplicate.component) {
            baseName = nodeToDuplicate.component;
          } else {
            baseName = "node";
          }
        }

        const allNames = [...existingNodeNames, ...newNodes.map((n) => n.name || "")];
        const uniqueNodeName = generateUniqueNodeName(baseName, allNames);
        const newNodeId = generateNodeId(baseName, uniqueNodeName);

        nodeIdMap[nodeId] = newNodeId;

        newNodes.push({
          ...nodeToDuplicate,
          id: newNodeId,
          name: uniqueNodeName,
          position: {
            x: (nodeToDuplicate.position?.x || 0) + 50,
            y: (nodeToDuplicate.position?.y || 0) + 50,
          },
          isCollapsed: false,
        });
      }

      if (newNodes.length === 0) return;

      const duplicatedNodeIds = new Set(nodeIds);
      const newEdges = (canvas.spec?.edges || [])
        .filter(
          (edge) =>
            edge.sourceId != null &&
            edge.targetId != null &&
            duplicatedNodeIds.has(edge.sourceId) &&
            duplicatedNodeIds.has(edge.targetId),
        )
        .map((edge) => ({
          ...edge,
          sourceId: nodeIdMap[edge.sourceId!],
          targetId: nodeIdMap[edge.targetId!],
        }));

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: [...(canvas.spec?.nodes || []), ...newNodes],
          edges: [...(canvas.spec?.edges || []), ...newEdges],
        },
      };

      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
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

      // Remove the edges from the workflow
      const updatedEdges = canvas.spec?.edges?.filter((edge) => {
        return !edgesToRemove.some(
          (toRemove) =>
            edge.sourceId === toRemove.sourceId &&
            edge.targetId === toRemove.targetId &&
            edge.channel === toRemove.channel,
        );
      });

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      applyLocalWorkflowUpdate(updatedWorkflow);

      if (!isReadOnly) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      }
    },
    [canvas, organizationId, canvasId, handleSaveWorkflow, isReadOnly, applyLocalWorkflowUpdate],
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
      if (!canvas || !organizationId || !canvasId) return;

      const roundedPosition = {
        x: Math.round(position.x),
        y: Math.round(position.y),
      };

      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              position: roundedPosition,
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
        pendingPositionUpdatesRef.current.set(nodeId, roundedPosition);
        setIsPositionAutoSaveQueued(true);
        debouncedAutoSave();
      }
    },
    [canvas, organizationId, canvasId, debouncedAutoSave, isReadOnly, applyLocalWorkflowUpdate],
  );

  const handleNodesPositionChange = useCallback(
    (updates: Array<{ nodeId: string; position: { x: number; y: number } }>) => {
      if (!canvas || !organizationId || !canvasId || updates.length === 0) return;

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

      // Update all nodes in a single operation
      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id && positionMap.has(node.id)
          ? {
              ...node,
              position: positionMap.get(node.id)!,
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
        // Add all position updates to pending updates
        positionMap.forEach((position, nodeId) => {
          pendingPositionUpdatesRef.current.set(nodeId, position);
        });
        setIsPositionAutoSaveQueued(true);
        debouncedAutoSave();
      }
    },
    [canvas, organizationId, canvasId, debouncedAutoSave, isReadOnly, applyLocalWorkflowUpdate],
  );

  const handleNodeCollapseChange = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes

      // Find the current node to determine its collapsed state
      const currentNode = canvas.spec?.nodes?.find((node) => node.id === nodeId);
      if (!currentNode) return;

      // Toggle the collapsed state
      const newIsCollapsed = !currentNode.isCollapsed;

      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              isCollapsed: newIsCollapsed,
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

  const handleRun = useCallback(
    async (nodeId: string, channel: string, data: any) => {
      if (!canvasId) return;

      try {
        await canvasesEmitNodeEvent(
          withOrganizationHeader({
            path: {
              canvasId: canvasId,
              nodeId: nodeId,
            },
            body: {
              channel,
              data,
            },
          }),
        );
        // Note: Success toast is shown by EmitEventModal
        const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
        if (node && organizationId) {
          const { nodeType, integration } = getNodeAnalyticsProps(node, availableIntegrations);
          analytics.eventEmit(nodeType, integration, organizationId);
        }
      } catch (error) {
        showErrorToast("Failed to emit event");
        throw error; // Re-throw to let EmitEventModal handle it
      }
    },
    [canvasId, canvas, availableIntegrations, organizationId],
  );

  const handleTogglePause = useCallback(
    async (nodeId: string) => {
      if (!canvasId || !organizationId || !canvas) return;

      const node = canvas.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      if (node.type === "TYPE_TRIGGER") {
        showErrorToast("Triggers cannot be paused");
        return;
      }

      const nextPaused = !node.paused;

      try {
        const result = await canvasesUpdateNodePause(
          withOrganizationHeader({
            path: {
              canvasId: canvasId,
              nodeId: nodeId,
            },
            body: {
              paused: nextPaused,
            },
          }),
        );

        const updatedPaused = result.data?.node?.paused ?? nextPaused;
        const updatedNodes = (canvas.spec?.nodes || []).map((item) =>
          item.id === nodeId ? { ...item, paused: updatedPaused } : item,
        );

        const updatedWorkflow = {
          ...canvas,
          spec: {
            ...canvas.spec,
            nodes: updatedNodes,
          },
        };

        applyLocalWorkflowUpdate(updatedWorkflow);
        showSuccessToast(updatedPaused ? "Component paused" : "Component resumed");
      } catch (error) {
        const parsedError = error as { message: string };
        if (parsedError?.message) {
          showErrorToast(parsedError.message);
        } else {
          console.error("Failed to update node pause state:", error);
        }
      }
    },
    [canvasId, organizationId, canvas, applyLocalWorkflowUpdate],
  );

  const handleReEmit = useCallback(
    async (nodeId: string, eventOrExecutionId: string) => {
      const nodeEvents = visibleNodeEventsMap[nodeId];
      if (!nodeEvents) return;
      const eventToReemit = nodeEvents.find((event) => event.id === eventOrExecutionId);
      if (!eventToReemit) return;
      handleRun(nodeId, eventToReemit.channel || "", eventToReemit.data);
    },
    [handleRun, visibleNodeEventsMap],
  );

  const handleNodeDuplicate = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      const nodeToDuplicate = canvas.spec?.nodes?.find((node) => node.id === nodeId);
      if (!nodeToDuplicate) return;

      const existingNodeNames = (canvas.spec?.nodes || []).map((n) => n.name || "").filter(Boolean);

      let baseName = nodeToDuplicate.name?.trim() || "";
      if (!baseName) {
        if (nodeToDuplicate.type === "TYPE_TRIGGER" && nodeToDuplicate.component) {
          baseName = nodeToDuplicate.component;
        } else if (nodeToDuplicate.type === "TYPE_ACTION" && nodeToDuplicate.component) {
          baseName = nodeToDuplicate.component;
        } else {
          baseName = "node";
        }
      }

      // Generate unique node name based on the existing node name + ordinal
      const uniqueNodeName = generateUniqueNodeName(baseName, existingNodeNames);

      const newNodeId = generateNodeId(baseName, uniqueNodeName);

      const offsetX = 50;
      const offsetY = 50;

      const duplicateNode: ComponentsNode = {
        ...nodeToDuplicate,
        id: newNodeId,
        name: uniqueNodeName,
        position: {
          x: (nodeToDuplicate.position?.x || 0) + offsetX,
          y: (nodeToDuplicate.position?.y || 0) + offsetY,
        },
        // Reset collapsed state for the duplicate
        isCollapsed: false,
      };

      // Add the duplicate node to the workflow
      const updatedNodes = [...(canvas.spec?.nodes || []), duplicateNode];

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      const finalWorkflow = await applyAutoLayoutOnAddedNode(updatedWorkflow, newNodeId);

      // Update local cache
      applyLocalWorkflowUpdate(finalWorkflow);
      if (!isReadOnly) {
        await handleSaveWorkflow(finalWorkflow, { showToast: false });
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      handleSaveWorkflow,
      applyAutoLayoutOnAddedNode,
      isReadOnly,
      applyLocalWorkflowUpdate,
    ],
  );

  const handleSave = useCallback(
    async (canvasNodes: CanvasNode[]) => {
      if (!canvas || !organizationId || !canvasId) return;
      if (isTemplate) {
        showErrorToast("Template canvases are read-only");
        return;
      }
      if (!activeCanvasVersionId) {
        showErrorToast("Enable edit mode before saving changes");
        return;
      }

      // Map canvas nodes back to ComponentsNode format with updated positions
      const updatedNodes = canvas.spec?.nodes?.map((node) => {
        const canvasNode = canvasNodes.find((cn) => cn.id === node.id);
        const componentType = (canvasNode?.data?.type as string) || "";
        if (canvasNode) {
          return {
            ...node,
            position: {
              x: Math.round(canvasNode.position.x),
              y: Math.round(canvasNode.position.y),
            },
            isCollapsed: (canvasNode.data[componentType] as { collapsed: boolean })?.collapsed || false,
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

      const changeSummary = summarizeWorkflowChanges({
        before: lastSavedWorkflowRef.current,
        after: updatedWorkflow,
        onNodeSelect: handleLogNodeSelect,
      });
      const changeMessage = changeSummary.changeCount
        ? `${changeSummary.changeCount} Canvas changes saved`
        : "Canvas changes saved";

      try {
        const savingVersionID = activeCanvasVersionId || undefined;
        const result = await enqueueCanvasSave(updatedWorkflow, savingVersionID);
        if (result.status !== "saved") {
          return;
        }
        if (result.response?.data?.version && savingVersionID && activeCanvasVersionIdRef.current === savingVersionID) {
          setActiveCanvasVersion(result.response.data.version);
        }
        if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
          return;
        }

        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-${Date.now()}`,
            message: changeMessage,
            type: "success",
            timestamp: new Date().toISOString(),
            detail: changeSummary.detail,
            searchText: changeSummary.searchText,
          }),
          ...prev,
        ]);
        showSuccessToast("Canvas changes saved");
        setLastSavedWorkflowSnapshot(updatedWorkflow);

        if (result.matchesCurrentCanvas && !result.hasQueuedFollowUp) {
          setHasUnsavedChanges(false);
          setHasNonPositionalUnsavedChanges(false);
        }
      } catch (error) {
        const errorMessage = getApiErrorMessage(error, "Failed to save changes to the canvas");
        showErrorToast(errorMessage);
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      activeCanvasVersionId,
      isTemplate,
      enqueueCanvasSave,
      setLastSavedWorkflowSnapshot,
      handleLogNodeSelect,
    ],
  );

  const handleCreateChangeRequest = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    const editVersionID = createChangeRequestVersion?.metadata?.id || "";

    if (!editVersionID) {
      showErrorToast("Enable edit mode before opening a change request");
      return;
    }

    setIsPreparingVersionAction(true);
    try {
      const isReady = await ensureVersionActionDraftReady(
        "Unable to prepare the latest version changes for a change request",
      );
      if (!isReady) {
        return;
      }

      if (activeCanvasVersionId !== editVersionID && createChangeRequestVersion) {
        setActiveCanvasVersion(createChangeRequestVersion);
        setSearchParams((current) => {
          const next = new URLSearchParams(current);
          next.set("version", editVersionID);
          return next;
        });
      }

      setSelectedChangeRequestId("");
      setIsCreateChangeRequestMode(true);
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    organizationId,
    canvasId,
    activeCanvasVersionId,
    createChangeRequestVersion,
    ensureVersionActionDraftReady,
    setSearchParams,
  ]);

  const refreshLatestLiveCanvasData = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: canvasKeys.detail(organizationId, canvasId),
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: canvasKeys.versionList(canvasId),
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: canvasKeys.versionHistory(canvasId),
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: canvasKeys.changeRequestList(canvasId),
        refetchType: "all",
      }),
    ]);
  }, [organizationId, canvasId, queryClient]);

  const handlePublishVersion = useCallback(async () => {
    if (!organizationId || !canvasId || !activeCanvasVersionId) {
      return;
    }

    setIsPreparingVersionAction(true);
    try {
      const isReady = await ensureVersionActionDraftReady(
        "Unable to prepare the latest version changes for publishing",
      );
      if (!isReady) {
        return;
      }

      const versionIdToPublish = activeCanvasVersionIdRef.current;
      if (!versionIdToPublish) {
        return;
      }

      await publishCanvasVersionMutation.mutateAsync(versionIdToPublish);
      activeCanvasVersionIdRef.current = "";
      setActiveCanvasVersion(null);
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        next.delete("version");
        return next;
      });
      await refreshLatestLiveCanvasData();
      showSuccessToast("Version published");
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to publish version")));
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    organizationId,
    canvasId,
    activeCanvasVersionId,
    ensureVersionActionDraftReady,
    publishCanvasVersionMutation,
    refreshLatestLiveCanvasData,
    setSearchParams,
  ]);

  const handleActOnChangeRequest = useCallback(
    async ({
      changeRequestId,
      action,
      successMessage,
      fallbackErrorMessage,
      onSuccess,
    }: {
      changeRequestId: string;
      action: ChangeRequestAction;
      successMessage: string;
      fallbackErrorMessage: string;
      onSuccess?: (actedChangeRequestId: string) => void | Promise<void>;
    }) => {
      if (!organizationId || !canvasId || !changeRequestId) {
        return;
      }

      try {
        const response = await actOnCanvasChangeRequestMutation.mutateAsync({
          changeRequestId,
          action,
        });

        const actedChangeRequestId = response?.data?.changeRequest?.metadata?.id || changeRequestId;
        setSelectedChangeRequestId(actedChangeRequestId);
        await onSuccess?.(actedChangeRequestId);
        showSuccessToast(successMessage);
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, fallbackErrorMessage)));
      }
    },
    [organizationId, canvasId, actOnCanvasChangeRequestMutation],
  );

  const handleApproveChangeRequest = useCallback(
    async (changeRequestId: string) => {
      await handleActOnChangeRequest({
        changeRequestId,
        action: "ACTION_APPROVE",
        successMessage: "Change request approved",
        fallbackErrorMessage: "Failed to approve",
      });
    },
    [handleActOnChangeRequest],
  );

  const handleUnapproveChangeRequest = useCallback(
    async (changeRequestId: string) => {
      await handleActOnChangeRequest({
        changeRequestId,
        action: "ACTION_UNAPPROVE",
        successMessage: "Approval removed",
        fallbackErrorMessage: "Failed to unapprove",
      });
    },
    [handleActOnChangeRequest],
  );

  const handlePublishChangeRequest = useCallback(
    async (changeRequestId: string) => {
      await handleActOnChangeRequest({
        changeRequestId,
        action: "ACTION_PUBLISH",
        successMessage: "Change request published",
        fallbackErrorMessage: "Failed to publish",
        onSuccess: async () => {
          activeCanvasVersionIdRef.current = "";
          setActiveCanvasVersion(null);
          setSearchParams((current) => {
            const next = new URLSearchParams(current);
            next.delete("version");
            return next;
          });
          await refreshLatestLiveCanvasData();
        },
      });
    },
    [handleActOnChangeRequest, refreshLatestLiveCanvasData, setSearchParams],
  );

  const handleRejectChangeRequest = useCallback(
    async (changeRequestId: string) => {
      await handleActOnChangeRequest({
        changeRequestId,
        action: "ACTION_REJECT",
        successMessage: "Change request rejected",
        fallbackErrorMessage: "Failed to reject",
      });
    },
    [handleActOnChangeRequest],
  );

  const handleReopenChangeRequest = useCallback(
    async (changeRequestId: string) => {
      await handleActOnChangeRequest({
        changeRequestId,
        action: "ACTION_REOPEN",
        successMessage: "Change request reopened",
        fallbackErrorMessage: "Failed to reopen",
      });
    },
    [handleActOnChangeRequest],
  );

  const handleGoToVersioningToResolveConflicts = useCallback((changeRequestId: string) => {
    setVersionNodeDiffContext(null);
    setSelectedChangeRequestId(changeRequestId);
    setResolvingConflictChangeRequestId(changeRequestId);
    setIsVersionControlOpen(true);
  }, []);

  const handlePreviewPreviousVersionViewDetails = useCallback(() => {
    if (!selectedCanvasVersionID || !selectedCanvasVersion) {
      return;
    }
    const index = liveVersions.findIndex((version) => version.metadata?.id === selectedCanvasVersionID);
    if (index < 0) {
      return;
    }
    const previousVersion = liveVersions[index + 1];
    if (!previousVersion) {
      return;
    }
    const changeRequest = liveVersionChangeRequestsByVersionId.get(selectedCanvasVersionID);
    setVersionNodeDiffContext({
      version: selectedCanvasVersion,
      previousVersion,
      changeRequest,
    });
  }, [selectedCanvasVersionID, selectedCanvasVersion, liveVersions, liveVersionChangeRequestsByVersionId]);

  const handleOpenAwaitingApprovalNodeDiff = useCallback(() => {
    if (!selectedCanvasVersionID) {
      return;
    }
    const entry = pendingApprovalVersions.find((item) => item.version.metadata?.id === selectedCanvasVersionID);
    const baseline = liveVersions[0];
    if (!entry || !baseline) {
      return;
    }
    setVersionNodeDiffContext({
      version: entry.version,
      previousVersion: baseline,
      changeRequest: entry.changeRequest,
    });
  }, [selectedCanvasVersionID, pendingApprovalVersions, liveVersions]);

  const awaitingApprovalBanner = useMemo(() => {
    if (!isViewingPendingApprovalVersion || !selectedCanvasVersionID) {
      return undefined;
    }

    const entry = pendingApprovalVersions.find((item) => item.version.metadata?.id === selectedCanvasVersionID);
    const changeRequest = entry?.changeRequest;
    const changeRequestId = changeRequest?.metadata?.id;
    if (!changeRequestId) {
      return undefined;
    }

    const phase = getChangeRequestReviewPhase(changeRequest, liveCanvas?.spec?.changeManagement);
    const reviewUi =
      phase.kind === "none"
        ? {
            label: "Awaiting Approval",
            floatingBarBgClassName: "bg-orange-50",
            dotClassName: "text-[11px] text-orange-500 shrink-0",
            titleClassName: "font-medium text-orange-500 truncate",
          }
        : {
            label: phase.label,
            floatingBarBgClassName: phase.floatingBarBgClassName,
            dotClassName: `${phase.floatingBarDotClassName} shrink-0`,
            titleClassName: `truncate font-medium ${phase.floatingBarTitleClassName}`,
          };

    return {
      title: changeRequest.metadata?.title?.trim() || "Change request",
      description: changeRequest.metadata?.description?.trim(),
      onApprove: () => handleApproveChangeRequest(changeRequestId),
      onReject: () => handleRejectChangeRequest(changeRequestId),
      onPublish: () => handlePublishChangeRequest(changeRequestId),
      onOpenVersioningTab: () => {
        setSelectedChangeRequestId(changeRequestId);
        setIsVersionControlOpen(true);
      },
      onViewNodeDiff: handleOpenAwaitingApprovalNodeDiff,
      canAct: canUpdateCanvas && !isTemplate && !canvasDeletedRemotely,
      actionPending: actOnCanvasChangeRequestMutation.isPending,
      reviewUi,
    };
  }, [
    isViewingPendingApprovalVersion,
    selectedCanvasVersionID,
    pendingApprovalVersions,
    liveCanvas?.spec?.changeManagement,
    handleApproveChangeRequest,
    handleRejectChangeRequest,
    handlePublishChangeRequest,
    handleOpenAwaitingApprovalNodeDiff,
    canUpdateCanvas,
    isTemplate,
    canvasDeletedRemotely,
    actOnCanvasChangeRequestMutation.isPending,
  ]);

  const handleResolveChangeRequest = useCallback(
    async (data: { changeRequestId: string; nodes: Record<string, unknown>[]; edges: Record<string, unknown>[] }) => {
      if (!organizationId || !canvasId || !canvas?.metadata?.name) {
        return;
      }

      try {
        const response = await resolveCanvasChangeRequestMutation.mutateAsync({
          changeRequestId: data.changeRequestId,
          name: canvas.metadata.name,
          description: canvas.metadata.description || "",
          nodes: data.nodes,
          edges: data.edges,
        });

        const resolvedVersion = response?.data?.version;
        const resolvedVersionID = resolvedVersion?.metadata?.id || "";
        if (resolvedVersion && resolvedVersionID) {
          setActiveCanvasVersion(resolvedVersion);
          setSearchParams((current) => {
            const next = new URLSearchParams(current);
            next.set("version", resolvedVersionID);
            return next;
          });
        }

        const resolvedChangeRequestID = response?.data?.changeRequest?.metadata?.id || data.changeRequestId;
        setSelectedChangeRequestId(resolvedChangeRequestID);
        setResolvingConflictChangeRequestId("");
        showSuccessToast("Change request conflicts resolved");
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to resolve")));
      }
    },
    [
      organizationId,
      canvasId,
      canvas?.metadata?.name,
      canvas?.metadata?.description,
      resolveCanvasChangeRequestMutation,
      setSearchParams,
    ],
  );

  const handleUseVersion = useCallback(
    (versionID: string) => {
      if (!organizationId || !canvasId) {
        return;
      }

      const version = selectableVersionsById.get(versionID);
      if (!version) {
        showErrorToast("Version not found");
        return;
      }

      setIsCreateChangeRequestMode(false);

      const isPublished = isPublishedVersion(version);
      const isOwnedDraft = !isPublished && version.metadata?.owner?.id === currentUserId;
      const isPendingApprovalVersion = pendingApprovalVersionIds.has(version.metadata?.id || "");
      const isCurrentLive = version.metadata?.id === liveCanvasVersionId;
      if (!isOwnedDraft && !isPublished && !isPendingApprovalVersion) {
        showErrorToast("You can only use your edit version, open change requests, or published live history");
        return;
      }

      clearPendingAutoSaveWork();

      const previousDraftVersionId = activeCanvasVersionIdRef.current;
      if (previousDraftVersionId && draftCanvasSpec) {
        draftCanvasSpecsRef.current.set(previousDraftVersionId, draftCanvasSpec);
      }

      if (!isCurrentLive) {
        void queryClient.cancelQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      }

      activeCanvasVersionIdRef.current = isCurrentLive ? "" : versionID;

      if (isCurrentLive) {
        setDraftCanvasSpec(null);
        setActiveCanvasVersion(null);
      } else {
        setDraftCanvasSpec(version.spec ?? null);
        setActiveCanvasVersion(version);
      }

      const versionChangeRequest = canvasChangeRequests.find(
        (changeRequest) => changeRequest.metadata?.versionId === version.metadata?.id,
      );
      setSelectedChangeRequestId(versionChangeRequest?.metadata?.id || "");

      lastAppliedVersionSnapshotRef.current = "";
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
      setLastSavedWorkflowSnapshot(null);

      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (isCurrentLive) {
          next.delete("version");
        } else {
          next.set("version", versionID);
        }
        return next;
      });

      queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
        if (!current) {
          return current;
        }

        if (isCurrentLive) {
          const liveSpec = liveCanvasVersion?.spec || liveCanvas?.spec;
          return {
            ...current,
            spec: liveSpec,
          };
        }

        if (!version.spec) {
          return current;
        }
        return {
          ...current,
          spec: { ...current.spec, ...version.spec },
        };
      });

      if (isCurrentLive) {
        // Refresh live data in background to pick up latest status/events.
        void Promise.all([
          queryClient.invalidateQueries({
            queryKey: canvasKeys.detail(organizationId, canvasId),
            refetchType: "all",
          }),
          queryClient.invalidateQueries({
            queryKey: canvasKeys.eventList(canvasId, 50),
            refetchType: "all",
          }),
        ]).then(() => {
          const refreshedLiveCanvas = queryClient.getQueryData<CanvasesCanvas>(
            canvasKeys.detail(organizationId, canvasId),
          );
          if (!refreshedLiveCanvas) {
            return;
          }

          if (activeCanvasVersionIdRef.current !== "") {
            return;
          }

          initializeFromWorkflow(refreshedLiveCanvas);
        });
      }
    },
    [
      organizationId,
      canvasId,
      selectableVersionsById,
      currentUserId,
      pendingApprovalVersionIds,
      liveCanvasVersionId,
      liveCanvasVersion?.spec,
      liveCanvas?.spec,
      queryClient,
      setSearchParams,
      canvasChangeRequests,
      initializeFromWorkflow,
      clearPendingAutoSaveWork,
      setLastSavedWorkflowSnapshot,
      draftCanvasSpec,
    ],
  );

  const handleSubmitCreateChangeRequest = useCallback(
    async ({ title, description }: { title: string; description: string }) => {
      if (!organizationId || !canvasId) {
        return;
      }

      if (isChangeManagementDisabled) {
        showErrorToast("Change management is disabled for this canvas.");
        return;
      }

      const editVersionID = createChangeRequestVersion?.metadata?.id || "";

      if (!editVersionID) {
        showErrorToast("Enable edit mode before creating a change request");
        return;
      }

      const isReady = await ensureVersionActionDraftReady(
        "Unable to prepare the latest version changes for a change request",
      );
      if (!isReady) {
        return;
      }

      try {
        const response = await createCanvasChangeRequestMutation.mutateAsync({
          title,
          description,
        });
        const changeRequest = response?.data?.changeRequest;
        const changeRequestID = changeRequest?.metadata?.id || "";

        await queryClient.invalidateQueries({ queryKey: canvasKeys.changeRequestList(canvasId) });
        setIsCreateChangeRequestMode(false);
        if (liveCanvasVersionId) {
          handleUseVersion(liveCanvasVersionId);
        }
        if (changeRequestID) {
          setSelectedChangeRequestId(changeRequestID);
        }
        setIsVersionControlOpen(true);
        setSuppressUnpublishedDraftDiscard(true);
        showSuccessToast("Change request created");
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to create change request")));
      }
    },
    [
      organizationId,
      canvasId,
      isChangeManagementDisabled,
      createChangeRequestVersion,
      createCanvasChangeRequestMutation,
      ensureVersionActionDraftReady,
      queryClient,
      liveCanvasVersionId,
      handleUseVersion,
    ],
  );

  const handleToggleEditMode = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    if (!canUpdateCanvas) {
      showErrorToast("You don't have permission to edit this canvas");
      return;
    }

    if (isTemplate) {
      showErrorToast("Template canvases are read-only");
      return;
    }

    if (hasEditableVersion) {
      if (!liveCanvasVersionId) {
        showErrorToast("No live version available");
        return;
      }
      handleUseVersion(liveCanvasVersionId);
      return;
    }

    setSuppressUnpublishedDraftDiscard(false);

    const existingDraftVersionID = draftVersions[0]?.metadata?.id;
    if (existingDraftVersionID) {
      handleUseVersion(existingDraftVersionID);
      return;
    }

    await handleCreateVersion();
  }, [
    organizationId,
    canvasId,
    canUpdateCanvas,
    isTemplate,
    hasEditableVersion,
    liveCanvasVersionId,
    draftVersions,
    handleUseVersion,
    handleCreateVersion,
  ]);

  const handleResetDraftChanges = useCallback(async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    if (!canUpdateCanvas) {
      showErrorToast("You don't have permission to edit this canvas");
      return;
    }

    if (isTemplate) {
      showErrorToast("Template canvases are read-only");
      return;
    }

    if (!hasEditableVersion || !activeCanvasVersionId) {
      showErrorToast("Enable edit mode before discarding draft");
      return;
    }

    const shouldDiscard = window.confirm("Discard this draft? You will be redirected to the live canvas.");
    if (!shouldDiscard) {
      return;
    }

    clearPendingAutoSaveWork();

    try {
      setIsResetDraftPending(true);
      await deleteCanvasVersionMutation.mutateAsync(activeCanvasVersionId);

      setIsCreateChangeRequestMode(false);
      setSelectedChangeRequestId("");
      setActiveCanvasVersion(null);
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
      setLastSavedWorkflowSnapshot(null);
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        next.delete("version");
        return next;
      });

      if (liveCanvasVersion?.spec) {
        queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
          if (!current) {
            return current;
          }

          return {
            ...current,
            spec: { ...current.spec, ...liveCanvasVersion.spec },
          };
        });
      }

      showSuccessToast("Draft discarded");
    } catch (error) {
      const errorMessage =
        (error as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        (error as { message?: string })?.message ||
        "Failed to discard draft";
      showErrorToast(errorMessage);
    } finally {
      setIsResetDraftPending(false);
    }
  }, [
    organizationId,
    canvasId,
    canUpdateCanvas,
    isTemplate,
    hasEditableVersion,
    activeCanvasVersionId,
    deleteCanvasVersionMutation,
    liveCanvasVersion,
    queryClient,
    setSearchParams,
    clearPendingAutoSaveWork,
    setLastSavedWorkflowSnapshot,
  ]);

  const getYamlExportPayload = useCallback(
    (canvasNodes: CanvasNode[]) => {
      if (!canvas) return null;

      const updatedNodes =
        canvas.spec?.nodes?.map((node) => {
          const canvasNode = canvasNodes.find((cn) => cn.id === node.id);
          const componentType = (canvasNode?.data?.type as string) || "";
          if (canvasNode) {
            return {
              ...node,
              position: {
                x: Math.round(canvasNode.position.x),
                y: Math.round(canvasNode.position.y),
              },
              isCollapsed: (canvasNode.data[componentType] as { collapsed: boolean })?.collapsed || false,
            };
          }
          return node;
        }) || [];

      const exportWorkflow = {
        apiVersion: "v1",
        kind: "Canvas",
        metadata: {
          id: canvas.metadata?.id || "",
          name: canvas.metadata?.name || "Canvas",
          description: canvas.metadata?.description || "",
          isTemplate: canvas.metadata?.isTemplate ?? false,
        },
        spec: {
          nodes: updatedNodes,
          edges: canvas.spec?.edges || [],
        },
      };

      const yamlText = yaml.dump(exportWorkflow, {
        forceQuotes: true,
        quotingType: '"',
        lineWidth: 0,
      });

      const safeName = (canvas.metadata?.name || "canvas")
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/(^-|-$)/g, "");
      const filename = `${safeName || "canvas"}.yaml`;

      return { yamlText, filename };
    },
    [canvas],
  );

  const handleUseTemplateSubmit = useCallback(
    async (data: { name: string; description?: string; templateId?: string }) => {
      if (!canvas || !organizationId) return;

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId!)) || canvas;

      const result = await createWorkflowMutation.mutateAsync({
        name: data.name,
        description: data.description,
        nodes: latestWorkflow.spec?.nodes,
        edges: latestWorkflow.spec?.edges,
        method: "template",
        templateId: data.templateId ?? canvasId,
      });

      if (result?.data?.canvas?.metadata?.id) {
        setIsUseTemplateOpen(false);
        navigate(`/${organizationId}/canvases/${result.data.canvas.metadata.id}`);
      }
    },
    [canvas, organizationId, createWorkflowMutation, navigate, queryClient, canvasId],
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
        setResolvedExecutionIds((prev) => {
          const next = new Set(prev);
          executionIds.forEach((id) => next.add(id));
          return next;
        });
        await queryClient.invalidateQueries({ queryKey: [...canvasKeys.events(), canvasId] });
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
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
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
    [canvas],
  );

  const getCustomField = useCallback(
    (nodeId: string, onRun?: (initialData?: string) => void, integration?: OrganizationsIntegration) => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
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
        onRun?: (initialData?: string) => void;
        integration?: OrganizationsIntegration;
      } = onRun ? { onRun } : {};
      if (integration) context.integration = integration;

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
    [canvas],
  );

  const { yamlPayload, handleYamlViewCopy, handleYamlViewDownload } = useCanvasYaml({
    canvasId: canvasId!,
    organizationId: organizationId!,
    nodes,
    getYamlExportPayload,
  });

  const importYamlGuardError =
    !canvas || !organizationId || !canvasId
      ? "Canvas data is not available"
      : !canUpdateCanvas
        ? "You don't have permission to update this canvas"
        : isTemplate
          ? "Template canvases are read-only"
          : !activeCanvasVersionId
            ? "Enable edit mode before saving changes"
            : null;

  const handleImportYaml = useCallback(
    async (data: { nodes: unknown[]; edges: unknown[] }) => {
      if (importYamlGuardError) throw new Error(importYamlGuardError);

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas!.spec,
          nodes: data.nodes as ComponentsNode[],
          edges: data.edges as ComponentsEdge[],
        },
      };

      const savingVersionID = activeCanvasVersionId || undefined;
      const result = await enqueueCanvasSave(updatedWorkflow, savingVersionID);
      if (result.status !== "saved") {
        return;
      }
      if (result.response?.data?.version && savingVersionID && activeCanvasVersionIdRef.current === savingVersionID) {
        setActiveCanvasVersion(result.response.data.version);
      }
      if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
        return;
      }
      queryClient.setQueryData(canvasKeys.detail(organizationId!, canvasId!), updatedWorkflow);
      setLastSavedWorkflowSnapshot(updatedWorkflow);

      if (result.matchesCurrentCanvas && !result.hasQueuedFollowUp) {
        setHasUnsavedChanges(false);
        setHasNonPositionalUnsavedChanges(false);
      }
      showSuccessToast("Canvas changes saved");
    },
    [
      importYamlGuardError,
      canvas,
      activeCanvasVersionId,
      enqueueCanvasSave,
      organizationId,
      canvasId,
      queryClient,
      setLastSavedWorkflowSnapshot,
    ],
  );

  const isInitialCanvasBootstrapLoading =
    !canvas && (canvasLoading || triggersLoading || componentsLoading || widgetsLoading || usersLoading);
  const isDraftCanvasLoading =
    isViewingDraftVersion &&
    !!activeCanvasVersionId &&
    !draftSpecToRender &&
    (loadedCanvasVersionLoading || loadedCanvasVersionFetching || !loadedCanvasVersion?.spec);

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

  if (!canvas && !canvasLoading && !isDraftCanvasLoading) {
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
    setHasUnsavedChanges(false);
    setHasNonPositionalUnsavedChanges(false);
    setRemoteCanvasUpdatePending(false);
    setLastSavedWorkflowSnapshot(null);

    await queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId) });
    if (isViewingLiveVersion) {
      await queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
      await queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
      return;
    }

    if (activeCanvasVersionId) {
      await queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(canvasId, activeCanvasVersionId) });
    }
  };

  const hasRunBlockingChanges = hasUnsavedChanges && hasNonPositionalUnsavedChanges;
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
  const templateIntegrations = isTemplate ? extractIntegrations(canvas?.spec?.nodes) : [];
  const templateTags = isTemplate ? getTemplateTags(canvas?.metadata?.name) : [];
  const templateNodeCounts = isTemplate ? countNodesByType(canvas?.spec?.nodes) : { components: 0, triggers: 0 };
  const templateBanner = isTemplate ? (
    <div className="bg-orange-50 border-b border-orange-200 px-4 py-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 mb-1 truncate">{canvas?.metadata?.name || "Template"}</p>

          {canvas?.metadata?.description ? (
            <p className="text-[13px] text-gray-600 mb-2 max-w-2xl">{canvas.metadata.description}</p>
          ) : null}

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
            {templateTags.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {templateTags.map((tag) => (
                  <Badge key={tag} variant="outline" className="text-[11px] px-1.5 py-0 text-gray-600 bg-white">
                    {tag}
                  </Badge>
                ))}
              </div>
            ) : null}

            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-gray-500">Requires:</span>
              {templateIntegrations.length > 0 ? (
                <div className="flex items-center gap-2.5">
                  {templateIntegrations.map((name) => {
                    const iconSrc = getIntegrationIconSrc(name);
                    if (!iconSrc) return null;
                    return (
                      <span key={name} className="inline-flex items-center gap-1">
                        <span className="inline-block h-4 w-4 shrink-0">
                          <img src={iconSrc} alt={name} className="h-full w-full object-contain" />
                        </span>
                        <span className="text-xs text-gray-600 capitalize">{name}</span>
                      </span>
                    );
                  })}
                </div>
              ) : (
                <span className="text-xs text-gray-500">No integrations needed</span>
              )}
            </div>

            <span className="text-xs text-gray-500">
              {templateNodeCounts.components > 0 &&
                `${templateNodeCounts.components} ${templateNodeCounts.components === 1 ? "component" : "components"}`}
              {templateNodeCounts.components > 0 && templateNodeCounts.triggers > 0 && " · "}
              {templateNodeCounts.triggers > 0 &&
                `${templateNodeCounts.triggers} ${templateNodeCounts.triggers === 1 ? "trigger" : "triggers"}`}
            </span>
          </div>
        </div>

        <div className="flex flex-col items-end gap-2 shrink-0 self-start">
          <Link
            to={`/${organizationId}/templates`}
            className="flex items-center gap-1 text-sm text-gray-600 hover:text-gray-900 transition-colors"
          >
            <ArrowLeft size={14} />
            <span>Back to templates</span>
          </Link>
          <Button size="sm" onClick={() => setIsUseTemplateOpen(true)}>
            {hasUnsavedChanges ? "Save changes to new canvas" : "Use template"}
          </Button>
        </div>
      </div>
    </div>
  ) : null;

  const canvasViewKey = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "live";
  const canvasRenderKey = `${canvasViewKey}:${isDraftCanvasLoading ? "draft-loading" : "draft-ready"}`;
  const headerBanners = [remoteUpdateBanner, templateBanner].filter(Boolean);
  const headerBanner = headerBanners.length > 0 ? <div className="flex flex-col">{headerBanners}</div> : null;
  const saveDisabled = !canUpdateCanvas || !hasEditableVersion;
  const saveDisabledTooltip = !canUpdateCanvas
    ? "You don't have permission to edit this canvas."
    : !hasEditableVersion
      ? "Enable edit mode to save changes."
      : undefined;
  const saveButtonHidden =
    isTemplate || !canUpdateCanvas || !hasEditableVersion || !hasUnsavedChanges || (!isReadOnly && isAutoSaveQueued);
  const saveIsPrimary = hasUnsavedChanges && !isReadOnly && !isAutoSaveQueued;
  const toggleEditModeDisabled =
    !canUpdateCanvas ||
    canvasDeletedRemotely ||
    createCanvasVersionMutation.isPending ||
    (hasEditableVersion && !liveCanvasVersionId);
  const toggleEditModeDisabledTooltip = !canUpdateCanvas
    ? "You don't have permission to edit this canvas."
    : canvasDeletedRemotely
      ? "This canvas was deleted in another session."
      : hasEditableVersion && !liveCanvasVersionId
        ? "No live version available."
        : undefined;
  const resetDraftDisabled =
    !hasEditableVersion ||
    !canUpdateCanvas ||
    canvasDeletedRemotely ||
    deleteCanvasVersionMutation.isPending ||
    isResetDraftPending ||
    !activeCanvasVersionId;
  const resetDraftDisabledTooltip = !canUpdateCanvas
    ? "You don't have permission to edit this canvas."
    : canvasDeletedRemotely
      ? "This canvas was deleted in another session."
      : !activeCanvasVersionId
        ? "Draft version not found."
        : !hasEditableVersion
          ? "Enable edit mode before discarding draft."
          : undefined;
  const { publishVersionDisabled, publishVersionDisabledTooltip } = getVersionActionAvailability({
    isChangeManagementDisabled,
    hasEditableVersion,
    createChangeRequestPending: createCanvasChangeRequestMutation.isPending,
    publishPending: publishCanvasVersionMutation.isPending,
    canvasDeletedRemotely,
    isPreparingVersionAction,
    hasDraftDiffVersusLive: !!latestDraftVersion && hasDraftGraphDiffVersusLive,
  });
  const headerMode = canvasMode === "edit" ? "version-edit" : "version-live";
  const hasUnpublishedDraftChanges =
    !suppressUnpublishedDraftDiscard && !!latestDraftVersion && hasDraftGraphDiffVersusLive;
  const canvasStateMode = hasEditableVersion
    ? "editing"
    : isViewingPendingApprovalVersion
      ? "awaiting-approval"
      : !isViewingCurrentLiveVersion
        ? "previewing-previous-version"
        : "default";
  const exitEditModeDisabled =
    !canUpdateCanvas || canvasDeletedRemotely || !hasEditableVersion || createCanvasVersionMutation.isPending;
  const exitEditModeDisabledTooltip = !canUpdateCanvas
    ? "You don't have permission to edit this canvas."
    : canvasDeletedRemotely
      ? "This canvas was deleted in another session."
      : !hasEditableVersion
        ? "Edit mode is not enabled."
        : undefined;
  const runDisabled =
    hasRunBlockingChanges ||
    isTemplate ||
    !canUpdateCanvas ||
    canvasDeletedRemotely ||
    isViewingDraftVersion ||
    !isViewingCurrentLiveVersion;
  const runDisabledTooltip = canvasDeletedRemotely
    ? "This canvas was deleted in another session."
    : isViewingDraftVersion
      ? "Draft versions do not execute. Publish to run this canvas."
      : !isViewingCurrentLiveVersion
        ? "Only the current live version can execute."
        : !canUpdateCanvas
          ? "You don't have permission to emit events on this canvas."
          : isTemplate
            ? "Templates are read-only"
            : hasRunBlockingChanges
              ? "Save canvas changes before running"
              : undefined;

  return (
    <>
      <div className="relative h-full w-full">
        <CanvasPage
          key={canvasRenderKey}
          // Persist right sidebar in query params
          initialSidebar={{
            isOpen: searchParams.get("sidebar") === "1",
            nodeId: searchParams.get("node") || null,
          }}
          onSidebarChange={handleSidebarChange}
          title={canvas?.metadata?.name || liveCanvas?.metadata?.name || (isTemplate ? "Template" : "Canvas")}
          headerBanner={headerBanner}
          canvasStateMode={canvasStateMode}
          onPreviewPreviousVersionViewDetails={handlePreviewPreviousVersionViewDetails}
          awaitingApprovalBanner={awaitingApprovalBanner}
          showCanvasSettingsMenu={canUpdateCanvas}
          isVersionControlOpen={isVersionControlOpen}
          onOpenVersionControl={!hasEditableVersion ? () => setIsVersionControlOpen((prev) => !prev) : undefined}
          versionControlButtonTooltip={isVersionControlOpen ? "Close versions" : "Open versions"}
          versionControlNotificationCount={pendingApprovalVersions.length}
          showBottomStatusControls={!isTemplate}
          hideAddControls={isTemplate}
          memoryItemCount={canvasMemoryEntries.length}
          onMemoryOpen={() => setIsMemoryViewModalOpen(true)}
          onYamlOpen={() => setIsYamlViewModalOpen(true)}
          nodes={nodes}
          edges={edges}
          organizationId={organizationId}
          canvasId={canvasId}
          getSidebarData={getSidebarData}
          loadSidebarData={loadSidebarData}
          getTabData={getTabData}
          getNodeEditData={getNodeEditData}
          getAutocompleteExampleObj={getAutocompleteExampleObj}
          getCustomField={getCustomField}
          onNodeConfigurationSave={!isReadOnly ? handleNodeConfigurationSave : undefined}
          configurationSaveMode={isReadOnly ? "manual" : "auto"}
          onAnnotationUpdate={!isReadOnly ? handleAnnotationUpdate : undefined}
          onAnnotationBlur={!isReadOnly ? handleAnnotationBlur : undefined}
          onSave={isTemplate ? undefined : handleSave}
          onEdgeCreate={!isReadOnly ? handleEdgeCreate : undefined}
          onNodeDelete={!isReadOnly ? handleNodeDelete : undefined}
          onNodesDelete={!isReadOnly ? handleNodesDelete : undefined}
          onDuplicateNodes={!isReadOnly ? handleNodesDuplicate : undefined}
          onAutoLayoutNodes={!isReadOnly ? handleAutoLayoutNodes : undefined}
          onEdgeDelete={!isReadOnly ? handleEdgeDelete : undefined}
          isAutoLayoutOnUpdateEnabled={isAutoLayoutOnUpdateEnabled && !isReadOnly}
          onToggleAutoLayoutOnUpdate={!isReadOnly ? handleToggleAutoLayoutOnUpdate : undefined}
          onNodePositionChange={!isReadOnly ? handleNodePositionChange : undefined}
          onNodesPositionChange={!isReadOnly ? handleNodesPositionChange : undefined}
          onToggleView={!isReadOnly ? handleNodeCollapseChange : undefined}
          onRun={isViewingLiveVersion ? handleRun : undefined}
          onTogglePause={!isReadOnly && isViewingLiveVersion ? handleTogglePause : undefined}
          onDuplicate={!isReadOnly ? handleNodeDuplicate : undefined}
          buildingBlocks={buildingBlocks}
          isEditing={isEditing}
          activeCanvasVersionId={activeCanvasVersionId}
          onNodeAdd={!isReadOnly ? handleNodeAdd : undefined}
          onApplyAiOperations={!isReadOnly ? handleApplyAiOperations : undefined}
          onPlaceholderAdd={!isReadOnly ? handlePlaceholderAdd : undefined}
          onPlaceholderConfigure={!isReadOnly ? handlePlaceholderConfigure : undefined}
          integrations={canReadIntegrations ? integrations : []}
          canReadIntegrations={canReadIntegrations}
          canCreateIntegrations={canCreateIntegrations}
          canUpdateIntegrations={canUpdateIntegrations}
          missingIntegrations={missingIntegrations}
          onConnectIntegration={!isReadOnly ? handleConnectIntegration : undefined}
          readOnly={isReadOnly}
          hasFitToViewRef={hasFitToViewRef}
          hasUserToggledSidebarRef={hasUserToggledSidebarRef}
          isSidebarOpenRef={isSidebarOpenRef}
          viewportRef={viewportRef}
          initialFocusNodeId={initialFocusNodeIdRef.current}
          saveIsPrimary={saveIsPrimary}
          saveButtonHidden={saveButtonHidden}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          onPublishVersion={isChangeManagementDisabled ? handlePublishVersion : handleCreateChangeRequest}
          publishVersionLabel={isChangeManagementDisabled ? "Publish" : "Propose Change"}
          publishVersionDisabled={publishVersionDisabled}
          publishVersionDisabledTooltip={publishVersionDisabledTooltip}
          onDiscardVersion={handleResetDraftChanges}
          discardVersionDisabled={resetDraftDisabled}
          discardVersionDisabledTooltip={resetDraftDisabledTooltip}
          headerMode={headerMode}
          onEnterEditMode={handleToggleEditMode}
          enterEditModeDisabled={toggleEditModeDisabled}
          enterEditModeDisabledTooltip={toggleEditModeDisabledTooltip}
          onExitEditMode={handleToggleEditMode}
          exitEditModeDisabled={exitEditModeDisabled}
          exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
          hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
          autoLayoutOnUpdateDisabled={isReadOnly}
          autoLayoutOnUpdateDisabledTooltip={isReadOnly ? "You don't have permission to edit this canvas." : undefined}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onCancelQueueItem={onCancelQueueItem}
          onCancelExecution={isViewingLiveVersion ? onCancelExecution : undefined}
          getAllHistoryEvents={getAllHistoryEvents}
          onLoadMoreHistory={handleLoadMoreHistory}
          getHasMoreHistory={getHasMoreHistory}
          getLoadingMoreHistory={getLoadingMoreHistory}
          onLoadMoreQueue={onLoadMoreQueue}
          getAllQueueEvents={getAllQueueEvents}
          getHasMoreQueue={getHasMoreQueue}
          getLoadingMoreQueue={getLoadingMoreQueue}
          onReEmit={canUpdateCanvas && isViewingLiveVersion ? handleReEmit : undefined}
          onRunItemOpen={isViewingLiveVersion ? handleRunItemOpen : undefined}
          onLogView={handleLogView}
          loadExecutionChain={loadExecutionChain}
          getExecutionState={getExecutionState}
          workflowNodes={canvas?.spec?.nodes}
          components={allComponents}
          triggers={allTriggers}
          logEntries={logEntries}
          runsEvents={isViewingLiveVersion ? runsEventsData.events : []}
          runsTotalCount={isViewingLiveVersion ? runsEventsData.totalCount : 0}
          runsHasNextPage={!!infiniteEventsQuery.hasNextPage}
          runsIsFetchingNextPage={infiniteEventsQuery.isFetchingNextPage}
          onRunsLoadMore={() => infiniteEventsQuery.fetchNextPage()}
          runsNodes={canvas?.spec?.nodes || []}
          runsComponentIconMap={componentIconMap}
          runsNodeQueueItemsMap={visibleNodeQueueItemsMap}
          onRunNodeSelect={handleLogRunNodeSelect}
          onRunExecutionSelect={handleLogRunExecutionSelect}
          onAcknowledgeErrors={canUpdateCanvas && isViewingLiveVersion ? handleAcknowledgeErrors : undefined}
          focusRequest={focusRequest}
          onExecutionChainHandled={handleExecutionChainHandled}
          versionControlSidebar={
            !hasEditableVersion ? (
              <CanvasVersionControlSidebar
                isOpen={isVersionControlOpen}
                onToggle={setIsVersionControlOpen}
                liveCanvasVersionId={liveCanvasVersionId}
                selectedCanvasVersion={selectedCanvasVersion}
                pendingApprovalVersions={pendingApprovalVersions}
                liveVersions={liveVersions}
                liveVersionChangeRequestsByVersionId={liveVersionChangeRequestsByVersionId}
                canUpdateCanvas={canUpdateCanvas}
                isTemplate={isTemplate}
                canvasDeletedRemotely={canvasDeletedRemotely}
                onUseVersion={handleUseVersion}
                onVersionNodeDiffContextChange={setVersionNodeDiffContext}
                onLoadMoreLiveVersions={hasMoreLiveVersions ? () => canvasLiveVersionsQuery.fetchNextPage() : undefined}
                loadMoreLiveVersionsDisabled={!hasMoreLiveVersions || isLoadingMoreLiveVersions}
                loadMoreLiveVersionsPending={isLoadingMoreLiveVersions}
                changeRequestApprovalConfig={liveCanvas?.spec?.changeManagement}
                rejectedVersions={rejectedVersions}
              />
            ) : undefined
          }
        />
        {isDraftCanvasLoading ? (
          <div className="absolute inset-0 z-20 flex items-center justify-center bg-white/70 backdrop-blur-[1px]">
            <div className="flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-600 shadow-sm">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Loading draft canvas...</span>
            </div>
          </div>
        ) : null}
      </div>
      {yamlPayload ? (
        <CanvasYamlModal
          open={isYamlViewModalOpen}
          onOpenChange={setIsYamlViewModalOpen}
          yamlText={yamlPayload.yamlText}
          filename={yamlPayload.filename}
          onCopy={handleYamlViewCopy}
          onDownload={handleYamlViewDownload}
          onImport={!isReadOnly ? handleImportYaml : undefined}
          isImporting={hasLocalSaveActivity}
        />
      ) : null}
      <CanvasMemoryModal
        open={isMemoryViewModalOpen}
        onOpenChange={setIsMemoryViewModalOpen}
        entries={isViewingDraftVersion ? [] : canvasMemoryEntries}
        isLoading={isViewingDraftVersion ? false : canvasMemoryLoading}
        errorMessage={
          isViewingDraftVersion ? undefined : canvasMemoryError instanceof Error ? canvasMemoryError.message : undefined
        }
        onDeleteEntry={
          canUpdateCanvas && isViewingLiveVersion ? (memoryId) => deleteCanvasMemoryEntry.mutate(memoryId) : undefined
        }
        deletingId={deleteCanvasMemoryEntry.isPending ? deleteCanvasMemoryEntry.variables : undefined}
      />
      {resolvingConflictChangeRequest ? (
        <div className="fixed inset-0 z-[100] min-h-0 bg-slate-50">
          <CanvasChangeRequestConflictResolver
            liveCanvasVersion={liveCanvasVersion}
            changeRequest={resolvingConflictChangeRequest}
            canvasName={canvas?.metadata?.name || ""}
            canvasDescription={canvas?.metadata?.description}
            isSubmitting={resolveCanvasChangeRequestMutation.isPending}
            onBack={() => setResolvingConflictChangeRequestId("")}
            onSubmit={handleResolveChangeRequest}
          />
        </div>
      ) : null}
      <CanvasVersionNodeDiffDialog
        context={versionNodeDiffContext}
        onOpenChange={(open) => {
          if (!open) {
            setVersionNodeDiffContext(null);
          }
        }}
        liveVersionOwnerProfilesById={liveVersionOwnerProfilesById}
        changeRequestApprovalConfig={liveCanvas?.spec?.changeManagement}
        canActOnChangeRequests={canUpdateCanvas && !isTemplate && !canvasDeletedRemotely}
        currentUserId={currentUserId}
        changeRequestActionPending={actOnCanvasChangeRequestMutation.isPending}
        onApproveChangeRequest={handleApproveChangeRequest}
        onUnapproveChangeRequest={handleUnapproveChangeRequest}
        onPublishChangeRequest={handlePublishChangeRequest}
        onRejectChangeRequest={handleRejectChangeRequest}
        onReopenChangeRequest={handleReopenChangeRequest}
        liveChangeRequest={versionNodeDiffLiveChangeRequest}
        resolvePending={resolveCanvasChangeRequestMutation.isPending}
        onGoToVersioningToResolveConflicts={handleGoToVersioningToResolveConflicts}
      />
      <CanvasPageModals
        organizationId={organizationId || ""}
        canvas={canvas}
        isUseTemplateOpen={isUseTemplateOpen}
        onCloseUseTemplate={() => setIsUseTemplateOpen(false)}
        onUseTemplateSubmit={handleUseTemplateSubmit}
        isCreateCanvasPending={createWorkflowMutation.isPending}
        isCreateChangeRequestMode={isCreateChangeRequestMode}
        onCreateChangeRequestModeChange={(open) => {
          if (!createCanvasChangeRequestMutation.isPending) {
            setIsCreateChangeRequestMode(open);
          }
        }}
        isCreateChangeRequestPending={createCanvasChangeRequestMutation.isPending}
        createChangeRequestVersion={createChangeRequestVersion}
        createChangeRequestTitle={createChangeRequestTitle}
        createChangeRequestDescription={createChangeRequestDescription}
        onCreateChangeRequestTitleChange={setCreateChangeRequestTitle}
        onCreateChangeRequestDescriptionChange={setCreateChangeRequestDescription}
        createChangeRequestNodeDiffSummary={createChangeRequestNodeDiffSummary}
        isCreateChangeRequestDraftOutdated={isCreateChangeRequestDraftOutdated}
        onSubmitCreateChangeRequest={() =>
          handleSubmitCreateChangeRequest({
            title: createChangeRequestTitle.trim(),
            description: createChangeRequestDescription,
          })
        }
        canvasDeletedRemotely={canvasDeletedRemotely}
        onGoToCanvases={() => {
          if (organizationId) {
            navigate(`/${organizationId}`, { replace: true });
          }
        }}
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
    </>
  );
}

function useExecutionChainData(workflowId: string, queryClient: QueryClient, workflow?: CanvasesCanvas) {
  const loadExecutionChain = useCallback(
    async (
      eventId: string,
      nodeId?: string,
      currentExecution?: Record<string, unknown>,
      forceReload = false,
    ): Promise<CanvasesCanvasNodeExecution[]> => {
      const queryOptions = eventExecutionsQueryOptions(workflowId, eventId);

      let allExecutions: CanvasesCanvasNodeExecution[] = [];

      if (!forceReload) {
        const cachedData = queryClient.getQueryData(queryOptions.queryKey);
        if (cachedData) {
          allExecutions = (cachedData as CanvasesListEventExecutionsResponse)?.executions || [];
        }
      }

      if (allExecutions.length === 0) {
        const options = forceReload ? { ...queryOptions, staleTime: 0 } : queryOptions;
        const data = await queryClient.fetchQuery(options);
        allExecutions = (data as CanvasesListEventExecutionsResponse)?.executions || [];
      }

      // Apply topological filtering - the logic you wanted back!
      if (!allExecutions.length || !workflow || !nodeId) return allExecutions;

      const currentExecutionTime = currentExecution?.createdAt
        ? new Date(currentExecution.createdAt as string).getTime()
        : Date.now();
      const nodesBefore = getNodesBeforeTarget(nodeId, workflow);
      nodesBefore.add(nodeId); // Include current node

      const executionsUpToCurrent = allExecutions.filter((exec) => {
        const execTime = exec.createdAt ? new Date(exec.createdAt).getTime() : 0;
        const isNodeBefore = nodesBefore.has(exec.nodeId || "");
        const isBeforeCurrentTime = execTime <= currentExecutionTime;
        return isNodeBefore && isBeforeCurrentTime;
      });

      // Sort the filtered executions by creation time to get chronological order
      executionsUpToCurrent.sort((a, b) => {
        const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
        const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
        return timeA - timeB;
      });

      return executionsUpToCurrent;
    },
    [workflowId, queryClient, workflow],
  );

  return { loadExecutionChain };
}

// Helper function to build topological path to find all nodes that should execute before the given target node
function getNodesBeforeTarget(targetNodeId: string, workflow: CanvasesCanvas): Set<string> {
  const nodesBefore = new Set<string>();
  if (!workflow?.spec?.edges) return nodesBefore;

  // Build adjacency list for the workflow graph
  const adjacencyList: Record<string, string[]> = {};
  workflow.spec.edges.forEach((edge) => {
    if (!edge.sourceId || !edge.targetId) return;
    if (!adjacencyList[edge.sourceId]) {
      adjacencyList[edge.sourceId] = [];
    }
    adjacencyList[edge.sourceId].push(edge.targetId);
  });

  // DFS to find all nodes that can reach the target
  const visited = new Set<string>();
  const canReachTarget = (nodeId: string): boolean => {
    if (visited.has(nodeId)) return false; // Avoid cycles
    if (nodeId === targetNodeId) return true;

    visited.add(nodeId);
    const neighbors = adjacencyList[nodeId] || [];
    const canReach = neighbors.some((neighbor) => canReachTarget(neighbor));
    visited.delete(nodeId); // Allow revisiting in different paths

    return canReach;
  };

  // Check all nodes to see which ones can reach the target
  const allNodeIds = new Set<string>();
  workflow.spec.edges?.forEach((edge) => {
    if (edge.sourceId) allNodeIds.add(edge.sourceId);
    if (edge.targetId) allNodeIds.add(edge.targetId);
  });
  workflow.spec.nodes?.forEach((node) => {
    if (node.id) allNodeIds.add(node.id);
  });

  allNodeIds.forEach((nodeId) => {
    if (canReachTarget(nodeId)) {
      nodesBefore.add(nodeId);
    }
  });

  return nodesBefore;
}

function prepareData(
  workflow: CanvasesCanvas,
  triggers: TriggersTrigger[],
  components: SuperplaneActionsAction[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  user?: SuperplaneMeUser | null,
  canvasMode: "live" | "edit" = "live",
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const currentUser = buildUserInfo(user);
  const edges = workflow?.spec?.edges?.map(prepareEdge) || [];
  const workflowEdges = workflow?.spec?.edges || [];
  const workflowNodes = workflow?.spec?.nodes || [];
  const nodes =
    workflowNodes
      ?.map((node) => {
        return prepareNode(
          workflowNodes,
          node,
          triggers,
          components,
          nodeEventsMap,
          nodeExecutionsMap,
          nodeQueueItemsMap,
          workflowId,
          queryClient,
          currentUser,
          workflowEdges,
          canvasMode,
        );
      })
      .map((node) => ({
        ...node,
        dragHandle: ".canvas-node-drag-handle",
      })) || [];

  return { nodes, edges };
}

function prepareNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  components: SuperplaneActionsAction[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  currentUser?: User,
  edges?: ComponentsEdge[],
  canvasMode: "live" | "edit" = "live",
): CanvasNode {
  switch (node.type) {
    case "TYPE_TRIGGER":
      return prepareTriggerNode(node, triggers, nodeEventsMap, canvasMode);
    case "TYPE_WIDGET":
      return prepareAnnotationNode(node);

    default:
      return prepareComponentNode({
        nodes,
        node,
        components,
        nodeExecutionsMap,
        nodeQueueItemsMap,
        canvasId: workflowId,
        queryClient,
        currentUser,
        edges,
        canvasMode,
      });
  }
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}-targets->${edge.targetId!}-using->${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
    sourceHandle: edge.channel!,
  };
}

function prepareSidebarData(
  node: ComponentsNode,
  nodes: ComponentsNode[],
  components: SuperplaneActionsAction[],
  triggers: TriggersTrigger[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  totalHistoryCount?: number,
  totalQueueCount?: number,
): SidebarData {
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];
  const events = nodeEventsMap[node.id!] || [];

  // Get metadata based on node type
  const componentMetadata = node.type === "TYPE_ACTION" ? components.find((c) => c.name === node.component) : undefined;
  const triggerMetadata = node.type === "TYPE_TRIGGER" ? triggers.find((t) => t.name === node.component) : undefined;

  const nodeTitle = componentMetadata?.label || triggerMetadata?.label || node.name || "Unknown";
  let iconSlug = "boxes";
  let color = "indigo";

  if (componentMetadata) {
    iconSlug = componentMetadata.icon || iconSlug;
    color = componentMetadata.color || color;
  } else if (triggerMetadata) {
    iconSlug = triggerMetadata.icon || iconSlug;
    color = triggerMetadata.color || color;
  }

  const latestEvents =
    node.type === "TYPE_TRIGGER"
      ? mapTriggerEventsToSidebarEvents(events, node, 5)
      : mapExecutionsToSidebarEvents(executions, nodes, 5);

  // Convert queue items to sidebar events (next in queue)
  const nextInQueueEvents = mapQueueItemsToSidebarEvents(queueItems, nodes, 5);
  const hideQueueEvents = node.type === "TYPE_TRIGGER";

  return {
    latestEvents,
    nextInQueueEvents,
    title: nodeTitle,
    iconSlug,
    iconColor: getColorClass(color),
    totalInHistoryCount: totalHistoryCount ? totalHistoryCount : 0,
    totalInQueueCount: totalQueueCount ? totalQueueCount : 0,
    hideQueueEvents,
    isComposite: false,
  };
}
