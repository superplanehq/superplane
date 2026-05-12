/* eslint-disable max-lines-per-function, complexity */
import type React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { Checkbox } from "@/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import {
  buildNodeMap,
  buildRunPresentation,
  RUN_RESULT_FILTER_OPTIONS,
  RUN_STATUS_META,
  type RunResultFilter,
} from "@/ui/Runs/runPresentation";
import { RunNodeIcon } from "@/ui/Runs/RunNodeIcon";
import { Filter, Link as LinkIcon, Loader2, Search, X } from "lucide-react";
import { toast } from "sonner";

export const RUNS_SIDEBAR_WIDTH_STORAGE_KEY = "runs-sidebar-width";
const RUNS_SIDEBAR_FILTERS_STORAGE_KEY = "runs-sidebar-filters";
const RUNS_SIDEBAR_MIN_WIDTH = 280;
const RUNS_SIDEBAR_MAX_WIDTH = 640;
const RUNS_SIDEBAR_DEFAULT_WIDTH = 340;

interface RunsSidebarProps {
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  totalCount?: number;
  onResultFiltersChange?: (filters: RunResultFilter[]) => void;
}

function loadPersistedFilters(): { statuses: Set<RunResultFilter>; triggerIds: Set<string> } {
  if (typeof window === "undefined") return { statuses: new Set(), triggerIds: new Set() };

  try {
    const raw = window.localStorage.getItem(RUNS_SIDEBAR_FILTERS_STORAGE_KEY);
    if (!raw) return { statuses: new Set(), triggerIds: new Set() };

    const parsed = JSON.parse(raw) as { statuses?: unknown; triggerIds?: unknown };
    const validStatuses = new Set<RunResultFilter>(RUN_RESULT_FILTER_OPTIONS.map((option) => option.id));
    const statuses = new Set<RunResultFilter>(
      Array.isArray(parsed.statuses)
        ? parsed.statuses.filter(
            (status: unknown): status is RunResultFilter =>
              typeof status === "string" && validStatuses.has(status as RunResultFilter),
          )
        : [],
    );
    const triggerIds = new Set<string>(
      Array.isArray(parsed.triggerIds)
        ? parsed.triggerIds.filter((triggerId: unknown): triggerId is string => typeof triggerId === "string")
        : [],
    );

    return { statuses, triggerIds };
  } catch {
    return { statuses: new Set(), triggerIds: new Set() };
  }
}

export function RunsSidebar({
  runs,
  selectedRunId,
  onSelectRun,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  isLoading,
  workflowNodes = [],
  componentIconMap = {},
  totalCount,
  onResultFiltersChange,
}: RunsSidebarProps) {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = typeof window !== "undefined" ? localStorage.getItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY) : null;
    const parsed = saved ? parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) return RUNS_SIDEBAR_DEFAULT_WIDTH;
    return Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);
  const [search, setSearch] = useState("");
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(() => loadPersistedFilters().triggerIds);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<RunResultFilter>>(() => loadPersistedFilters().statuses);
  const [isFilterOpen, setIsFilterOpen] = useState(false);

  const nodeMap = useMemo(() => {
    return buildNodeMap(workflowNodes);
  }, [workflowNodes]);

  const triggerOptions = useMemo(() => {
    return workflowNodes
      .filter((node) => node.id && node.type === "TYPE_TRIGGER")
      .map((node) => ({
        id: node.id!,
        name: node.name || node.component || "Trigger",
        iconSrc: getHeaderIconSrc(node.component),
        iconSlug: node.component ? componentIconMap[node.component] : undefined,
      }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [workflowNodes, componentIconMap]);

  useEffect(() => {
    localStorage.setItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    onResultFiltersChange?.(Array.from(selectedStatuses));

    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(
        RUNS_SIDEBAR_FILTERS_STORAGE_KEY,
        JSON.stringify({
          statuses: Array.from(selectedStatuses),
          triggerIds: Array.from(selectedTriggerIds),
        }),
      );
    } catch {
      // Filter persistence is optional.
    }
  }, [selectedStatuses, selectedTriggerIds, onResultFiltersChange]);

  useEffect(() => {
    if (triggerOptions.length === 0) return;
    const valid = new Set(triggerOptions.map((option) => option.id));
    setSelectedTriggerIds((prev) => {
      const next = new Set(Array.from(prev).filter((id) => valid.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [triggerOptions]);

  const handleMouseDown = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      const nextWidth = Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, event.clientX - left));
      setSidebarWidth(nextWidth);
    };

    const handleMouseUp = () => setIsResizing(false);

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
  }, [isResizing]);

  const decoratedRuns = useMemo(() => {
    return runs.map((run) => {
      return buildRunPresentation(run, nodeMap);
    });
  }, [runs, nodeMap]);

  const filteredRuns = useMemo(() => {
    const query = search.trim().toLowerCase();
    return decoratedRuns.filter(({ run, status, haystack }) => {
      if (query && !haystack.includes(query)) return false;
      if (selectedStatuses.size > 0) {
        if (status === "running" || status === "unknown" || !selectedStatuses.has(status)) {
          return false;
        }
      }
      if (selectedTriggerIds.size > 0) {
        const triggerNodeId = run.rootEvent?.nodeId;
        if (!triggerNodeId || !selectedTriggerIds.has(triggerNodeId)) return false;
      }
      return true;
    });
  }, [decoratedRuns, search, selectedStatuses, selectedTriggerIds]);

  const orderedRuns = useMemo(() => {
    const active = filteredRuns.filter((run) => run.status === "running");
    const rest = filteredRuns.filter((run) => run.status !== "running");
    return { active, rest };
  }, [filteredRuns]);

  const hasSearch = search.trim().length > 0;
  const hasTriggerFilter = selectedTriggerIds.size > 0;
  const hasStatusFilter = selectedStatuses.size > 0;
  const hasAnyFilter = hasSearch || hasTriggerFilter || hasStatusFilter;

  const clearFilters = useCallback(() => {
    setSearch("");
    setSelectedStatuses(new Set());
    setSelectedTriggerIds(new Set());
  }, []);

  const renderRow = (item: (typeof decoratedRuns)[number]) => {
    const { run, triggerName, title, status, triggerNode } = item;
    const isSelected = run.id === selectedRunId;
    const iconSrc = getHeaderIconSrc(triggerNode?.component);
    const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;

    return (
      <div
        key={run.id}
        data-testid="runs-sidebar-row"
        role="button"
        tabIndex={0}
        onClick={() => run.id && onSelectRun(run.id)}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            if (run.id) {
              onSelectRun(run.id);
            }
          }
        }}
        className={cn(
          "group flex w-full cursor-pointer items-center gap-1.5 border-b border-l-2 border-slate-100 px-3 py-2 text-left transition-colors",
          status === "failed" ? "border-l-red-400" : "border-l-transparent",
          isSelected ? "border-l-sky-500 bg-sky-100" : "hover:bg-gray-50",
        )}
      >
        <RunNodeIcon
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          alt={triggerName}
          size={14}
          className="shrink-0 text-gray-400"
        />
        <span
          className={cn(
            "max-w-[35%] shrink-0 truncate rounded px-1.5 py-0.5 text-[10px] font-medium",
            isSelected ? "bg-sky-200 text-sky-800" : "bg-slate-100 text-slate-600",
          )}
        >
          {triggerName}
        </span>
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-xs",
            isSelected ? "font-semibold text-sky-900" : "font-medium text-gray-800",
          )}
        >
          {title}
        </span>
        <span
          aria-label={RUN_STATUS_META[status].label}
          title={RUN_STATUS_META[status].label}
          className={cn("inline-block h-2 w-2 shrink-0 rounded-full", RUN_STATUS_META[status].dotClassName)}
        />
        <button
          type="button"
          title="Copy link to run"
          className="hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:inline-flex"
          onClick={(event) => {
            event.stopPropagation();
            const url = new URL(window.location.href);
            url.searchParams.set("view", "runs");
            url.searchParams.set("run", run.id || "");
            navigator.clipboard.writeText(url.toString());
            toast.success("Run link copied");
          }}
        >
          <LinkIcon className="h-3 w-3" />
        </button>
        {run.createdAt ? (
          <span className="shrink-0 text-[10px] tabular-nums text-gray-400">
            <TimeAgo date={run.createdAt} />
          </span>
        ) : null}
      </div>
    );
  };

  return (
    <div
      ref={sidebarRef}
      data-testid="runs-sidebar"
      className="relative flex shrink-0 flex-col border-r border-slate-200 bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
        <span className="text-sm font-medium text-gray-700">Runs</span>
        {totalCount != null && totalCount > 0 ? (
          <span className="ml-1.5 text-xs text-gray-400">({totalCount})</span>
        ) : null}
      </div>

      <div className="flex shrink-0 items-center gap-1.5 border-b border-slate-200 px-2 py-1.5">
        <Popover open={isFilterOpen} onOpenChange={setIsFilterOpen}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn(
                "relative h-7 w-7 shrink-0 p-0",
                (hasTriggerFilter || hasStatusFilter) && "bg-sky-50 text-sky-700 hover:bg-sky-100",
              )}
              aria-label="Filter runs"
              title="Filter runs"
            >
              <Filter className="h-3.5 w-3.5" />
              {hasTriggerFilter || hasStatusFilter ? (
                <span className="absolute -right-0.5 -top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-sky-500 px-1 text-[9px] font-semibold text-white">
                  {selectedTriggerIds.size + selectedStatuses.size}
                </span>
              ) : null}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-64 p-0" sideOffset={6}>
            <div className="flex items-center justify-between border-b border-slate-200 px-3 py-2">
              <span className="text-[12px] font-medium text-gray-700">Filter by status</span>
              <button
                type="button"
                onClick={() => setSelectedStatuses(new Set())}
                disabled={!hasStatusFilter}
                className={cn("text-[11px]", hasStatusFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-300")}
              >
                Clear
              </button>
            </div>
            <div className="py-1">
              {RUN_RESULT_FILTER_OPTIONS.map((option) => (
                <label
                  key={option.id}
                  className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
                >
                  <Checkbox
                    checked={selectedStatuses.has(option.id)}
                    onCheckedChange={() => {
                      setSelectedStatuses((prev) => {
                        const next = new Set(prev);
                        if (next.has(option.id)) next.delete(option.id);
                        else next.add(option.id);
                        return next;
                      });
                    }}
                    className="h-3.5 w-3.5"
                  />
                  <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", option.dotClassName)} />
                  <span className="min-w-0 truncate">{option.label}</span>
                </label>
              ))}
            </div>

            <div className="flex items-center justify-between border-t border-slate-100 px-3 py-2">
              <span className="text-[12px] font-medium text-gray-700">Filter by trigger</span>
              <button
                type="button"
                onClick={() => setSelectedTriggerIds(new Set())}
                disabled={!hasTriggerFilter}
                className={cn("text-[11px]", hasTriggerFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-300")}
              >
                Clear
              </button>
            </div>
            <div className="max-h-64 overflow-y-auto py-1">
              {triggerOptions.length === 0 ? (
                <div className="px-3 py-4 text-center text-[11px] text-gray-400">No triggers in this canvas</div>
              ) : (
                triggerOptions.map((option) => (
                  <label
                    key={option.id}
                    className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
                  >
                    <Checkbox
                      checked={selectedTriggerIds.has(option.id)}
                      onCheckedChange={() => {
                        setSelectedTriggerIds((prev) => {
                          const next = new Set(prev);
                          if (next.has(option.id)) next.delete(option.id);
                          else next.add(option.id);
                          return next;
                        });
                      }}
                      className="h-3.5 w-3.5"
                    />
                    <RunNodeIcon
                      iconSrc={option.iconSrc}
                      iconSlug={option.iconSlug}
                      alt={option.name}
                      size={12}
                      className="shrink-0 text-gray-400"
                    />
                    <span className="min-w-0 truncate">{option.name}</span>
                  </label>
                ))
              )}
            </div>
          </PopoverContent>
        </Popover>

        <InputGroup className="h-7 flex-1 border border-slate-200 shadow-none !ring-0 focus-within:!ring-0 focus-within:ring-offset-0 [&_[data-slot=input-group-control]]:!text-[12px]">
          <InputGroupAddon className="!text-[12px]">
            <Search className="h-3.5 w-3.5 text-gray-500" />
          </InputGroupAddon>
          <InputGroupInput
            placeholder="Search runs..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="h-6 !text-[12px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0"
          />
          {hasSearch ? (
            <InputGroupAddon>
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => setSearch("")}
                className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
              >
                <X className="h-3 w-3" />
              </button>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
          </div>
        ) : runs.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-gray-400">No runs yet</div>
        ) : filteredRuns.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-gray-400">
            <span>No runs match your filters</span>
            <button type="button" onClick={clearFilters} className="text-[11px] text-sky-600 hover:text-sky-800">
              Clear filters
            </button>
          </div>
        ) : (
          <>
            {orderedRuns.active.map(renderRow)}
            {orderedRuns.active.length > 0 && orderedRuns.rest.length > 0 ? (
              <div className="h-px bg-slate-300" aria-hidden />
            ) : null}
            {orderedRuns.rest.map(renderRow)}
            {hasNextPage && onLoadMore ? (
              <div className="px-3 py-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="w-full text-xs"
                  onClick={onLoadMore}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : null}
                  Load more
                </Button>
              </div>
            ) : null}
          </>
        )}
      </div>

      {hasAnyFilter && runs.length > 0 ? (
        <div className="flex shrink-0 items-center justify-between gap-2 border-t border-slate-200 bg-slate-50 px-3 py-1.5 text-[11px] text-gray-500">
          <span>
            Showing {filteredRuns.length} of {runs.length} loaded
          </span>
          <button type="button" onClick={clearFilters} className="shrink-0 text-sky-600 hover:text-sky-800">
            Clear filters
          </button>
        </div>
      ) : null}

      <div
        onMouseDown={handleMouseDown}
        className={cn(
          "absolute right-0 top-0 bottom-0 z-30 flex w-4 cursor-ew-resize items-center justify-center transition-colors hover:bg-gray-100 group",
          isResizing && "bg-blue-50",
        )}
        style={{ marginRight: "-8px" }}
        aria-label="Resize runs sidebar"
        role="separator"
      >
        <div
          className={cn(
            "h-14 w-2 rounded-full bg-gray-300 transition-colors group-hover:bg-gray-800",
            isResizing && "bg-blue-500",
          )}
        />
      </div>
    </div>
  );
}
