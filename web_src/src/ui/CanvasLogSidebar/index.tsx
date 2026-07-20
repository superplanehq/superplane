import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ChevronDown, ChevronRight, CircleAlert, CircleX, Search, X } from "lucide-react";

import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { countUnacknowledgedErrors } from "@/pages/app/lib/canvas-runs";
import { ErrorsConsoleContent } from "@/pages/app/ErrorsConsoleContent";

export type ConsoleTab = "errors" | "warnings";
export type LogEntryType = "success" | "error" | "warning" | "resolved-error";
export type LogScope = "canvas";

export interface LogEntry {
  id: string;
  type: LogEntryType;
  title: ReactNode;
  timestamp: string;
  source: LogScope;
  searchText?: string;
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
  logRuns?: CanvasesCanvasRun[];
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: { runId: string; nodeId: string }) => void;
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
  logRuns = [],
  runsNodes = [],
  runsComponentIconMap = {},
  onRunNodeSelect,
  onRunExecutionSelect,
  onAcknowledgeErrors,
}: CanvasLogSidebarProps) {
  const [internalTab, setInternalTab] = useState<ConsoleTab>("errors");
  const activeTab = controlledTab ?? internalTab;
  const setActiveTab = useCallback(
    (tab: ConsoleTab) => {
      if (onTabChange) {
        onTabChange(tab);
        return;
      }
      setInternalTab(tab);
    },
    [onTabChange],
  );

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

  const unacknowledgedCount = useMemo(() => countUnacknowledgedErrors(logRuns), [logRuns]);

  if (!isOpen) {
    return null;
  }

  const searchPlaceholder = activeTab === "errors" ? "Search errors…" : "Search warnings…";

  return (
    <aside className="ph-no-capture absolute left-0 right-0 bottom-0 z-31 pointer-events-auto">
      <div
        className={cn("flex flex-col border-t bg-white dark:bg-gray-900", appDarkModeClasses.sidebarEdge)}
        style={{ height: sidebarHeight, minHeight, maxHeight }}
      >
        <div
          onMouseDown={handleResizeStart}
          className="group absolute left-0 right-0 top-0 z-30 h-4 cursor-row-resize bg-transparent"
          style={{ marginTop: "-8px" }}
        >
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute left-0 right-0 top-1/2 h-px -translate-y-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-500/50",
              isResizing && "bg-slate-950/50 dark:bg-gray-500/50",
            )}
          />
        </div>
        <div className={cn("flex items-center justify-between pl-4 border-b h-8", appDarkModeClasses.sidebarEdge)}>
          <div className="flex items-center gap-4 -mb-2">
            <button
              type="button"
              onClick={() => setActiveTab("errors")}
              className={cn(
                "group flex items-center gap-2 pb-2 !text-[13px] font-medium leading-none border-b transition-colors",
                activeTab === "errors"
                  ? "border-gray-800 text-gray-800 dark:border-indigo-300 dark:text-indigo-300"
                  : "border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100",
              )}
            >
              <CircleX
                className={cn(
                  "h-4 w-4",
                  unacknowledgedCount > 0
                    ? "text-red-500 dark:text-red-400"
                    : activeTab === "errors"
                      ? "text-gray-800 dark:text-indigo-300"
                      : "text-gray-500 group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
                )}
              />
              <span
                className={cn(
                  "tabular-nums !text-[13px]",
                  unacknowledgedCount > 0
                    ? "text-red-500 dark:text-red-400"
                    : activeTab === "errors"
                      ? "text-gray-800 dark:text-indigo-300"
                      : "text-gray-500 group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
                )}
              >
                {unacknowledgedCount}
              </span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("warnings")}
              className={cn(
                "group flex items-center gap-2 pb-2 !text-[13px] font-medium leading-none border-b transition-colors",
                activeTab === "warnings"
                  ? "border-gray-800 text-gray-800 dark:border-indigo-300 dark:text-indigo-300"
                  : "border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100",
              )}
            >
              <CircleAlert
                className={cn(
                  "h-4 w-4",
                  counts.warning > 0
                    ? "text-orange-500 dark:text-orange-300"
                    : activeTab === "warnings"
                      ? "text-gray-800 dark:text-indigo-300"
                      : "text-gray-500 group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
                )}
              />
              <span
                className={cn(
                  "tabular-nums !text-[13px]",
                  counts.warning > 0
                    ? "text-orange-500 dark:text-orange-300"
                    : activeTab === "warnings"
                      ? "text-gray-800 dark:text-indigo-300"
                      : "text-gray-500 group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
                )}
              >
                {counts.warning}
              </span>
            </button>
          </div>
          <div className="flex shrink-0 items-stretch">
            <div className="flex items-center px-1">
              <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
                <X className="size-3.5" />
              </Button>
            </div>
          </div>
        </div>
        <div className={cn("px-2 h-8 border-b", appDarkModeClasses.sidebarEdge)}>
          <InputGroup className="h-8 border-0 bg-transparent shadow-none !ring-0 !focus-within:ring-0 focus-within:ring-offset-0 dark:bg-transparent [&_[data-slot=input-group-control]]:!text-[13px]">
            <InputGroupAddon className="border-0 shadow-none !text-[13px]">
              <Search className="h-4 w-4 -ml-1 text-gray-500 dark:text-gray-400" />
            </InputGroupAddon>
            <InputGroupInput
              placeholder={searchPlaceholder}
              value={searchValue}
              onChange={(event) => onSearchChange(event.target.value)}
              className="h-7 !text-[13px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0 dark:text-gray-100 dark:placeholder:text-gray-500"
            />
          </InputGroup>
        </div>

        {activeTab === "errors" ? (
          <ErrorsConsoleContent
            runs={logRuns}
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
              <div className="px-4 py-1.5 text-[13px] text-gray-800 dark:text-gray-100">No warnings found.</div>
            ) : (
              <div className="divide-y divide-gray-200 dark:divide-gray-800/50">
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
    <div className="flex items-start gap-3 px-4 py-1.5 text-[13px] text-gray-800 dark:text-gray-100">
      <div className="pt-0.5">
        <CircleAlert className="h-4 w-4 text-orange-500 dark:text-orange-300" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          {hasDetail ? (
            <button
              type="button"
              className="flex flex-1 min-w-0 items-center gap-2 text-left hover:text-gray-800 dark:hover:text-gray-100"
              onClick={() => setIsDetailExpanded((prev) => !prev)}
              aria-expanded={isDetailExpanded}
            >
              {isDetailExpanded ? (
                <ChevronDown className="h-4 w-4 text-gray-500 dark:text-gray-400" />
              ) : (
                <ChevronRight className="h-4 w-4 text-gray-500 dark:text-gray-400" />
              )}
              <div className="min-w-0">{entry.title}</div>
            </button>
          ) : (
            <div className="flex-1 min-w-0 text-xs font-mono mt-0.5">{entry.title}</div>
          )}
          <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap dark:text-gray-400">
            {formatLogTimestamp(entry.timestamp)}
          </span>
        </div>
        {entry.detail && isDetailExpanded && (
          <div className="mt-2 text-[13px] text-gray-500 dark:text-gray-400">{entry.detail}</div>
        )}
      </div>
    </div>
  );
}
