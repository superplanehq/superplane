/* eslint-disable @typescript-eslint/no-explicit-any */
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { resolveIcon } from "@/lib/utils";
import { Check, Copy, Loader2, Settings, TriangleAlert, X } from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getHeaderIconSrc, IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import {
  useAvailableIntegrations,
  useCreateIntegration,
  useIntegration,
  useUpdateIntegration,
} from "@/hooks/useIntegrations";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast } from "@/utils/toast";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { ChildEventsState } from "../composite";
import { TabData } from "./SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "./types";
import { LatestTab } from "./LatestTab";
import { SettingsTab } from "./SettingsTab";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import {
  AuthorizationDomainType,
  ConfigurationField,
  CanvasesCanvasNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
  OrganizationsIntegration,
  ComponentsIntegrationRef,
} from "@/api-client";
import { EventState, EventStateMap } from "../componentBase";
import { ReactNode } from "react";
import { ExecutionChainPage, HistoryQueuePage, PageHeader } from "./pages";
import { mapTriggerEventToSidebarEvent } from "@/pages/workflowv2/utils";

/** Optional create-dialog overrides per integration (two-step API + webhook flow). Key = integration name. */
const CREATE_INTEGRATION_DIALOG_OPTIONS: Record<
  string,
  {
    instructionsEndBeforeHeading?: string;
    initialStepFieldNames?: string[];
    webhookStepDescription?: ReactNode;
  }
> = {};

interface ComponentSidebarProps {
  isOpen?: boolean;
  setIsOpen?: (isOpen: boolean) => void;

  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  nodeId?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  totalInQueueCount: number;
  totalInHistoryCount: number;
  hideQueueEvents?: boolean;

  onEventClick?: (event: SidebarEvent) => void;
  onClose?: () => void;
  onSeeFullHistory?: () => void;
  onSeeQueue?: () => void;

  // Action handlers
  onRun?: () => void;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onDuplicate?: () => void;
  onDocs?: () => void;
  onEdit?: () => void;
  onConfigure?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  isCompactView?: boolean;

  // Tab data function to get tab data for each event
  getTabData?: (event: SidebarEvent) => TabData | undefined;

  // Execution and Queue actions
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onPushThrough?: (executionId: string) => void;
  supportsPushThrough?: boolean;

  // Full history props
  getAllHistoryEvents?: () => SidebarEvent[];
  onLoadMoreHistory?: () => void;
  getHasMoreHistory?: () => boolean;
  getLoadingMoreHistory?: () => boolean;

  // Queue pr ops
  getAllQueueEvents?: () => SidebarEvent[];
  onLoadMoreQueue?: () => void;
  getHasMoreQueue?: () => boolean;
  getLoadingMoreQueue?: () => boolean;

  // Execution chain lazy loading
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<any[]>;

  // State registry function for determining execution states
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Settings tab props
  showSettingsTab?: boolean;
  hideRunsTab?: boolean; // Hide the "Runs" tab when showing only settings
  hideNodeId?: boolean; // Hide the node ID with copy functionality
  currentTab?: "latest" | "settings";
  onTabChange?: (tab: "latest" | "settings") => void;
  nodeConfigMode?: "create" | "edit";
  nodeName?: string;
  nodeLabel?: string;
  blockName?: string;
  nodeConfiguration?: Record<string, unknown>;
  nodeConfigurationFields?: ConfigurationField[];
  onNodeConfigSave?: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void;
  onNodeConfigCancel?: () => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  customField?: (configuration: Record<string, unknown>) => ReactNode;
  integrationName?: string;
  integrationRef?: ComponentsIntegrationRef;
  integrations?: OrganizationsIntegration[];
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
  autocompleteExampleObj?: Record<string, unknown> | null;

  // Workflow metadata for ExecutionChainPage
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];

  // Highlighting callback for execution chain nodes
  onHighlightedNodesChange?: (nodeIds: Set<string>) => void;

  // External request to open execution chain
  executionChainEventId?: string | null;
  executionChainExecutionId?: string | null;
  executionChainTriggerEvent?: SidebarEvent | null;
  executionChainRequestId?: number;
  onExecutionChainHandled?: () => void;
  readOnly?: boolean;
}

export const ComponentSidebar = ({
  isOpen,
  nodeId,
  iconSrc,
  iconSlug,
  onEventClick,
  onClose,
  latestEvents,
  nextInQueueEvents,
  totalInQueueCount = 0,
  totalInHistoryCount = 0,
  hideQueueEvents = false,
  onSeeQueue,
  onSeeFullHistory,
  onRun: _onRun,
  runDisabled: _runDisabled,
  runDisabledTooltip: _runDisabledTooltip,
  onDuplicate: _onDuplicate,
  onDocs: _onDocs,
  onEdit: _onEdit,
  onConfigure: _onConfigure,
  onDeactivate: _onDeactivate,
  onToggleView: _onToggleView,
  onDelete: _onDelete,
  onReEmit,
  isCompactView: _isCompactView = false,
  getTabData,
  onCancelQueueItem,
  onCancelExecution,
  onPushThrough,
  supportsPushThrough,
  onLoadMoreHistory,
  getAllHistoryEvents,
  getHasMoreHistory,
  getLoadingMoreHistory,
  onLoadMoreQueue,
  getAllQueueEvents,
  getHasMoreQueue,
  getLoadingMoreQueue,
  loadExecutionChain,
  getExecutionState,
  showSettingsTab = false,
  hideRunsTab = false,
  hideNodeId = false,
  currentTab = "latest",
  onTabChange,
  nodeConfigMode = "edit",
  nodeName = "",
  nodeLabel,
  blockName,
  nodeConfiguration = {},
  nodeConfigurationFields = [],
  onNodeConfigSave,
  onNodeConfigCancel,
  domainId,
  domainType,
  customField,
  integrationName,
  integrationRef,
  integrations,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
  autocompleteExampleObj,
  workflowNodes = [],
  components = [],
  triggers = [],
  blueprints = [],
  onHighlightedNodesChange,
  executionChainEventId,
  executionChainExecutionId,
  executionChainTriggerEvent,
  executionChainRequestId,
  onExecutionChainHandled,
  readOnly = false,
}: ComponentSidebarProps) => {
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);
  // Keep expanded state stable across parent re-renders
  const [openEventIds, setOpenEventIds] = useState<Set<string>>(new Set());

  const [page, setPage] = useState<"overview" | "history" | "queue" | "execution-chain">("overview");
  const [previousPage, setPreviousPage] = useState<"overview" | "history" | "queue" | "execution-chain">("overview");
  const [activeExecutionChainEventId, setActiveExecutionChainEventId] = useState<string | null>(null);
  const [activeExecutionChainTriggerEvent, setActiveExecutionChainTriggerEvent] = useState<SidebarEvent | null>(null);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  const activeTab = currentTab || "latest";
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<ChildEventsState | "all">("all");
  const [justCopied, setJustCopied] = useState(false);
  const [isCreateIntegrationDialogOpen, setIsCreateIntegrationDialogOpen] = useState(false);
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  const [configureIntegrationName, setConfigureIntegrationName] = useState("");
  // Use autocompleteExampleObj directly - current node is already filtered out upstream
  const resolvedAutocompleteExampleObj = autocompleteExampleObj ?? null;

  const { data: availableIntegrationDefinitions = [] } = useAvailableIntegrations();
  const { data: configureIntegration, isLoading: configureIntegrationLoading } = useIntegration(
    domainId ?? "",
    configureIntegrationId ?? "",
  );
  const updateIntegrationMutation = useUpdateIntegration(domainId ?? "", configureIntegrationId ?? "");
  const createIntegrationMutation = useCreateIntegration(domainId ?? "");
  const configureIntegrationDefinition = useMemo(
    () =>
      configureIntegration?.spec?.integrationName
        ? availableIntegrationDefinitions.find((d) => d.name === configureIntegration.spec?.integrationName)
        : undefined,
    [availableIntegrationDefinitions, configureIntegration?.spec?.integrationName],
  );
  const [configureIntegrationConfig, setConfigureIntegrationConfig] = useState<Record<string, unknown>>({});
  const createIntegrationDefinition = useMemo(
    () => (integrationName ? availableIntegrationDefinitions.find((d) => d.name === integrationName) : undefined),
    [availableIntegrationDefinitions, integrationName],
  );
  const selectedIntegrationForDialog = isCreateIntegrationDialogOpen ? createIntegrationDefinition : undefined;
  const integrationHomeHref = useMemo(() => {
    if (!domainId) return "#";
    const selectedIntegrationId =
      integrationRef?.id ||
      integrations?.find((integration) => integration.spec?.integrationName === selectedIntegrationForDialog?.name)
        ?.metadata?.id;
    if (selectedIntegrationId) {
      return `/${domainId}/settings/integrations/${selectedIntegrationId}`;
    }
    return `/${domainId}/settings/integrations`;
  }, [domainId, integrationRef?.id, integrations, selectedIntegrationForDialog?.name]);

  const handleCopyNodeId = useCallback(async () => {
    if (nodeId) {
      await navigator.clipboard.writeText(nodeId);
      setJustCopied(true);
      setTimeout(() => setJustCopied(false), 1000);
    }
  }, [nodeId]);

  const handleOpenCreateIntegrationDialog = useCallback(() => {
    setIsCreateIntegrationDialogOpen(true);
  }, []);

  const handleCloseCreateIntegrationDialog = useCallback(() => {
    setIsCreateIntegrationDialogOpen(false);
  }, []);

  const handleOpenConfigureIntegrationDialog = useCallback((integrationId: string) => {
    setConfigureIntegrationId(integrationId);
  }, []);

  const handleCloseConfigureIntegrationDialog = useCallback(() => {
    setConfigureIntegrationId(null);
    setConfigureIntegrationName("");
    setConfigureIntegrationConfig({});
    updateIntegrationMutation.reset();
  }, [updateIntegrationMutation]);

  const handleConfigureIntegrationSubmit = useCallback(async () => {
    if (!configureIntegrationId || !domainId) return;

    const nextName = configureIntegrationName.trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      await updateIntegrationMutation.mutateAsync({ name: nextName, configuration: configureIntegrationConfig });
      handleCloseConfigureIntegrationDialog();
    } catch (_error) {
      showErrorToast("Failed to update integration");
    }
  }, [
    configureIntegrationId,
    domainId,
    configureIntegrationName,
    configureIntegrationConfig,
    updateIntegrationMutation,
    handleCloseConfigureIntegrationDialog,
  ]);

  const handleConfigureBrowserAction = useCallback(() => {
    const browserAction = configureIntegration?.status?.browserAction;
    if (!browserAction) return;
    const { url, method, formFields } = browserAction;
    if (method?.toUpperCase() === "POST" && formFields) {
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";
      Object.entries(formFields).forEach(([key, value]) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = String(value);
        form.appendChild(input);
      });
      document.body.appendChild(form);
      form.submit();
      document.body.removeChild(form);
    } else if (url) {
      window.open(url, "_blank");
    }
  }, [configureIntegration?.status?.browserAction]);

  useEffect(() => {
    if (configureIntegration?.spec?.configuration) {
      setConfigureIntegrationConfig(configureIntegration.spec.configuration);
    }
  }, [configureIntegration?.spec?.configuration]);

  useEffect(() => {
    setConfigureIntegrationName(
      configureIntegration?.metadata?.name || configureIntegration?.spec?.integrationName || "",
    );
  }, [configureIntegration?.metadata?.name, configureIntegration?.spec?.integrationName]);

  // Seed open ids from incoming props (without closing already open ones)
  useEffect(() => {
    const seeded = new Set(openEventIds);
    latestEvents.forEach((e) => {
      if (e.isOpen) seeded.add(e.id);
    });
    nextInQueueEvents.forEach((e) => {
      if (e.isOpen) seeded.add(e.id);
    });
    if (seeded.size !== openEventIds.size) setOpenEventIds(seeded);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [latestEvents, nextInQueueEvents]);

  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = window.innerWidth - e.clientX;
      // Set min width to 320px and max width to 800px
      const clampedWidth = Math.max(320, Math.min(800, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  // Save sidebar width to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";

      return () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
    }
  }, [isResizing, handleMouseMove, handleMouseUp]);

  const handleToggleOpen = useCallback((eventId: string) => {
    setOpenEventIds((prev) => {
      const next = new Set(prev);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);

  const handleSeeQueue = useCallback(() => {
    setPreviousPage(page as "overview" | "history" | "queue");
    setPage("queue");
    onSeeQueue?.();
  }, [onSeeQueue, page]);

  const handleSeeFullHistory = useCallback(() => {
    setPreviousPage(page as "overview" | "history" | "queue" | "execution-chain");
    setPage("history");
    onSeeFullHistory?.();
  }, [onSeeFullHistory, page]);

  const handleBackToOverview = useCallback(() => {
    if (page === "execution-chain") {
      // When coming back from execution chain, go to the previous page
      setPage(previousPage !== "execution-chain" ? previousPage : "overview");
      // Clear highlights when leaving execution chain
      onHighlightedNodesChange?.(new Set());
    } else {
      setPage("overview");
    }
    setSearchQuery("");
    setStatusFilter("all");
    setActiveExecutionChainEventId(null);
    setActiveExecutionChainTriggerEvent(null);
    setSelectedExecutionId(null);
  }, [page, previousPage, onHighlightedNodesChange]);

  const handleSeeExecutionChain = useCallback(
    (eventId: string, triggerEvent?: SidebarEvent, selectedExecId?: string) => {
      setPreviousPage(page as "overview" | "history" | "queue");
      setActiveExecutionChainEventId(eventId);
      setActiveExecutionChainTriggerEvent(triggerEvent || null);
      setSelectedExecutionId(selectedExecId || null);
      setPage("execution-chain");
    },
    [page],
  );

  useEffect(() => {
    if (!executionChainEventId) {
      return;
    }

    handleSeeExecutionChain(
      executionChainEventId,
      executionChainTriggerEvent || undefined,
      executionChainExecutionId || undefined,
    );
  }, [
    executionChainEventId,
    executionChainExecutionId,
    executionChainRequestId,
    executionChainTriggerEvent,
    handleSeeExecutionChain,
  ]);

  useEffect(() => {
    if (page === "execution-chain") {
      onExecutionChainHandled?.();
    }
  }, [page, onExecutionChainHandled]);

  const listPage = page === "execution-chain" ? previousPage : page;
  const allEvents = React.useMemo(() => {
    if (listPage === "overview") return [];

    switch (listPage) {
      case "history":
        return getAllHistoryEvents?.() || [];
      case "queue":
        return getAllQueueEvents?.() || [];
      default:
        return [];
    }
  }, [getAllHistoryEvents, getAllQueueEvents, listPage]);

  const hasMoreItems = React.useMemo(() => {
    if (listPage === "overview") return false;

    switch (listPage) {
      case "history":
        return getHasMoreHistory?.() || false;
      case "queue":
        return getHasMoreQueue?.() || false;
      default:
        return false;
    }
  }, [getHasMoreHistory, getHasMoreQueue, listPage]);

  const loadingMoreItems = React.useMemo(() => {
    if (listPage === "overview") return false;

    switch (listPage) {
      case "history":
        return getLoadingMoreHistory?.() || false;
      case "queue":
        return getLoadingMoreQueue?.() || false;
      default:
        return false;
    }
  }, [getLoadingMoreHistory, getLoadingMoreQueue, listPage]);

  const handleLoadMoreItems = React.useCallback(() => {
    if (listPage === "overview") return;

    switch (listPage) {
      case "history":
        return onLoadMoreHistory?.();
      case "queue":
        return onLoadMoreQueue?.();
      default:
        return;
    }
  }, [onLoadMoreHistory, onLoadMoreQueue, listPage]);

  const showMoreCount = React.useMemo(() => {
    if (listPage === "overview") return 0;

    switch (listPage) {
      case "history":
        return totalInHistoryCount - allEvents.length;
      case "queue":
        return totalInQueueCount - allEvents.length;
      default:
        return 0;
    }
  }, [allEvents, totalInHistoryCount, totalInQueueCount, listPage]);

  const filteredHistoryEvents = React.useMemo(() => {
    if (!allEvents) return [];
    let events = allEvents;

    if (statusFilter !== "all") {
      events = events.filter((event) => event.state === statusFilter);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      events = events.filter(
        (event) =>
          event.title.toLowerCase().includes(query) ||
          (typeof event.subtitle === "string" && event.subtitle?.toLowerCase().includes(query)) ||
          Object.values(event.values || {}).some((value) => String(value).toLowerCase().includes(query)),
      );
    }

    return events;
  }, [allEvents, statusFilter, searchQuery]);

  // Clear highlights when sidebar closes or when leaving execution chain page
  useEffect(() => {
    if (!isOpen && onHighlightedNodesChange) {
      onHighlightedNodesChange(new Set());
    }
  }, [isOpen, onHighlightedNodesChange]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      onHighlightedNodesChange?.(new Set());
    };
  }, [onHighlightedNodesChange]);

  const isDetailView = page !== "overview";
  const appIconSrc = getHeaderIconSrc(blockName);
  const headerIconSrc = iconSrc ?? appIconSrc;

  if (!isOpen) return null;

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-20 overflow-hidden bg-white flex flex-col"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`absolute left-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-gray-100 transition-colors flex items-center justify-center group z-30 ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-8px" }}
      >
        <div
          className={`w-2 h-14 rounded-full bg-gray-300 group-hover:bg-gray-800 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>
      <div className={"flex items-center justify-between gap-3 px-4 pt-3 relative" + (hideNodeId ? " pb-3" : " pb-8")}>
        <div className="flex flex-col items-start gap-3 w-full">
          <div className="flex justify-between gap-3 w-full">
            <div className="flex flex-col gap-0.5">
              <div className="flex items-center gap-2">
                <div className={`h-7 rounded-full overflow-hidden flex items-center justify-center`}>
                  {headerIconSrc ? (
                    <img src={headerIconSrc} alt={nodeName} className="w-4 h-4 object-contain" />
                  ) : (
                    <Icon size={16} />
                  )}
                </div>
                <h2 className="text-base font-semibold">{nodeName}</h2>
              </div>
              {nodeId && !hideNodeId && (
                <div className="flex items-center gap-2">
                  <span className="text-[13px] text-gray-500 font-mono">{nodeId}</span>
                  <button
                    onClick={handleCopyNodeId}
                    className={"text-gray-500 hover:text-gray-800"}
                    title={justCopied ? "Copied!" : "Copy Node ID"}
                  >
                    {justCopied ? <Check size={14} /> : <Copy size={14} />}
                  </button>
                </div>
              )}
            </div>
            {null}
          </div>
          <div
            onClick={() => onClose?.()}
            className="absolute top-3 right-2 w-6 h-6 hover:bg-slate-950/5 rounded flex items-center justify-center cursor-pointer leading-none"
          >
            <X size={16} />
          </div>
        </div>
      </div>
      <div className="relative flex-1 min-h-0 overflow-hidden">
        <div
          className={`absolute inset-0 flex flex-col bg-white transition-transform duration-300 ease-in-out ${
            isDetailView ? "-translate-x-full" : "translate-x-0"
          } ${isDetailView ? "pointer-events-none" : "pointer-events-auto"}`}
        >
          <Tabs
            value={activeTab}
            onValueChange={(value) => onTabChange?.(value as "latest" | "settings")}
            className="flex-1"
          >
            {showSettingsTab && (
              <div className="border-border border-b-1">
                <div className="flex px-4">
                  {!hideRunsTab && (
                    <button
                      onClick={() => onTabChange?.("latest")}
                      className={`py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors ${
                        activeTab === "latest"
                          ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
                          : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                      }`}
                    >
                      Runs
                    </button>
                  )}
                  <button
                    onClick={() => onTabChange?.("settings")}
                    className={`py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors flex items-center gap-1.5 ${
                      activeTab === "settings"
                        ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
                        : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                    }`}
                  >
                    Configuration
                    {nodeId && workflowNodes.find((n) => n.id === nodeId)?.errorMessage && (
                      <span className="w-1.5 h-1.5 bg-orange-500 rounded-full" />
                    )}
                  </button>
                </div>
              </div>
            )}

            <TabsContent
              value="latest"
              className={!showSettingsTab ? "overflow-y-auto" : "mt-0"}
              style={!showSettingsTab ? { maxHeight: "40vh" } : undefined}
            >
              <LatestTab
                latestEvents={latestEvents}
                nextInQueueEvents={nextInQueueEvents}
                totalInQueueCount={totalInQueueCount}
                hideQueueEvents={hideQueueEvents}
                openEventIds={openEventIds}
                onToggleOpen={handleToggleOpen}
                onEventClick={onEventClick}
                onSeeFullHistory={handleSeeFullHistory}
                onSeeQueue={handleSeeQueue}
                onSeeExecutionChain={handleSeeExecutionChain}
                getTabData={getTabData}
                onCancelQueueItem={onCancelQueueItem}
                onCancelExecution={onCancelExecution}
                onPushThrough={onPushThrough}
                supportsPushThrough={supportsPushThrough}
                onReEmit={onReEmit}
                loadExecutionChain={loadExecutionChain}
                getExecutionState={getExecutionState}
                workflowNodes={workflowNodes}
                components={components}
              />
            </TabsContent>

            {showSettingsTab && (
              <TabsContent value="settings" className="mt-0">
                <SettingsTab
                  mode={nodeConfigMode}
                  nodeId={nodeId}
                  nodeName={nodeName}
                  nodeLabel={nodeLabel}
                  configuration={nodeConfiguration}
                  configurationFields={nodeConfigurationFields}
                  onSave={onNodeConfigSave || (() => {})}
                  onCancel={onNodeConfigCancel}
                  domainId={domainId}
                  domainType={domainType}
                  customField={customField}
                  integrationName={integrationName}
                  integrationRef={integrationRef}
                  integrations={integrations}
                  readOnly={readOnly}
                  canReadIntegrations={canReadIntegrations}
                  canCreateIntegrations={canCreateIntegrations}
                  canUpdateIntegrations={canUpdateIntegrations}
                  integrationDefinition={createIntegrationDefinition}
                  autocompleteExampleObj={resolvedAutocompleteExampleObj}
                  onOpenCreateIntegrationDialog={handleOpenCreateIntegrationDialog}
                  onOpenConfigureIntegrationDialog={handleOpenConfigureIntegrationDialog}
                />
              </TabsContent>
            )}
          </Tabs>
        </div>

        <div
          className={`absolute inset-0 flex flex-col bg-white transition-transform duration-300 ease-in-out ${
            isDetailView ? "translate-x-0" : "translate-x-full"
          } ${isDetailView ? "pointer-events-auto" : "pointer-events-none"}`}
        >
          {page !== "overview" && (
            <div className="flex flex-col flex-1 min-h-0 bg-white">
              <PageHeader
                page={page as "history" | "queue" | "execution-chain"}
                onBackToOverview={handleBackToOverview}
                previousPage={previousPage}
              />

              <div className="relative flex-1 min-h-0 overflow-hidden">
                <div
                  className={`absolute inset-0 flex flex-col transition-transform duration-300 ease-in-out ${
                    page === "execution-chain" ? "-translate-x-full" : "translate-x-0"
                  } ${page === "execution-chain" ? "pointer-events-none" : "pointer-events-auto"}`}
                >
                  {(page === "history" ||
                    page === "queue" ||
                    previousPage === "history" ||
                    previousPage === "queue") && (
                    <HistoryQueuePage
                      page={(page === "execution-chain" ? previousPage : page) as "history" | "queue"}
                      filteredEvents={filteredHistoryEvents}
                      openEventIds={openEventIds}
                      onToggleOpen={handleToggleOpen}
                      onEventClick={onEventClick}
                      onTriggerNavigate={(event) => {
                        if (event.kind === "trigger") {
                          const eventId = event.triggerEventId || event.id;
                          handleSeeExecutionChain(eventId, event);
                        } else if (event.kind === "execution") {
                          const node = workflowNodes?.find((n) => n.id === event.originalExecution?.rootEvent?.nodeId);

                          const rootEventId = event.originalExecution?.rootEvent?.id;
                          if (rootEventId && node && event.originalExecution?.rootEvent) {
                            const triggerEvent = mapTriggerEventToSidebarEvent(
                              event.originalExecution?.rootEvent,
                              node,
                            );
                            handleSeeExecutionChain(rootEventId, triggerEvent, event.executionId);
                          } else {
                            const eventId = event.triggerEventId || event.id;
                            handleSeeExecutionChain(eventId, event, event.executionId);
                          }
                        }
                      }}
                      getTabData={getTabData}
                      onPushThrough={onPushThrough}
                      onCancelExecution={onCancelExecution}
                      supportsPushThrough={supportsPushThrough}
                      onReEmit={onReEmit}
                      loadExecutionChain={loadExecutionChain}
                      getExecutionState={getExecutionState}
                      hasMoreItems={hasMoreItems}
                      loadingMoreItems={loadingMoreItems}
                      showMoreCount={showMoreCount}
                      onLoadMoreItems={handleLoadMoreItems}
                      searchQuery={searchQuery}
                      statusFilter={statusFilter}
                    />
                  )}
                </div>
                <div
                  className={`absolute inset-0 flex flex-col transition-transform duration-300 ease-in-out ${
                    page === "execution-chain" ? "translate-x-0" : "translate-x-full"
                  } ${page === "execution-chain" ? "pointer-events-auto" : "pointer-events-none"}`}
                >
                  {page === "execution-chain" && (
                    <ExecutionChainPage
                      eventId={activeExecutionChainEventId}
                      triggerEvent={activeExecutionChainTriggerEvent || undefined}
                      selectedExecutionId={selectedExecutionId}
                      loadExecutionChain={loadExecutionChain}
                      openEventIds={openEventIds}
                      onToggleOpen={handleToggleOpen}
                      getExecutionState={getExecutionState}
                      getTabData={getTabData}
                      onEventClick={onEventClick}
                      workflowNodes={workflowNodes}
                      components={components}
                      triggers={triggers}
                      blueprints={blueprints}
                      onHighlightedNodesChange={onHighlightedNodesChange}
                    />
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      <IntegrationCreateDialog
        open={isCreateIntegrationDialogOpen}
        onOpenChange={(open) => !open && handleCloseCreateIntegrationDialog()}
        integrationDefinition={createIntegrationDefinition}
        organizationId={domainId ?? ""}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={createIntegrationDefinition?.name ?? ""}
        integrationHomeHref={integrationHomeHref}
        onCreated={() => handleCloseCreateIntegrationDialog()}
        instructionsEndBeforeHeading={
          CREATE_INTEGRATION_DIALOG_OPTIONS[createIntegrationDefinition?.name ?? ""]?.instructionsEndBeforeHeading
        }
        initialStepFieldNames={
          CREATE_INTEGRATION_DIALOG_OPTIONS[createIntegrationDefinition?.name ?? ""]?.initialStepFieldNames
        }
        webhookStepDescription={
          CREATE_INTEGRATION_DIALOG_OPTIONS[createIntegrationDefinition?.name ?? ""]?.webhookStepDescription
        }
      />

      <Dialog open={!!configureIntegrationId} onOpenChange={(open) => !open && handleCloseConfigureIntegrationDialog()}>
        <DialogContent
          className="sm:max-w-2xl max-h-[80vh] overflow-y-auto"
          showCloseButton={!updateIntegrationMutation.isPending}
        >
          {configureIntegrationLoading ? (
            <div className="flex justify-center items-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-500 dark:text-gray-400" />
            </div>
          ) : configureIntegrationId && configureIntegration ? (
            <>
              <DialogHeader>
                <div className="flex items-center gap-3">
                  <IntegrationIcon
                    integrationName={configureIntegration.spec?.integrationName}
                    iconSlug={configureIntegrationDefinition?.icon}
                    className="h-6 w-6 text-gray-500 dark:text-gray-400"
                  />
                  <div className="flex items-center gap-2">
                    <DialogTitle>
                      Configure{" "}
                      {getIntegrationTypeDisplayName(undefined, configureIntegration.spec?.integrationName) ||
                        configureIntegration.spec?.integrationName}
                    </DialogTitle>
                    <a
                      href={
                        configureIntegration.metadata?.id
                          ? `/${domainId}/settings/integrations/${configureIntegration.metadata.id}`
                          : `/${domainId}/settings/integrations`
                      }
                      className="inline-flex h-4 w-4 items-center justify-center text-gray-500 hover:text-gray-800 transition-colors"
                      aria-label="Open integration settings"
                    >
                      <Settings className="h-4 w-4" />
                    </a>
                  </div>
                </div>
              </DialogHeader>
              {configureIntegration.status?.state === "error" && configureIntegration.status?.stateDescription && (
                <div className="flex items-start gap-2 text-sm text-red-700 dark:text-red-300">
                  <TriangleAlert className="h-4 w-4 mt-0.5 flex-shrink-0" />
                  <p>{configureIntegration.status.stateDescription}</p>
                </div>
              )}
              {configureIntegration?.status?.browserAction && (
                <IntegrationInstructions
                  description={configureIntegration.status.browserAction.description}
                  onContinue={configureIntegration.status.browserAction.url ? handleConfigureBrowserAction : undefined}
                  className="mb-6"
                />
              )}
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void handleConfigureIntegrationSubmit();
                }}
                className="space-y-4"
              >
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Integration Name
                    <span className="text-gray-800 ml-1">*</span>
                  </Label>
                  <Input
                    type="text"
                    value={configureIntegrationName}
                    onChange={(e) => setConfigureIntegrationName(e.target.value)}
                    placeholder="e.g., my-app-integration"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">A unique name for this integration</p>
                </div>

                {configureIntegrationDefinition?.configuration &&
                configureIntegrationDefinition.configuration.length > 0 ? (
                  configureIntegrationDefinition.configuration.map((field: ConfigurationField) => {
                    if (!field.name) return null;
                    return (
                      <ConfigurationFieldRenderer
                        key={field.name}
                        field={field}
                        value={configureIntegrationConfig[field.name]}
                        onChange={(value) =>
                          setConfigureIntegrationConfig((prev) => ({ ...prev, [field.name || ""]: value }))
                        }
                        allValues={configureIntegrationConfig}
                        domainId={domainId ?? ""}
                        domainType="DOMAIN_TYPE_ORGANIZATION"
                        organizationId={domainId ?? ""}
                        appInstallationId={configureIntegration.metadata?.id}
                      />
                    );
                  })
                ) : (
                  <p className="text-sm text-gray-500 dark:text-gray-400">No configuration fields available.</p>
                )}

                <DialogFooter className="gap-2 sm:justify-start pt-4">
                  <Button
                    type="submit"
                    color="blue"
                    disabled={updateIntegrationMutation.isPending || !configureIntegrationName.trim()}
                    className="flex items-center gap-2"
                  >
                    {updateIntegrationMutation.isPending ? (
                      <>
                        <Loader2 className="w-4 h-4 animate-spin" />
                        Saving...
                      </>
                    ) : (
                      "Save"
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCloseConfigureIntegrationDialog}
                    disabled={updateIntegrationMutation.isPending}
                  >
                    Cancel
                  </Button>
                </DialogFooter>
                {updateIntegrationMutation.isError && (
                  <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                    <p className="text-sm text-red-800 dark:text-red-200">
                      Failed to update integration: {getApiErrorMessage(updateIntegrationMutation.error)}
                    </p>
                  </div>
                )}
              </form>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
};
