import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleX,
  MoreHorizontal,
  ScrollText,
  Search,
  TriangleAlert,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";

export type LogEntryType = "success" | "error" | "warning" | "run";
export type LogScope = "runs" | "canvas";
export type LogScopeFilter = "all" | LogScope;
export type LogTypeFilter = Set<"success" | "error" | "warning">;

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
  filter: LogTypeFilter;
  onFilterChange: (filter: LogTypeFilter) => void;
  height?: number;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  onHeightChange?: (height: number) => void;
  scope: LogScopeFilter;
  onScopeChange: (scope: LogScopeFilter) => void;
  searchValue: string;
  onSearchChange: (value: string) => void;
  entries: LogEntry[];
  counts: LogCounts;
  expandedRuns: Set<string>;
  onToggleRun: (runId: string) => void;
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
  filter,
  onFilterChange,
  height,
  defaultHeight = 320,
  minHeight = 240,
  maxHeight = 820,
  onHeightChange,
  onScopeChange,
  searchValue,
  onSearchChange,
  entries,
  counts,
  expandedRuns,
  onToggleRun,
}: CanvasLogSidebarProps) {
  const [internalHeight, setInternalHeight] = useState(defaultHeight);
  const dragStartRef = useRef<{ y: number; height: number } | null>(null);
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const stickToBottomRef = useRef(true);

  // Normalize filter: if all three types are selected, treat as empty set (show all)
  const normalizedFilter = useMemo(() => {
    if (filter.size === 3) {
      return new Set<"success" | "error" | "warning">();
    }
    return filter;
  }, [filter]);

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
      onScopeChange("all");
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

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!dragStartRef.current) return;
        const delta = dragStartRef.current.y - moveEvent.clientY;
        setSidebarHeight(dragStartRef.current.height + delta);
      };

      const handleMouseUp = () => {
        dragStartRef.current = null;
        document.body.style.userSelect = "";
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
      };

      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    },
    [setSidebarHeight, sidebarHeight],
  );

  if (!isOpen) {
    return null;
  }

  return (
    <aside className="absolute left-0 right-0 bottom-0 z-31 pointer-events-auto">
      <div
        className="bg-white dark:bg-gray-900 outline outline-slate-950/15 dark:outline-gray-700 flex flex-col"
        style={{ height: sidebarHeight, minHeight, maxHeight }}
      >
        <div className="h-0.5 cursor-row-resize rounded-t-lg transition-colors" onMouseDown={handleResizeStart} />
        <div className="flex items-center justify-between pl-4 pr-2 border-b border-gray-200 h-8">
          <div className="flex items-center gap-4 -mb-2">
            <button
              type="button"
              onClick={() => onFilterChange(new Set())}
              className={cn(
                "flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                normalizedFilter.size === 0
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <ScrollText className="h-4 w-4" />
              All Logs
            </button>
            <button
              type="button"
              onClick={() => onFilterChange(new Set(["error"]))}
              className={cn(
                "group flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                normalizedFilter.size === 1 && normalizedFilter.has("error")
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <CircleX
                className={cn(
                  "h-4 w-4",
                  counts.error > 0
                    ? "text-red-500"
                    : normalizedFilter.size === 1 && normalizedFilter.has("error")
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              />
              <span
                className={cn(
                  "tabular-nums",
                  counts.error > 0
                    ? "text-red-500"
                    : normalizedFilter.size === 1 && normalizedFilter.has("error")
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              >
                {counts.error}
              </span>
            </button>
            <button
              type="button"
              onClick={() => onFilterChange(new Set(["warning"]))}
              className={cn(
                "group flex items-center gap-2 pb-2 text-[13px] font-medium leading-none border-b transition-colors",
                normalizedFilter.size === 1 && normalizedFilter.has("warning")
                  ? "border-gray-800 text-gray-800"
                  : "border-transparent text-gray-500 hover:text-gray-800",
              )}
            >
              <TriangleAlert
                className={cn(
                  "h-4 w-4",
                  counts.warning > 0
                    ? "text-orange-500"
                    : normalizedFilter.size === 1 && normalizedFilter.has("warning")
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              />
              <span
                className={cn(
                  "tabular-nums",
                  counts.warning > 0
                    ? "text-orange-500"
                    : normalizedFilter.size === 1 && normalizedFilter.has("warning")
                      ? "text-gray-800"
                      : "text-gray-500 group-hover:text-gray-800",
                )}
              >
                {counts.warning}
              </span>
            </button>
          </div>
          <Button variant="ghost" size="icon-sm" onClick={onClose} className="size-5 rounded hover:bg-gray-100 -mt-0.5">
            <X className="h-3 w-3" />
          </Button>
        </div>
        <div className="px-2 border-b border-slate-200 h-8">
          <InputGroup className="h-8 border-0 shadow-none !ring-0 !focus-within:ring-0 focus-within:ring-offset-0">
            <InputGroupAddon className="border-0 shadow-none">
              <Search className="h-4 w-4 -ml-1 text-gray-500" />
            </InputGroupAddon>
            <InputGroupInput
              placeholder="Search through Logs…"
              value={searchValue}
              onChange={(event) => onSearchChange(event.target.value)}
              className="h-7 !text-[13px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0"
            />
          </InputGroup>
        </div>
        <div className="flex-1 overflow-auto" data-log-scroll ref={scrollContainerRef}>
          {entries.length === 0 ? (
            <div className="px-4 py-1.5 text-[13px] text-gray-800">No logs found.</div>
          ) : (
            <div className="divide-y divide-gray-200">
              {[...entries].reverse().map((entry) => (
                <LogEntryRow
                  key={entry.id}
                  entry={entry}
                  isExpanded={expandedRuns.has(entry.id)}
                  onToggleRun={onToggleRun}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}

function LogEntryRow({
  entry,
  isExpanded,
  onToggleRun,
}: {
  entry: LogEntry;
  isExpanded: boolean;
  onToggleRun: (runId: string) => void;
}) {
  const icon = {
    success: <CircleCheck className="h-4 w-4 text-emerald-500" />,
    error: <CircleX className="h-4 w-4 text-red-500" />,
    warning: <TriangleAlert className="h-4 w-4 text-amber-600" />,
  } as const;
  const [isDetailExpanded, setIsDetailExpanded] = useState(false);

  if (entry.type === "run") {
    const runItems = entry.runItems || [];
    const showChildren = isExpanded && runItems.length > 0;

    return (
      <div>
        <button
          type="button"
          onClick={() => onToggleRun(entry.id)}
          className="flex w-full items-center gap-3 px-4 py-1.5 text-sm text-gray-800 hover:bg-gray-50 min-h-8"
          aria-expanded={isExpanded}
        >
          <div className="h-4 w-4 rounded-full text-xs font-mono text-gray-500 flex items-center justify-center border border-gray-400">
            {runItems.length}
          </div>
          <div className="flex-1 min-w-0 text-left font-mono text-xs mt-0.5">{entry.title}</div>
          <span className="ml-auto text-xs text-gray-500 tabular-nums whitespace-nowrap">
            {formatLogTimestamp(entry.timestamp)}
          </span>
        </button>
        {showChildren && (
          <div>
            {runItems.map((item) => (
              <div
                key={item.id}
                className="flex items-start gap-3 px-11 pr-4 py-1.5 text-sm text-gray-800 bg-gray-50 border-t border-gray-200 transition-colors min-h-8"
              >
                <div className="pt-0.5">
                  {item.isRunning ? <MoreHorizontal className="h-4 w-4 text-gray-500" /> : icon[item.type]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <div className="flex-1 min-w-0 text-xs font-mono mt-0.5">{item.title}</div>
                    <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
                      {formatLogTimestamp(item.timestamp)}
                    </span>
                  </div>
                  {item.detail && <div className="mt-1 text-xs text-gray-800">{item.detail}</div>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  const hasDetail = Boolean(entry.detail);

  return (
    <div className="flex items-start gap-3 px-4 py-1.5 text-[13px] text-gray-800">
      <div className="pt-0.5">{icon[entry.type]}</div>
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
