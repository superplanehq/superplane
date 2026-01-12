import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleX,
  MoreHorizontal,
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
  scope,
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

  const scopeTabs = useMemo<Array<{ id: LogScopeFilter; label: string }>>(
    () => [
      { id: "all", label: "All Events" },
      { id: "runs", label: "Runs" },
      { id: "canvas", label: "Canvas" },
    ],
    [],
  );

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
        className="bg-white border-t border-slate-300 flex flex-col"
        style={{ height: sidebarHeight, minHeight, maxHeight }}
      >
        <div
          className="h-0.5 cursor-row-resize rounded-t-lg hover:bg-slate-100 transition-colors"
          onMouseDown={handleResizeStart}
        />
        <div className="flex items-center justify-between px-4 border-b border-slate-200 h-8">
          <div className="flex items-center gap-4 -mb-2">
            {scopeTabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => onScopeChange(tab.id)}
                className={cn(
                  "pb-2.5 text-[13px] font-medium leading-none border-b transition-colors",
                  scope === tab.id
                    ? "border-gray-800 text-gray-800"
                    : "border-transparent text-gray-500 hover:text-gray-800",
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-2 pb-1">
            <Button
              variant="ghost"
              size="sm"
              className={cn(
                "h-7 px-2 text-xs",
                normalizedFilter.size === 0 || normalizedFilter.has("error")
                  ? "bg-rose-50 text-rose-600"
                  : "text-slate-500",
              )}
              onClick={() => {
                const nextFilter = new Set(normalizedFilter);
                const isChecked = normalizedFilter.size === 0 || normalizedFilter.has("error");
                if (isChecked) {
                  // If all are currently selected (empty set), unchecking one should leave the other two checked
                  if (normalizedFilter.size === 0) {
                    nextFilter.add("warning");
                    nextFilter.add("success");
                  } else {
                    nextFilter.delete("error");
                  }
                } else {
                  nextFilter.add("error");
                  // If all three are now selected, normalize to empty set
                  if (nextFilter.size === 3) {
                    nextFilter.clear();
                  }
                }
                onFilterChange(nextFilter);
              }}
            >
              <CircleX
                className={cn(
                  "h-3.5 w-3.5",
                  normalizedFilter.size === 0 || normalizedFilter.has("error") ? "text-rose-500" : "text-slate-400",
                )}
              />
              <span className="tabular-nums">{counts.error}</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className={cn(
                "h-7 px-2 text-xs",
                normalizedFilter.size === 0 || normalizedFilter.has("warning")
                  ? "bg-amber-50 text-amber-600"
                  : "text-slate-500",
              )}
              onClick={() => {
                const nextFilter = new Set(normalizedFilter);
                const isChecked = normalizedFilter.size === 0 || normalizedFilter.has("warning");
                if (isChecked) {
                  // If all are currently selected (empty set), unchecking one should leave the other two checked
                  if (normalizedFilter.size === 0) {
                    nextFilter.add("error");
                    nextFilter.add("success");
                  } else {
                    nextFilter.delete("warning");
                  }
                } else {
                  nextFilter.add("warning");
                  // If all three are now selected, normalize to empty set
                  if (nextFilter.size === 3) {
                    nextFilter.clear();
                  }
                }
                onFilterChange(nextFilter);
              }}
            >
              <TriangleAlert
                className={cn(
                  "h-3.5 w-3.5",
                  normalizedFilter.size === 0 || normalizedFilter.has("warning") ? "text-amber-500" : "text-slate-400",
                )}
              />
              <span className="tabular-nums">{counts.warning}</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className={cn(
                "h-7 px-2 text-xs",
                normalizedFilter.size === 0 || normalizedFilter.has("success")
                  ? "bg-emerald-50 text-emerald-600"
                  : "text-slate-500",
              )}
              onClick={() => {
                const nextFilter = new Set(normalizedFilter);
                const isChecked = normalizedFilter.size === 0 || normalizedFilter.has("success");
                if (isChecked) {
                  // If all are currently selected (empty set), unchecking one should leave the other two checked
                  if (normalizedFilter.size === 0) {
                    nextFilter.add("error");
                    nextFilter.add("warning");
                  } else {
                    nextFilter.delete("success");
                  }
                } else {
                  nextFilter.add("success");
                  // If all three are now selected, normalize to empty set
                  if (nextFilter.size === 3) {
                    nextFilter.clear();
                  }
                }
                onFilterChange(nextFilter);
              }}
            >
              <CircleCheck
                className={cn(
                  "h-3.5 w-3.5",
                  normalizedFilter.size === 0 || normalizedFilter.has("success")
                    ? "text-emerald-500"
                    : "text-slate-400",
                )}
              />
              <span className="tabular-nums">{counts.success}</span>
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <div className="px-2 border-b border-slate-200">
          <InputGroup className="h-7 border-0 shadow-none !ring-0 !focus-within:ring-0 focus-within:ring-offset-0">
            <InputGroupAddon className="border-0 shadow-none">
              <Search className="h-4 w-4 -ml-1 text-gray-800" />
            </InputGroupAddon>
            <InputGroupInput
              placeholder="Search through logs…"
              value={searchValue}
              onChange={(event) => onSearchChange(event.target.value)}
              className="h-7 !text-[13px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0"
            />
          </InputGroup>
        </div>
        <div className="flex-1 overflow-auto" data-log-scroll ref={scrollContainerRef}>
          {entries.length === 0 ? (
            <div className="px-4 py-6 text-sm text-slate-500">No logs match the current filters.</div>
          ) : (
            <div className="divide-y divide-slate-200">
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
    error: <CircleX className="h-4 w-4 text-rose-500" />,
    warning: <TriangleAlert className="h-4 w-4 text-amber-500" />,
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
          className="flex w-full items-center gap-3 px-4 py-1.5 text-sm text-gray-800 hover:bg-slate-50"
          aria-expanded={isExpanded}
        >
          <div className="h-4 w-4 rounded-full bg-slate-100 text-xs font-semibold text-slate-600 flex items-center justify-center">
            {runItems.length}
          </div>
          <div className="flex-1 min-w-0 text-left font-mono text-xs">{entry.title}</div>
          <span className="ml-auto text-xs text-gray-500 tabular-nums whitespace-nowrap">
            {formatLogTimestamp(entry.timestamp)}
          </span>
        </button>
        {showChildren && (
          <div className="pb-2">
            {runItems.map((item) => (
              <div
                key={item.id}
                className="flex items-start gap-3 px-10 pr-4 py-2 text-sm text-slate-700 bg-slate-50 hover:bg-slate-100 transition-colors"
              >
                <div className="pt-0.5">
                  {item.isRunning ? <MoreHorizontal className="h-4 w-4 text-slate-400" /> : icon[item.type]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <div className="flex-1 min-w-0 text-[13px]">{item.title}</div>
                    <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
                      {formatLogTimestamp(item.timestamp)}
                    </span>
                  </div>
                  {item.detail && <div className="mt-1 text-xs text-slate-500">{item.detail}</div>}
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
    <div className="flex items-start gap-3 px-4 py-1.5 text-sm text-gray-800">
      <div className="pt-0.5">{icon[entry.type]}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          {hasDetail ? (
            <button
              type="button"
              className="flex flex-1 min-w-0 items-center gap-2 text-left hover:text-slate-900"
              onClick={() => setIsDetailExpanded((prev) => !prev)}
              aria-expanded={isDetailExpanded}
            >
              {isDetailExpanded ? (
                <ChevronDown className="h-4 w-4 text-slate-500" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-500" />
              )}
              <div className="min-w-0">{entry.title}</div>
            </button>
          ) : (
            <div className="flex-1 min-w-0 text-[13px]">{entry.title}</div>
          )}
          <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
            {formatLogTimestamp(entry.timestamp)}
          </span>
        </div>
        {entry.detail && isDetailExpanded && <div className="mt-2 text-xs text-slate-500">{entry.detail}</div>}
      </div>
    </div>
  );
}
