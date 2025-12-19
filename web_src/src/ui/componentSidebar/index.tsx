/* eslint-disable @typescript-eslint/no-explicit-any */
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { resolveIcon } from "@/lib/utils";
import { Check, Copy, X } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { ChildEventsState } from "../composite";
import { SidebarActionsDropdown } from "./SidebarActionsDropdown";
import { TabData } from "./SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "./types";
import { LatestTab } from "./LatestTab";
import { SettingsTab } from "./SettingsTab";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import {
  AuthorizationDomainType,
  ComponentsAppInstallationRef,
  OrganizationsAppInstallation,
  ConfigurationField,
  WorkflowsWorkflowNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
} from "@/api-client";
import { EventState, EventStateMap } from "../componentBase";
import { NewNodeData } from "../CanvasPage";
import { ReactNode } from "react";
import { ExecutionChainPage, HistoryQueuePage, PageHeader } from "./pages";
import { mapTriggerEventToSidebarEvent } from "@/pages/workflowv2/utils";

const DEFAULT_STATUS_OPTIONS: { value: ChildEventsState; label: string }[] = [
  { value: "processed", label: "Processed" },
  { value: "discarded", label: "Failed" },
  { value: "running", label: "Running" },
];
interface ComponentSidebarProps {
  isOpen?: boolean;
  setIsOpen?: (isOpen: boolean) => void;

  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  nodeId?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
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
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Settings tab props
  showSettingsTab?: boolean;
  currentTab?: "latest" | "settings";
  onTabChange?: (tab: "latest" | "settings") => void;
  templateNodeId?: string | null;
  newNodeData: NewNodeData | null;
  onCancelTemplate?: () => void;
  nodeConfigMode?: "create" | "edit";
  nodeName?: string;
  nodeLabel?: string;
  nodeConfiguration?: Record<string, unknown>;
  nodeConfigurationFields?: ConfigurationField[];
  onNodeConfigSave?: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    appInstallationRef?: ComponentsAppInstallationRef,
  ) => void;
  onNodeConfigCancel?: () => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  customField?: (configuration: Record<string, unknown>) => ReactNode;
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
  installedApplications?: OrganizationsAppInstallation[];

  // Workflow metadata for ExecutionChainPage
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];

  // Highlighting callback for execution chain nodes
  onHighlightedNodesChange?: (nodeIds: Set<string>) => void;
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
  onRun,
  runDisabled,
  runDisabledTooltip,
  onDuplicate,
  onDocs,
  onEdit,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
  onReEmit,
  isCompactView = false,
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
  currentTab = "latest",
  onTabChange,
  templateNodeId,
  onCancelTemplate,
  newNodeData,
  nodeConfigMode = "edit",
  nodeName = "",
  nodeLabel,
  nodeConfiguration = {},
  nodeConfigurationFields = [],
  onNodeConfigSave,
  onNodeConfigCancel,
  domainId,
  domainType,
  customField,
  appName,
  appInstallationRef,
  installedApplications,
  workflowNodes = [],
  components = [],
  triggers = [],
  blueprints = [],
  onHighlightedNodesChange,
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
  const [previousPage, setPreviousPage] = useState<"overview" | "history" | "queue">("overview");
  const [executionChainEventId, setExecutionChainEventId] = useState<string | null>(null);
  const [executionChainTriggerEvent, setExecutionChainTriggerEvent] = useState<SidebarEvent | null>(null);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  // For template nodes, force settings tab and block latest tab
  const isTemplateNode = !!templateNodeId && !!newNodeData;
  const activeTab = isTemplateNode ? "settings" : currentTab || "latest";
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<ChildEventsState | "all">("all");
  const [justCopied, setJustCopied] = useState(false);

  const handleCopyNodeId = useCallback(async () => {
    if (nodeId) {
      await navigator.clipboard.writeText(nodeId);
      setJustCopied(true);
      setTimeout(() => setJustCopied(false), 1000);
    }
  }, [nodeId]);

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
    setPreviousPage(page as "overview" | "history" | "queue");
    setPage("history");
    onSeeFullHistory?.();
  }, [onSeeFullHistory, page]);

  const handleBackToOverview = useCallback(() => {
    if (page === "execution-chain") {
      // When coming back from execution chain, go to the previous page
      setPage(previousPage);
      // Clear highlights when leaving execution chain
      onHighlightedNodesChange?.(new Set());
    } else {
      setPage("overview");
    }
    setSearchQuery("");
    setStatusFilter("all");
    setExecutionChainEventId(null);
    setExecutionChainTriggerEvent(null);
    setSelectedExecutionId(null);
  }, [page, previousPage, onHighlightedNodesChange]);

  const handleSeeExecutionChain = useCallback(
    (eventId: string, triggerEvent?: SidebarEvent, selectedExecId?: string) => {
      setPreviousPage(page as "overview" | "history" | "queue");
      setExecutionChainEventId(eventId);
      setExecutionChainTriggerEvent(triggerEvent || null);
      setSelectedExecutionId(selectedExecId || null);
      setPage("execution-chain");
    },
    [page],
  );

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

  const statusOptions = React.useMemo(() => {
    const statuses = new Set<ChildEventsState>();
    allEvents.forEach((event) => {
      if (event.state) {
        statuses.add(event.state);
      }
    });
    return Array.from(statuses);
  }, [allEvents]);

  const extraStatusOptions = React.useMemo(
    () => statusOptions.filter((status) => !DEFAULT_STATUS_OPTIONS.some((option) => option.value === status)),
    [statusOptions],
  );

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

  if (!isOpen) return null;

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-20 overflow-hidden bg-white shadow-2xl flex flex-col"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`absolute left-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-blue-50 transition-colors flex items-center justify-center group ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-8px" }}
      >
        <div
          className={`w-2 h-14 rounded-full bg-gray-300 group-hover:bg-blue-500 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-1 border-border bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div className="flex justify-between gap-3 w-full">
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <div className={`h-7 rounded-full overflow-hidden flex items-center justify-center`}>
                  {iconSrc ? <img src={iconSrc} alt={nodeName} className="w-6 h-6" /> : <Icon size={16} />}
                </div>
                <h2 className="text-xl font-semibold">{isTemplateNode ? newNodeData.nodeName : nodeName}</h2>
              </div>
              {nodeId && !isTemplateNode && (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-gray-500 font-mono">{nodeId}</span>
                  <button
                    onClick={handleCopyNodeId}
                    className={"text-gray-400 hover:text-gray-600"}
                    title={justCopied ? "Copied!" : "Copy Node ID"}
                  >
                    {justCopied ? <Check size={14} /> : <Copy size={14} />}
                  </button>
                </div>
              )}
            </div>
            {!templateNodeId && (
              <div className="absolute top-14 right-[0.8rem]">
                <SidebarActionsDropdown
                  onRun={onRun}
                  runDisabled={runDisabled}
                  runDisabledTooltip={runDisabledTooltip}
                  onDuplicate={onDuplicate}
                  onDocs={onDocs}
                  onEdit={onEdit}
                  onConfigure={onConfigure}
                  onDeactivate={onDeactivate}
                  onToggleView={onToggleView}
                  onDelete={onDelete}
                  isCompactView={isCompactView}
                />
              </div>
            )}
          </div>
          <div
            onClick={() => onClose?.()}
            className="flex items-center justify-center absolute top-5 right-3 cursor-pointer"
          >
            <X size={18} />
          </div>
        </div>
      </div>
      {page !== "overview" ? (
        <div className="flex flex-col flex-1 min-h-0">
          <PageHeader
            page={page as "history" | "queue" | "execution-chain"}
            onBackToOverview={handleBackToOverview}
            previousPage={previousPage}
            showSearchAndFilter={page !== "execution-chain"}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            statusFilter={statusFilter}
            onStatusFilterChange={(value) => setStatusFilter(value as ChildEventsState | "all")}
            extraStatusOptions={extraStatusOptions}
          />

          <div className={`${page === "execution-chain" ? "flex flex-col flex-1 min-h-0" : "py-2 px-2 pb-3"}`}>
            {page === "execution-chain" ? (
              <ExecutionChainPage
                eventId={executionChainEventId}
                triggerEvent={executionChainTriggerEvent || undefined}
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
            ) : (
              <HistoryQueuePage
                page={page as "history" | "queue"}
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
                      const triggerEvent = mapTriggerEventToSidebarEvent(event.originalExecution?.rootEvent, node);
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
        </div>
      ) : (
        <>
          <Tabs
            value={activeTab}
            onValueChange={(value) => onTabChange?.(value as "latest" | "settings")}
            className="flex-1"
          >
            {showSettingsTab && (
              <div className="px-3">
                <div className="flex border-gray-200 dark:border-zinc-700">
                  <button
                    onClick={() => !isTemplateNode && onTabChange?.("latest")}
                    disabled={isTemplateNode}
                    className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                      isTemplateNode
                        ? "border-transparent text-gray-300 cursor-not-allowed dark:text-gray-600"
                        : activeTab === "latest"
                          ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
                          : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                    }`}
                  >
                    Latest
                  </button>
                  <button
                    onClick={() => onTabChange?.("settings")}
                    className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                      activeTab === "settings"
                        ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
                        : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                    }`}
                  >
                    Settings
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
                  mode={isTemplateNode ? "create" : nodeConfigMode}
                  nodeName={isTemplateNode ? newNodeData.nodeName : nodeName}
                  nodeLabel={isTemplateNode ? newNodeData.displayLabel : nodeLabel}
                  configuration={isTemplateNode ? newNodeData.configuration : nodeConfiguration}
                  configurationFields={
                    isTemplateNode
                      ? (newNodeData.buildingBlock.configuration as ConfigurationField[])
                      : nodeConfigurationFields
                  }
                  onSave={onNodeConfigSave || (() => {})}
                  onCancel={isTemplateNode ? onCancelTemplate : onNodeConfigCancel}
                  domainId={domainId}
                  domainType={domainType}
                  customField={customField}
                  appName={isTemplateNode ? newNodeData.appName : appName}
                  appInstallationRef={isTemplateNode ? newNodeData.appInstallationRef : appInstallationRef}
                  installedApplications={installedApplications}
                />
              </TabsContent>
            )}
          </Tabs>
        </>
      )}
    </div>
  );
};
