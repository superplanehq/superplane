import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { CircleCheck, CircleX, MoreHorizontal, Search, TriangleAlert, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";

export type LogEntryType = "success" | "error" | "warning" | "run";
export type LogScope = "runs" | "canvas";
export type LogScopeFilter = "all" | LogScope;
export type LogTypeFilter = "all" | "success" | "error" | "warning";

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
    <aside className="absolute inset-x-3 bottom-3 z-31 pointer-events-auto">
      <div
        className="bg-white border border-slate-200 rounded-lg shadow-lg flex flex-col"
        style={{ height: sidebarHeight, minHeight, maxHeight }}
      >
        <div
          className="h-1 cursor-row-resize rounded-t-lg hover:bg-slate-100 transition-colors"
          onMouseDown={handleResizeStart}
        />
        <div className="flex items-center justify-between px-4 border-b border-slate-200">
          <div className="flex items-center gap-4">
            {scopeTabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => onScopeChange(tab.id)}
                className={cn(
                  "pb-2 text-sm font-medium border-b-2 transition-colors",
                  scope === tab.id
                    ? "border-slate-900 text-slate-900"
                    : "border-transparent text-slate-500 hover:text-slate-700",
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
              className={cn("h-7 px-2 text-xs", filter === "error" ? "bg-rose-50 text-rose-600" : "text-slate-500")}
              onClick={() => onFilterChange(filter === "error" ? "all" : "error")}
            >
              <CircleX className="h-3.5 w-3.5 text-rose-500" />
              <span className="tabular-nums">{counts.error}</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className={cn("h-7 px-2 text-xs", filter === "warning" ? "bg-amber-50 text-amber-600" : "text-slate-500")}
              onClick={() => onFilterChange(filter === "warning" ? "all" : "warning")}
            >
              <TriangleAlert className="h-3.5 w-3.5 text-amber-500" />
              <span className="tabular-nums">{counts.warning}</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className={cn(
                "h-7 px-2 text-xs",
                filter === "success" ? "bg-emerald-50 text-emerald-600" : "text-slate-500",
              )}
              onClick={() => onFilterChange(filter === "success" ? "all" : "success")}
            >
              <CircleCheck className="h-3.5 w-3.5 text-emerald-500" />
              <span className="tabular-nums">{counts.success}</span>
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <div className="px-4 py-3 border-b border-slate-200">
          <InputGroup className="bg-slate-50">
            <InputGroupAddon>
              <Search className="h-4 w-4" />
            </InputGroupAddon>
            <InputGroupInput
              placeholder="search through logs"
              value={searchValue}
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </InputGroup>
        </div>
        <div className="flex-1 overflow-auto" data-log-scroll ref={scrollContainerRef}>
          {entries.length === 0 ? (
            <div className="px-4 py-6 text-sm text-slate-500">No logs match the current filters.</div>
          ) : (
            <div className="divide-y divide-slate-100">
              {entries.map((entry) => (
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

  if (entry.type === "run") {
    const runItems = entry.runItems || [];
    const showChildren = isExpanded && runItems.length > 0;

    return (
      <div>
        <button
          type="button"
          onClick={() => onToggleRun(entry.id)}
          className="flex w-full items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
          aria-expanded={isExpanded}
        >
          <div className="h-4 w-4 rounded-full bg-slate-100 text-xs font-semibold text-slate-600 flex items-center justify-center">
            {runItems.length}
          </div>
          <div className="flex-1 min-w-0 text-left">{entry.title}</div>
          <span className="ml-auto text-xs text-slate-400 tabular-nums whitespace-nowrap">
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
                    <div className="flex-1 min-w-0">{item.title}</div>
                    <span className="text-xs text-slate-400 tabular-nums whitespace-nowrap">
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

  return (
    <div className="flex items-center gap-3 px-4 py-2 text-sm text-slate-700">
      <div>{icon[entry.type]}</div>
      <div className="flex-1 min-w-0">{entry.title}</div>
      <span className="ml-auto text-xs text-slate-400 tabular-nums whitespace-nowrap">
        {formatLogTimestamp(entry.timestamp)}
      </span>
    </div>
  );
}
