import { useCallback, useEffect, useMemo, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { buildNodeMap, buildRunPresentation, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { AlertCircle, Loader2, Search, X } from "lucide-react";
import { loadPersistedFilters, savePersistedFilters } from "./filterPersistence";
import { RunRow } from "./RunRow";
import { RunFiltersPopover, type TriggerOption } from "./RunFiltersPopover";
import { useResizableSidebar } from "./useResizableSidebar";

export { RUNS_SIDEBAR_WIDTH_STORAGE_KEY } from "./useResizableSidebar";

interface RunsSidebarProps {
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  totalCount?: number;
  onStatusFiltersChange?: (filters: RunStatusFilter[]) => void;
}

export function RunsSidebar({
  runs,
  selectedRunId,
  onSelectRun,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  isLoading,
  isError,
  onRetry,
  workflowNodes = [],
  componentIconMap = {},
  totalCount,
  onStatusFiltersChange,
}: RunsSidebarProps) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useResizableSidebar();
  const [search, setSearch] = useState("");
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(() => loadPersistedFilters().triggerIds);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<RunStatusFilter>>(() => loadPersistedFilters().statuses);

  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);

  const triggerOptions = useMemo<TriggerOption[]>(() => {
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
    onStatusFiltersChange?.(Array.from(selectedStatuses));
    savePersistedFilters({ statuses: selectedStatuses, triggerIds: selectedTriggerIds });
  }, [selectedStatuses, selectedTriggerIds, onStatusFiltersChange]);

  useEffect(() => {
    if (triggerOptions.length === 0) return;
    const valid = new Set(triggerOptions.map((option) => option.id));
    setSelectedTriggerIds((prev) => {
      const next = new Set(Array.from(prev).filter((id) => valid.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [triggerOptions]);

  const decoratedRuns = useMemo(() => {
    return runs.map((run) => buildRunPresentation(run, nodeMap));
  }, [runs, nodeMap]);

  const filteredRuns = useMemo(() => {
    const query = search.trim().toLowerCase();
    return decoratedRuns.filter(({ run, status, haystack }) => {
      if (query && !haystack.includes(query)) return false;
      if (selectedStatuses.size > 0) {
        if (status === "unknown" || !selectedStatuses.has(status)) {
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

  const toggleStatus = useCallback((status: RunStatusFilter) => {
    setSelectedStatuses((prev) => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  }, []);

  const toggleTrigger = useCallback((triggerId: string) => {
    setSelectedTriggerIds((prev) => {
      const next = new Set(prev);
      if (next.has(triggerId)) next.delete(triggerId);
      else next.add(triggerId);
      return next;
    });
  }, []);

  return (
    <div
      ref={sidebarRef}
      data-testid="runs-sidebar"
      className="relative flex shrink-0 flex-col border-r border-slate-200 bg-white"
      style={{ width: `${width}px`, minWidth: `${width}px`, maxWidth: `${width}px` }}
    >
      <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
        <span className="text-sm font-medium text-gray-700">Runs</span>
        {totalCount != null && totalCount > 0 ? (
          <span className="ml-1.5 text-xs text-gray-400">({totalCount})</span>
        ) : null}
      </div>

      <div className="flex shrink-0 items-center gap-1.5 border-b border-slate-200 px-2 py-1.5">
        <RunFiltersPopover
          selectedStatuses={selectedStatuses}
          selectedTriggerIds={selectedTriggerIds}
          triggerOptions={triggerOptions}
          onToggleStatus={toggleStatus}
          onClearStatuses={() => setSelectedStatuses(new Set())}
          onToggleTrigger={toggleTrigger}
          onClearTriggers={() => setSelectedTriggerIds(new Set())}
        />

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
        {isError && runs.length === 0 ? (
          <div role="alert" className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-gray-500">
            <AlertCircle className="h-5 w-5 text-red-500" aria-hidden />
            <span>Failed to load runs</span>
            {onRetry ? (
              <button type="button" onClick={onRetry} className="text-[11px] text-sky-600 hover:text-sky-800">
                Try again
              </button>
            ) : null}
          </div>
        ) : isLoading ? (
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
            {orderedRuns.active.map((item) => (
              <RunRow
                key={item.run.id}
                run={item.run}
                triggerName={item.triggerName}
                title={item.title}
                status={item.status}
                triggerNode={item.triggerNode}
                isSelected={item.run.id === selectedRunId}
                componentIconMap={componentIconMap}
                onSelectRun={onSelectRun}
              />
            ))}
            {orderedRuns.active.length > 0 && orderedRuns.rest.length > 0 ? (
              <div className="h-px bg-slate-300" aria-hidden />
            ) : null}
            {orderedRuns.rest.map((item) => (
              <RunRow
                key={item.run.id}
                run={item.run}
                triggerName={item.triggerName}
                title={item.title}
                status={item.status}
                triggerNode={item.triggerNode}
                isSelected={item.run.id === selectedRunId}
                componentIconMap={componentIconMap}
                onSelectRun={onSelectRun}
              />
            ))}
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
            {isError ? (
              <div role="alert" className="flex items-center justify-between gap-2 px-3 py-2 text-[11px] text-red-600">
                <span className="inline-flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" aria-hidden />
                  Failed to load more runs
                </span>
                {onRetry ? (
                  <button type="button" onClick={onRetry} className="text-sky-600 hover:text-sky-800">
                    Retry
                  </button>
                ) : null}
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
