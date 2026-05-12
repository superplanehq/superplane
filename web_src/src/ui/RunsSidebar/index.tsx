import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { cn } from "@/lib/utils";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { RunsList } from "./RunsList";
import { RunsToolbar } from "./RunsToolbar";
import { useResizableSidebar } from "./useResizableSidebar";
import { useRunFilters } from "./useRunFilters";

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
  const {
    search,
    setSearch,
    selectedStatuses,
    selectedTriggerIds,
    triggerOptions,
    filteredRuns,
    orderedRuns,
    hasAnyFilter,
    clearFilters,
    toggleStatus,
    toggleTrigger,
    clearStatuses,
    clearTriggers,
  } = useRunFilters({ runs, workflowNodes, componentIconMap, onStatusFiltersChange });

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

      <RunsToolbar
        search={search}
        onSearchChange={setSearch}
        selectedStatuses={selectedStatuses}
        selectedTriggerIds={selectedTriggerIds}
        triggerOptions={triggerOptions}
        onToggleStatus={toggleStatus}
        onClearStatuses={clearStatuses}
        onToggleTrigger={toggleTrigger}
        onClearTriggers={clearTriggers}
      />

      <div className="flex-1 overflow-y-auto">
        <RunsList
          runs={runs}
          filteredRuns={filteredRuns}
          orderedRuns={orderedRuns}
          selectedRunId={selectedRunId}
          onSelectRun={onSelectRun}
          componentIconMap={componentIconMap}
          isLoading={isLoading}
          isError={isError}
          onRetry={onRetry}
          hasNextPage={hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          onLoadMore={onLoadMore}
          onClearFilters={clearFilters}
        />
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
