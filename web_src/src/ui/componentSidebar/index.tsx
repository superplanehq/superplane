/* eslint-disable @typescript-eslint/no-explicit-any */
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { resolveIcon } from "@/lib/utils";
import { ArrowLeft, Plus, Search, TextAlignStart, X } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { MetadataItem, MetadataList } from "../metadataList";
import { ChildEventsState } from "../composite";
import { SidebarActionsDropdown } from "./SidebarActionsDropdown";
import { SidebarEventItem } from "./SidebarEventItem";
import { TabData } from "./SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "./types";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import { EventState, EventStateMap } from "../componentBase";

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
  metadata: MetadataItem[];
  title: string;
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
}

export const ComponentSidebar = ({
  isOpen,
  metadata,
  title,
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
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
}: ComponentSidebarProps) => {
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);
  // Keep expanded state stable across parent re-renders
  const [openEventIds, setOpenEventIds] = useState<Set<string>>(new Set());

  const [page, setPage] = useState<"overview" | "history" | "queue">("overview");
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<ChildEventsState | "all">("all");

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
    setPage("queue");
    onSeeQueue?.();
  }, [onSeeQueue]);

  const handleSeeFullHistory = useCallback(() => {
    setPage("history");
    onSeeFullHistory?.();
  }, [onSeeFullHistory]);

  const handleBackToOverview = useCallback(() => {
    setPage("overview");
    setSearchQuery("");
    setStatusFilter("all");
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
          event.subtitle?.toLowerCase().includes(query) ||
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

  if (!isOpen) return null;

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white shadow-2xl"
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
          className={`w-1 h-12 rounded-full bg-gray-300 group-hover:bg-blue-500 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-1 border-border bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div
            className={`w-7 h-7 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ""}`}
          >
            {iconSrc ? <img src={iconSrc} alt={title} className="w-6 h-6" /> : <Icon size={16} className={iconColor} />}
          </div>
          <div className="flex justify-between gap-3 w-full">
            <h2 className="text-xl font-semibold">{title}</h2>
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
          <div
            onClick={() => onClose?.()}
            className="flex items-center justify-center absolute top-6 right-3 cursor-pointer"
          >
            <X size={18} />
          </div>
        </div>
      </div>
      {page !== "overview" ? (
        <>
          {/* Back to Overview Section */}
          <div className="px-3 py-2 border-b-1 border-border">
            <button
              onClick={handleBackToOverview}
              className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 cursor-pointer"
            >
              <ArrowLeft size={16} />
              Back to Overview
            </button>
          </div>

          {/* Full History Header with Search and Filter */}
          <div className="px-3 py-3">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-xs font-semibold uppercase text-gray-500">
                {page === "history" ? "Full History" : "Queue"}
              </h2>
            </div>
            <div className="flex gap-2">
              {/* Search Input */}
              <div className="relative flex-1">
                <Search size={16} className="absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-500" />
                <Input
                  type="text"
                  placeholder="Search events..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8 h-9 text-sm"
                />
              </div>
              {/* Status Filter */}
              <Select
                value={statusFilter}
                onValueChange={(value) => setStatusFilter(value as ChildEventsState | "all")}
              >
                <SelectTrigger className="w-[160px] h-9 text-sm text-gray-500">
                  <SelectValue placeholder="All Statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all" className="text-gray-500">
                    All Statuses
                  </SelectItem>
                  {DEFAULT_STATUS_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value} className="text-gray-500">
                      {option.label}
                    </SelectItem>
                  ))}
                  {extraStatusOptions.map((status) => (
                    <SelectItem key={status} value={status} className="text-gray-500">
                      {status.charAt(0).toUpperCase() + status.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Full History Events List */}
          <div className="px-3 py-1 pb-3">
            <div className="flex flex-col gap-2">
              {filteredHistoryEvents.length === 0 ? (
                <div className="text-center py-8 text-gray-500 text-sm">
                  {searchQuery || statusFilter !== "all" ? "No matching events found" : "No events found"}
                </div>
              ) : (
                <>
                  {filteredHistoryEvents.map((event, index) => (
                    <SidebarEventItem
                      key={event.id}
                      event={event}
                      index={index}
                      variant={page === "history" ? "latest" : "queue"}
                      isOpen={openEventIds.has(event.id) || event.isOpen}
                      onToggleOpen={handleToggleOpen}
                      onEventClick={onEventClick}
                      tabData={getTabData?.(event)}
                      onPushThrough={onPushThrough}
                      onCancelExecution={onCancelExecution}
                      supportsPushThrough={supportsPushThrough}
                      onReEmit={onReEmit}
                      loadExecutionChain={loadExecutionChain}
                      getExecutionState={getExecutionState}
                    />
                  ))}
                  {hasMoreItems && !searchQuery && statusFilter === "all" && (
                    <div className="flex justify-center pt-1">
                      <button
                        onClick={handleLoadMoreItems}
                        disabled={loadingMoreItems}
                        className="flex items-center gap-1 text-sm font-medium text-gray-500 hover:text-gray-800 disabled:text-gray-400 disabled:cursor-not-allowed rounded-md px-2 py-1.5 border border-border shadow-xs"
                      >
                        {loadingMoreItems ? null : <Plus size={16} />}
                        {loadingMoreItems ? "Loading..." : `Show ${showMoreCount > 10 ? "10" : showMoreCount} more`}
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </>
      ) : (
        // Overview (Original Content)
        <>
          {metadata.length > 0 && (
            <div className="px-3 py-1 border-b-1 border-border">
              <MetadataList
                items={metadata}
                className="border-b-0 text-gray-500 font-medium gap-1.5 flex flex-col py-2"
              />
            </div>
          )}
          <div className="px-3 py-1 border-b-1 border-border pb-3 text-left">
            <h2 className="text-xs font-semibold uppercase text-gray-500 my-2">Latest events</h2>
            <div className="flex flex-col gap-2">
              {latestEvents.length === 0 ? (
                <div className="text-center py-4 text-gray-500 text-sm">No events found</div>
              ) : (
                <>
                  {latestEvents.slice(0, 5).map((event, index) => {
                    return (
                      <SidebarEventItem
                        key={event.id}
                        event={event}
                        index={index}
                        variant="latest"
                        isOpen={openEventIds.has(event.id) || event.isOpen}
                        onToggleOpen={handleToggleOpen}
                        onEventClick={onEventClick}
                        tabData={getTabData?.(event)}
                        onPushThrough={onPushThrough}
                        onCancelExecution={onCancelExecution}
                        supportsPushThrough={supportsPushThrough}
                        onReEmit={onReEmit}
                        loadExecutionChain={loadExecutionChain}
                        getExecutionState={getExecutionState}
                      />
                    );
                  })}
                  {handleSeeFullHistory && (
                    <button
                      onClick={handleSeeFullHistory}
                      className="text-sm text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                    >
                      <TextAlignStart size={16} />
                      See full history
                    </button>
                  )}
                </>
              )}
            </div>
          </div>
          {!hideQueueEvents && (
            <div className="px-3 py-1 pb-3 text-left">
              <h2 className="text-xs font-semibold uppercase text-gray-500 my-2">Next in queue</h2>
              <div className="flex flex-col gap-2">
                {nextInQueueEvents.length === 0 ? (
                  <div className="text-center py-4 text-gray-500 text-sm">Queue is empty</div>
                ) : (
                  <>
                    {nextInQueueEvents.slice(0, 5).map((event, index) => {
                      return (
                        <SidebarEventItem
                          key={event.id}
                          event={event}
                          index={index}
                          variant="queue"
                          isOpen={openEventIds.has(event.id) || event.isOpen}
                          onToggleOpen={handleToggleOpen}
                          onEventClick={onEventClick}
                          tabData={getTabData?.(event)}
                          onCancelQueueItem={onCancelQueueItem}
                          onPushThrough={onPushThrough}
                          supportsPushThrough={supportsPushThrough}
                          loadExecutionChain={loadExecutionChain}
                          getExecutionState={getExecutionState}
                        />
                      );
                    })}
                    {totalInQueueCount > 5 && (
                      <button
                        onClick={handleSeeQueue}
                        className="text-xs font-medium text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                      >
                        <TextAlignStart size={16} />
                        {totalInQueueCount - 5} more in the queue
                      </button>
                    )}
                  </>
                )}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
