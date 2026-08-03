import { Tabs, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { cn, resolveIcon } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { Check, Copy, X } from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { useAvailableIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import type { TabData } from "./SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "./types";
import { DocsTab } from "./DocsTab";
import { LatestTab } from "./LatestTab";
import { SettingsTab } from "./SettingsTab";
import { useSidebarLayoutStore, useSidebarLayoutViewport, useSidebarMount } from "@/stores/sidebarLayoutStore";
import type {
  ConfigurationField,
  CanvasesCanvasNodeExecution,
  SuperplaneComponentsNode as ComponentsNode,
  OrganizationsIntegration,
  ComponentsIntegrationRef,
} from "@/api-client";
import type { EventState, EventStateMap } from "../componentBase";
import type { ReactNode } from "react";
import { HistoryQueuePage, PageHeader } from "./pages";
import { analytics } from "@/lib/analytics";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";

/** Optional create-dialog overrides per integration (two-step API + webhook flow). Key = integration name. */
const CREATE_INTEGRATION_DIALOG_OPTIONS: Record<
  string,
  {
    instructionsEndBeforeHeading?: string;
    initialStepFieldNames?: string[];
    webhookStepDescription?: ReactNode;
  }
> = {};

function BottomInspectorTabButton({
  active,
  icon,
  label,
  onClick,
  trailing,
}: {
  active: boolean;
  icon: string;
  label: string;
  onClick: () => void;
  trailing?: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "mb-[-1px] flex items-center gap-1 self-stretch border-b px-2.5 text-[13px] font-medium transition-colors",
        active
          ? "border-gray-700 text-gray-800 dark:border-indigo-300 dark:text-indigo-300"
          : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300",
      )}
    >
      {React.createElement(resolveIcon(icon), { size: RUN_NODE_ICON_SIZE, className: "h-3.5 w-3.5 shrink-0" })}
      {label}
      {trailing}
    </button>
  );
}

interface ComponentSidebarProps {
  isOpen?: boolean;
  canvasMode?: "live" | "edit";

  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  nodeId?: string;
  iconSrc?: string;
  iconSlug?: string;
  totalInQueueCount: number;
  totalInHistoryCount: number;
  hideQueueEvents?: boolean;

  onEventClick?: (event: SidebarEvent) => void;
  onClose?: () => void;
  onSeeFullHistory?: () => void;
  onSeeQueue?: () => void;

  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;

  // Tab data function to get tab data for each event
  getTabData?: (event: SidebarEvent) => TabData | undefined;

  // Execution and Queue actions
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;

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

  // State registry function for determining execution states
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Settings tab props
  showSettingsTab?: boolean;
  hideRunsTab?: boolean; // Hide the "Runs" tab when showing only settings
  hideDocsTab?: boolean; // Hide the "Info" tab (e.g. for annotation nodes)
  hideNodeId?: boolean; // Hide the node ID with copy functionality
  currentTab?: "latest" | "settings" | "docs";
  onTabChange?: (tab: "latest" | "settings" | "docs") => void;

  // Docs tab props
  componentDescription?: string;
  componentExamplePayload?: Record<string, unknown>;
  componentPayloadLabel?: string;
  /** Full URL to SuperPlane docs (e.g. docs.superplane.com/components/…#section). */
  componentDocumentationUrl?: string;
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
  ) => void | Promise<void>;
  onNodeConfigCancel?: () => void;
  domainId?: string;
  customField?: (configuration: Record<string, unknown>) => ReactNode;
  integrationName?: string;
  integrationRef?: ComponentsIntegrationRef;
  integrations?: OrganizationsIntegration[];
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
  autocompleteExampleObj?: Record<string, unknown> | null;

  workflowNodes?: ComponentsNode[];
  readOnly?: boolean;
  layout?: "sidebar" | "bottom";
  resolveRunId?: (event: SidebarEvent) => string | null;
  fetchRunId?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRun?: (runId: string, options?: { nodeId?: string }) => void;
}

export const ComponentSidebar = ({
  isOpen,
  canvasMode = "live",
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
  onReEmit,
  getTabData,
  onCancelQueueItem,
  onCancelExecution,
  onLoadMoreHistory,
  getAllHistoryEvents,
  getHasMoreHistory,
  getLoadingMoreHistory,
  onLoadMoreQueue,
  getAllQueueEvents,
  getHasMoreQueue,
  getLoadingMoreQueue,
  getExecutionState,
  showSettingsTab = false,
  hideRunsTab = false,
  hideDocsTab = false,
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
  customField,
  integrationName,
  integrationRef,
  integrations,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
  autocompleteExampleObj,
  componentDescription,
  componentExamplePayload,
  componentPayloadLabel,
  componentDocumentationUrl,
  workflowNodes = [],
  readOnly = false,
  layout = "sidebar",
  resolveRunId,
  fetchRunId,
  onSelectRun,
}: ComponentSidebarProps) => {
  const isBottomLayout = layout === "bottom";
  const sidebarWidth = useSidebarLayoutStore((state) => state.rightWidth);
  const isResizing = useSidebarLayoutStore((state) => state.isRightResizing);
  const setRightResizing = useSidebarLayoutStore((state) => state.setRightResizing);
  const resizeRight = useSidebarLayoutStore((state) => state.resizeRight);
  useSidebarMount("right", Boolean(isOpen) && !isBottomLayout);
  useSidebarLayoutViewport();
  const sidebarRef = useRef<HTMLDivElement>(null);
  const activeResizePointerIdRef = useRef<number | null>(null);
  // Keep expanded state stable across parent re-renders
  const [openEventIds, setOpenEventIds] = useState<Set<string>>(new Set());

  const [page, setPage] = useState<"overview" | "history" | "queue">("overview");
  const shouldShowRunsTab = !hideRunsTab && canvasMode === "live";
  const activeTab = useMemo(() => {
    if (shouldShowRunsTab || currentTab !== "latest") {
      return currentTab || "latest";
    }

    return "settings";
  }, [currentTab, shouldShowRunsTab]);

  useEffect(() => {
    if (!shouldShowRunsTab && currentTab === "latest") {
      onTabChange?.("settings");
    }
  }, [currentTab, onTabChange, shouldShowRunsTab]);

  const [justCopied, setJustCopied] = useState(false);
  const [isCreateIntegrationDialogOpen, setIsCreateIntegrationDialogOpen] = useState(false);
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  // Use autocompleteExampleObj directly - current node is already filtered out upstream
  const resolvedAutocompleteExampleObj = autocompleteExampleObj ?? null;

  const { data: availableIntegrationDefinitions = [] } = useAvailableIntegrations();
  const createIntegrationMutation = useCreateIntegration(domainId ?? "", "node_configuration");

  const createIntegrationDefinition = useMemo(
    () => (integrationName ? availableIntegrationDefinitions.find((d) => d.name === integrationName) : undefined),
    [availableIntegrationDefinitions, integrationName],
  );
  const selectedIntegrationForDialog = isCreateIntegrationDialogOpen ? createIntegrationDefinition : undefined;
  const integrationHomeHref = useMemo(() => {
    if (!domainId) return "#";
    const selectedIntegrationId =
      integrationRef?.id ||
      integrations?.find((integration) => integration.metadata?.integrationName === selectedIntegrationForDialog?.name)
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
    if (integrationName && domainId) {
      analytics.integrationConnectStart(integrationName, "node_configuration", domainId);
    }
  }, [integrationName, domainId]);

  const handleCloseCreateIntegrationDialog = useCallback(() => {
    setIsCreateIntegrationDialogOpen(false);
  }, []);

  const handleOpenConfigureIntegrationDialog = useCallback((integrationId: string) => {
    setConfigureIntegrationId(integrationId);
  }, []);

  const handleCloseConfigureIntegrationDialog = useCallback(() => {
    setConfigureIntegrationId(null);
  }, []);

  // Seed open ids from incoming props (without closing already open ones)
  useEffect(() => {
    setOpenEventIds((current) => {
      const seeded = new Set(current);
      let changed = false;

      for (const event of [...latestEvents, ...nextInQueueEvents]) {
        if (event.isOpen && !seeded.has(event.id)) {
          seeded.add(event.id);
          changed = true;
        }
      }

      return changed ? seeded : current;
    });
  }, [latestEvents, nextInQueueEvents]);

  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const updateSidebarWidthFromPointer = useCallback(
    (clientX: number) => {
      resizeRight(window.innerWidth - clientX);
    },
    [resizeRight],
  );

  useEffect(() => {
    if (isBottomLayout || !isResizing) {
      return;
    }

    const handleWindowPointerMove = (event: PointerEvent) => {
      if (activeResizePointerIdRef.current !== null && event.pointerId !== activeResizePointerIdRef.current) {
        return;
      }
      updateSidebarWidthFromPointer(event.clientX);
    };

    const finishResize = (event: PointerEvent) => {
      if (activeResizePointerIdRef.current !== null && event.pointerId !== activeResizePointerIdRef.current) {
        return;
      }
      activeResizePointerIdRef.current = null;
      setRightResizing(false);
    };

    window.addEventListener("pointermove", handleWindowPointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      window.removeEventListener("pointermove", handleWindowPointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isBottomLayout, isResizing, updateSidebarWidthFromPointer, setRightResizing]);

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      e.preventDefault();
      activeResizePointerIdRef.current = e.pointerId;
      updateSidebarWidthFromPointer(e.clientX);
      setRightResizing(true);
    },
    [updateSidebarWidthFromPointer, setRightResizing],
  );

  const handleToggleOpen = useCallback((eventId: string) => {
    setOpenEventIds((prev) => {
      const next = new Set(prev);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);

  const handleSeeQueue = useCallback(() => {
    setPage("queue");
    onSeeQueue?.();
  }, [onSeeQueue]);

  const handleSeeFullHistory = useCallback(() => {
    setPage("history");
    onSeeFullHistory?.();
  }, [onSeeFullHistory]);

  const handleBackToOverview = useCallback(() => {
    setPage("overview");
  }, []);

  const allEvents = React.useMemo(() => {
    if (page === "overview") return [];

    switch (page) {
      case "history":
        return getAllHistoryEvents?.() || [];
      case "queue":
        return getAllQueueEvents?.() || [];
      default:
        return [];
    }
  }, [getAllHistoryEvents, getAllQueueEvents, page]);

  const hasMoreItems = React.useMemo(() => {
    if (page === "overview") return false;

    switch (page) {
      case "history":
        return getHasMoreHistory?.() || false;
      case "queue":
        return getHasMoreQueue?.() || false;
      default:
        return false;
    }
  }, [getHasMoreHistory, getHasMoreQueue, page]);

  const loadingMoreItems = React.useMemo(() => {
    if (page === "overview") return false;

    switch (page) {
      case "history":
        return getLoadingMoreHistory?.() || false;
      case "queue":
        return getLoadingMoreQueue?.() || false;
      default:
        return false;
    }
  }, [getLoadingMoreHistory, getLoadingMoreQueue, page]);

  const handleLoadMoreItems = React.useCallback(() => {
    if (page === "overview") return;

    switch (page) {
      case "history":
        return onLoadMoreHistory?.();
      case "queue":
        return onLoadMoreQueue?.();
      default:
        return;
    }
  }, [onLoadMoreHistory, onLoadMoreQueue, page]);

  const showMoreCount = React.useMemo(() => {
    if (page === "overview") return 0;

    switch (page) {
      case "history":
        return totalInHistoryCount - allEvents.length;
      case "queue":
        return totalInQueueCount - allEvents.length;
      default:
        return 0;
    }
  }, [allEvents, totalInHistoryCount, totalInQueueCount, page]);

  const isDetailView = page !== "overview";
  const appIconSrc = getHeaderIconSrc(blockName);
  const headerIconSrc = iconSrc ?? appIconSrc;
  const selectedWorkflowNode = useMemo(
    () => (nodeId ? workflowNodes.find((node) => node.id === nodeId) : undefined),
    [nodeId, workflowNodes],
  );
  const configurationHasError = Boolean(selectedWorkflowNode?.errorMessage);

  if (!isOpen) return null;

  return (
    <div
      ref={sidebarRef}
      className={
        isBottomLayout
          ? "ph-no-capture flex min-h-0 flex-1 flex-col"
          : "ph-no-capture absolute right-0 top-0 h-full z-20"
      }
      style={
        isBottomLayout
          ? undefined
          : { width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }
      }
    >
      {!isBottomLayout ? (
        <div
          onPointerDown={handlePointerDown}
          data-testid="component-sidebar-resize-handle"
          className="group absolute left-0 top-0 bottom-0 z-40 w-4 cursor-col-resize touch-none bg-transparent"
          style={{ marginLeft: "-8px" }}
        >
          <div
            aria-hidden
            className={`pointer-events-none absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-500/50 ${
              isResizing ? "bg-slate-950/50 dark:bg-gray-500/50" : ""
            }`}
          />
        </div>
      ) : null}
      <div
        className={
          isBottomLayout
            ? "flex h-full min-h-0 flex-1 flex-col overflow-hidden bg-white dark:bg-gray-900"
            : cn(
                "border-l h-full overflow-hidden bg-white flex flex-col dark:bg-gray-900",
                appDarkModeClasses.sidebarEdge,
              )
        }
      >
        <div
          className={
            isBottomLayout
              ? "flex h-9 shrink-0 items-stretch justify-between border-b border-slate-200 pl-3 dark:border-gray-800/70"
              : "flex items-center justify-between gap-3 px-4 pt-3 relative" + (hideNodeId ? " pb-3" : " pb-8")
          }
        >
          {isBottomLayout ? (
            <>
              <div className="flex min-w-0 flex-1 items-center gap-1.5">
                <RunNodeIcon
                  componentName={selectedWorkflowNode?.component}
                  iconSrc={headerIconSrc}
                  iconSlug={iconSlug}
                  alt={nodeName}
                  size={RUN_NODE_ICON_SIZE}
                  className="h-3.5 w-3.5 shrink-0 text-gray-800 dark:text-gray-100"
                />
                <h3 className="truncate text-[13px] font-medium text-gray-900 dark:text-gray-100">{nodeName}</h3>
              </div>
              <div className="flex shrink-0 items-stretch">
                <div className="flex items-center px-1">
                  <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={() => onClose?.()}>
                    <X className="size-3.5" />
                  </Button>
                </div>
              </div>
            </>
          ) : (
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
                      <span className="text-[13px] text-gray-500 font-mono dark:text-gray-400">{nodeId}</span>
                      <button
                        onClick={handleCopyNodeId}
                        className={"text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"}
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
                className="absolute top-3 right-2 w-6 h-6 hover:bg-slate-950/5 rounded-full flex items-center justify-center cursor-pointer leading-none dark:hover:bg-gray-800/50"
              >
                <X size={16} />
              </div>
            </div>
          )}
        </div>
        <div className="relative flex-1 min-h-0 overflow-hidden">
          <div
            className={`absolute inset-0 flex flex-col bg-white transition-transform duration-300 ease-in-out dark:bg-gray-900 ${
              isDetailView ? "-translate-x-full" : "translate-x-0"
            } ${isDetailView ? "pointer-events-none" : "pointer-events-auto"}`}
          >
            <Tabs
              value={activeTab}
              onValueChange={(value) => onTabChange?.(value as "latest" | "settings" | "docs")}
              className={isBottomLayout ? "flex min-h-0 flex-1 flex-col overflow-hidden" : "flex-1"}
            >
              {showSettingsTab &&
                (isBottomLayout ? (
                  <div
                    className={cn(
                      "relative z-10 flex h-9 shrink-0 items-stretch overflow-visible border-b px-2",
                      appDarkModeClasses.sidebarEdge,
                    )}
                  >
                    {shouldShowRunsTab ? (
                      <BottomInspectorTabButton
                        active={activeTab === "latest"}
                        icon="rabbit"
                        label="Runs"
                        onClick={() => onTabChange?.("latest")}
                      />
                    ) : null}
                    <BottomInspectorTabButton
                      active={activeTab === "settings"}
                      icon="settings"
                      label="Configuration"
                      onClick={() => onTabChange?.("settings")}
                      trailing={
                        configurationHasError ? <span className="h-1.5 w-1.5 rounded-full bg-orange-500" /> : undefined
                      }
                    />
                    {!hideDocsTab ? (
                      <BottomInspectorTabButton
                        active={activeTab === "docs"}
                        icon="info"
                        label="Info"
                        onClick={() => onTabChange?.("docs")}
                      />
                    ) : null}
                  </div>
                ) : (
                  <div className="border-b border-slate-950/15 dark:border-gray-800/70">
                    <div className="flex px-4">
                      {shouldShowRunsTab && (
                        <button
                          onClick={() => onTabChange?.("latest")}
                          className={`py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors ${
                            activeTab === "latest"
                              ? "border-gray-700 text-gray-800 dark:text-indigo-300 dark:border-indigo-300"
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
                            ? "border-gray-700 text-gray-800 dark:text-indigo-300 dark:border-indigo-300"
                            : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                        }`}
                      >
                        Configuration
                        {configurationHasError ? <span className="w-1.5 h-1.5 bg-orange-500 rounded-full" /> : null}
                      </button>
                      {!hideDocsTab && (
                        <button
                          onClick={() => onTabChange?.("docs")}
                          className={`py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors ${
                            activeTab === "docs"
                              ? "border-gray-700 text-gray-800 dark:text-indigo-300 dark:border-indigo-300"
                              : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                          }`}
                        >
                          Info
                        </button>
                      )}
                    </div>
                  </div>
                ))}

              {shouldShowRunsTab && (
                <TabsContent
                  value="latest"
                  className={cn(
                    isBottomLayout
                      ? "mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
                      : !showSettingsTab
                        ? "overflow-y-auto"
                        : "mt-0",
                  )}
                  style={!isBottomLayout && !showSettingsTab ? { maxHeight: "40vh" } : undefined}
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
                    getTabData={getTabData}
                    onCancelQueueItem={onCancelQueueItem}
                    onCancelExecution={onCancelExecution}
                    onReEmit={onReEmit}
                    getExecutionState={getExecutionState}
                    compact={isBottomLayout}
                    selectionNodeId={nodeId}
                    resolveRunId={resolveRunId}
                    fetchRunId={fetchRunId}
                    onSelectRun={onSelectRun}
                  />
                </TabsContent>
              )}

              {showSettingsTab && (
                <TabsContent
                  value="settings"
                  className={cn("mt-0", isBottomLayout && "min-h-0 flex-1 overflow-y-auto")}
                >
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

              {showSettingsTab && !hideDocsTab && (
                <TabsContent
                  value="docs"
                  className={cn("mt-0", isBottomLayout ? "min-h-0 flex-1 overflow-y-auto" : "overflow-y-auto")}
                  style={!isBottomLayout ? { maxHeight: "calc(100vh - 160px)" } : undefined}
                >
                  <DocsTab
                    description={componentDescription}
                    examplePayload={componentExamplePayload}
                    payloadLabel={componentPayloadLabel}
                    documentationUrl={componentDocumentationUrl}
                    configurationFields={nodeConfigurationFields}
                  />
                </TabsContent>
              )}
            </Tabs>
          </div>

          <div
            className={`absolute inset-0 flex flex-col bg-white transition-transform duration-300 ease-in-out dark:bg-gray-900 ${
              isDetailView ? "translate-x-0" : "translate-x-full"
            } ${isDetailView ? "pointer-events-auto" : "pointer-events-none"}`}
          >
            {page !== "overview" && (
              <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
                <PageHeader onBackToOverview={handleBackToOverview} compact={isBottomLayout} />
                <HistoryQueuePage
                  page={page}
                  events={allEvents}
                  openEventIds={openEventIds}
                  onToggleOpen={handleToggleOpen}
                  onEventClick={onEventClick}
                  compact={isBottomLayout}
                  selectionNodeId={nodeId}
                  resolveRunId={resolveRunId}
                  fetchRunId={fetchRunId}
                  onSelectRun={onSelectRun}
                  onCancelQueueItem={onCancelQueueItem}
                  getTabData={getTabData}
                  onCancelExecution={onCancelExecution}
                  onReEmit={onReEmit}
                  getExecutionState={getExecutionState}
                  hasMoreItems={hasMoreItems}
                  loadingMoreItems={loadingMoreItems}
                  showMoreCount={showMoreCount}
                  onLoadMoreItems={handleLoadMoreItems}
                />
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

        <ConfigureIntegrationDialog
          integrationId={configureIntegrationId}
          organizationId={domainId ?? ""}
          onClose={handleCloseConfigureIntegrationDialog}
        />
      </div>
    </div>
  );
};
