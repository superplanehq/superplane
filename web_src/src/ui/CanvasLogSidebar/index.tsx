import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ChevronDown, ChevronRight, CircleX, Play, Search, TriangleAlert, X } from "lucide-react";

import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeQueueItem, ComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";
import { countUnacknowledgedErrors } from "@/pages/workflowv2/canvasRunsUtils";
import { ErrorsConsoleContent } from "@/pages/workflowv2/ErrorsConsoleContent";
import { RunsConsoleContent } from "@/pages/workflowv2/CanvasRunsView";
import type { SidebarEvent } from "@/ui/componentSidebar/types";

export type ConsoleTab = "runs" | "errors" | "warnings";
export type LogEntryType = "success" | "error" | "warning" | "resolved-error" | "run";
export type LogScope = "runs" | "canvas";

export interface LogRunItem {
  id: string;
  type: Exclude<LogEntryType, "run">;
  title: ReactNode;
  timestamp: string;
  detail?: ReactNode;
  searchText?: string;
  isRunning?: boolean;
}

export interface LogEntry {
  id: string;
  type: LogEntryType;
  title: ReactNode;
  timestamp: string;
  source: LogScope;
  searchText?: string;
  runItems?: LogRunItem[];
  detail?: ReactNode;
}

export interface LogCounts {
  total: number;
  error: number;
  warning: number;
  success: number;
}

export interface CanvasLogSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  height?: number;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  onHeightChange?: (height: number) => void;
  searchValue: string;
  onSearchChange: (value: string) => void;
  entries: LogEntry[];
  counts: LogCounts;
  activeTab?: ConsoleTab;
  onTabChange?: (tab: ConsoleTab) => void;
  runsEvents?: CanvasesCanvasEventWithExecutions[];
  runsTotalCount?: number;
  runsHasNextPage?: boolean;
  runsIsFetchingNextPage?: boolean;
  onRunsLoadMore?: () => void;
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  runsNodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
}

function formatLogTimestamp(value: string) {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }

  const date = new Date(parsed);
  const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const weekday = weekdays[date.getDay()];
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  return `${weekday} ${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

export function CanvasLogSidebar({
  isOpen,
  onClose,
  height,
  defaultHeight = 320,
  minHeight = 240,
  maxHeight = 820,
  onHeightChange,
  searchValue,
  onSearchChange,
  entries,
  counts,
  activeTab: controlledTab,
  onTabChange,
  runsEvents = [],
  runsTotalCount,
  runsHasNextPage,
  runsIsFetchingNextPage,
  onRunsLoadMore,
  runsNodes = [],
  runsComponentIconMap = {},
  runsNodeQueueItemsMap = {},
  onRunNodeSelect,
  onRunExecutionSelect,
  onAcknowledgeErrors,
}: CanvasLogSidebarProps) {
  const [internalTab, setInternalTab] = useState<ConsoleTab>("runs");
  const activeTab = controlledTab ?? internalTab;
  const setActiveTab = onTabChange ?? setInternalTab;

  const [internalHeight, setInternalHeight] = useState(defaultHeight);
  const [isResizing, setIsResizing] = useState(false);
  const dragStartRef = useRef<{ y: number; height: number } | null>(null);
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const stickToBottomRef = useRef(true);

  const filteredEntries = useMemo(() => {
    const query = searchValue.trim().toLowerCase();
    const matchesSearch = (value?: string) => !query || (value || "").toLowerCase().includes(query);

    return entries.reduce<LogEntry[]>((acc, entry) => {
      if (entry.type !== "warning") {
        return acc;
      }

      const entrySearchMatch =
        matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : "");
      if (entrySearchMatch) {
        acc.push(entry);
      }
      return acc;
    }, []);
  }, [entries, searchValue]);

  const sidebarHeight = height ?? internalHeight;
  const clampHeight = useCallback(
    (value: number) => {
      const overrideMaxHeight = Math.min(document.body.clientHeight - 100, maxHeight);
      return Math.max(minHeight, Math.min(overrideMaxHeight, value));
    },
    [minHeight, maxHeight],
  );

  const setSidebarHeight = useCallback(
    (value: number) => {
      const nextHeight = clampHeight(value);
      if (height === undefined) {
        setInternalHeight(nextHeight);
      }
      onHeightChange?.(nextHeight);
    },
    [clampHeight, height, onHeightChange],
  );

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const container = scrollContainerRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = container.scrollHeight;
  }, [isOpen]);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) {
      return;
    }

    const handleScroll = () => {
      const threshold = 16;
      const { scrollTop, scrollHeight, clientHeight } = container;
      stickToBottomRef.current = scrollHeight - scrollTop - clientHeight <= threshold;
    };

    container.addEventListener("scroll", handleScroll);
    handleScroll();

    return () => {
      container.removeEventListener("scroll", handleScroll);
    };
  }, []);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container || !isOpen) {
      return;
    }

    const threshold = 40;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const isAtBottom = scrollHeight - scrollTop - clientHeight <= threshold;
    stickToBottomRef.current = isAtBottom;

    if (isAtBottom) {
      container.scrollTop = container.scrollHeight;
    }
  }, [entries, isOpen]);

  const handleResizeStart = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      dragStartRef.current = { y: event.clientY, height: sidebarHeight };
      document.body.style.userSelect = "none";
      document.body.style.cursor = "ns-resize";
      setIsResizing(true);

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!dragStartRef.current) return;
        const delta = dragStartRef.current.y - moveEvent.clientY;
        setSidebarHeight(dragStartRef.current.height + delta);
      };

      const handleMouseUp = () => {
        dragStartRef.current = null;
        document.body.style.userSelect = "";
        document.body.style.cursor = "";
        setIsResizing(false);
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
      };

      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    },
    [setSidebarHeight, sidebarHeight],
  );

  const unacknowledgedCount = useMemo(() => countUnacknowledgedErrors(runsEvents), [runsEvents]);

  if (!isOpen) {
    return null;
  }

  const searchPlaceholder =
    activeTab === "runs" ? "Search runs…" : activeTab === "errors" ? "Search errors…" : "Search warnings…";

  return (
    <aside className="absolute left-0 right-0 bottom-0 z-31 pointer-events-auto">
      <div
        className="bg-white outline outline-slate-950/15 flex flex-col"
        style={{ height: sidebarHeight, minHeight, maxHeight }}
      >
        <div
          onMouseDown={handleResizeStart}
          className={cn(
            "absolute left-0 right-0 top-0 h-4 cursor-ns-resize hover:bg-gray-100 transition-colors flex items-center justify-center group z-30",
            isResizing && "bg-blue-50",
          )}
          style={{ marginTop: "-8px" }}
        >
          <div
            className={cn(
              "h-1 w-14 rounded-full bg-gray-300 group-hover:bg-gray-800 transition-colors",
              isResizing && "bg-blue-500",
            )}
          />
        </div>
        <div className="flex items-center justify-between pl-4 pr-2 border-b border-gray-200 h-8">
          <div className="flex items-center gap-4 -mb-2">
            <button
              type="button"
              onClick={() => setActiveTab("runs")}
              className={cn(
                "flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                activeTab === "runs"
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <Play className="h-4 w-4" />
              Runs
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("errors")}
              className={cn(
                "group flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                activeTab === "errors"
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <CircleX
                className={cn(
                  "h-4 w-4",
                  unacknowledgedCount > 0
                    ? "text-red-500"
                    : activeTab === "errors"
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              />
              <span
                className={cn(
                  "tabular-nums",
                  unacknowledgedCount > 0
                    ? "text-red-500"
                    : activeTab === "errors"
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              >
                {unacknowledgedCount}
              </span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("warnings")}
              className={cn(
                "group flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                activeTab === "warnings"
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <TriangleAlert
                className={cn(
                  "h-4 w-4",
                  counts.warning > 0
                    ? "text-orange-500"
                    : activeTab === "warnings"
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              />
              <span
                className={cn(
                  "tabular-nums",
                  counts.warning > 0
                    ? "text-orange-500"
                    : activeTab === "warnings"
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              >
                {counts.warning}
              </span>
            </button>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onClose}
              className="size-5 rounded hover:bg-gray-100 -mt-0.5"
            >
              <X className="h-3 w-3" />
            </Button>
          </div>
        </div>
        <div className="px-2 border-b border-slate-200 h-8">
          <InputGroup className="h-8 border-0 shadow-none !ring-0 !focus-within:ring-0 focus-within:ring-offset-0">
            <InputGroupAddon className="border-0 shadow-none">
              <Search className="h-4 w-4 -ml-1 text-gray-500" />
            </InputGroupAddon>
            <InputGroupInput
              placeholder={searchPlaceholder}
              value={searchValue}
              onChange={(event) => onSearchChange(event.target.value)}
              className="h-7 !text-[13px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0"
            />
          </InputGroup>
        </div>

        {activeTab === "runs" ? (
          <RunsConsoleContent
            events={runsEvents}
            totalCount={runsTotalCount}
            hasNextPage={runsHasNextPage}
            isFetchingNextPage={runsIsFetchingNextPage}
            onLoadMore={onRunsLoadMore}
            nodes={runsNodes}
            componentIconMap={runsComponentIconMap}
            searchQuery={searchValue}
            nodeQueueItemsMap={runsNodeQueueItemsMap}
            onNodeSelect={onRunNodeSelect}
            onExecutionSelect={onRunExecutionSelect}
          />
        ) : activeTab === "errors" ? (
          <ErrorsConsoleContent
            events={runsEvents}
            nodes={runsNodes}
            componentIconMap={runsComponentIconMap}
            searchQuery={searchValue}
            onNodeSelect={onRunNodeSelect}
            onExecutionSelect={onRunExecutionSelect}
            onAcknowledgeErrors={onAcknowledgeErrors}
          />
        ) : (
          <div className="flex-1 overflow-auto" data-log-scroll ref={scrollContainerRef}>
            {filteredEntries.length === 0 ? (
              <div className="px-4 py-1.5 text-[13px] text-gray-800">No warnings found.</div>
            ) : (
              <div className="divide-y divide-gray-200">
                {filteredEntries.map((entry) => (
                  <LogEntryRow key={entry.id} entry={entry} />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </aside>
  );
}

function LogEntryRow({ entry }: { entry: LogEntry }) {
  const [isDetailExpanded, setIsDetailExpanded] = useState(false);
  const hasDetail = Boolean(entry.detail);

  return (
    <div className="flex items-start gap-3 px-4 py-1.5 text-[13px] text-gray-800">
      <div className="pt-0.5">
        <TriangleAlert className="h-4 w-4 text-amber-600" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          {hasDetail ? (
            <button
              type="button"
              className="flex flex-1 min-w-0 items-center gap-2 text-left hover:text-gray-800"
              onClick={() => setIsDetailExpanded((prev) => !prev)}
              aria-expanded={isDetailExpanded}
            >
              {isDetailExpanded ? (
                <ChevronDown className="h-4 w-4 text-gray-500" />
              ) : (
                <ChevronRight className="h-4 w-4 text-gray-500" />
              )}
              <div className="min-w-0">{entry.title}</div>
            </button>
          ) : (
            <div className="flex-1 min-w-0 text-xs font-mono mt-0.5">{entry.title}</div>
          )}
          <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
            {formatLogTimestamp(entry.timestamp)}
          </span>
        </div>
        {entry.detail && isDetailExpanded && <div className="mt-2 text-[13px] text-gray-500">{entry.detail}</div>}
      </div>
    </div>
  );
}
