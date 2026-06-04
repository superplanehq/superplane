import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { RunDetailPanel } from "./RunDetailPanel";
import { RunsList } from "./RunsList";
import { RunsToolbar } from "./RunsToolbar";
import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";
import { useRunFilters } from "./useRunFilters";

export type RunsSidebarView = "list" | "detail";

export interface RunsTabPanelProps {
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  onBackToRunList?: () => void;
  initialOpenDetail?: boolean;
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onStatusFiltersChange?: (filters: RunStatusFilter[]) => void;
}

export function RunsTabPanel({
  canvasId,
  runs,
  selectedRunId,
  onSelectRun,
  onBackToRunList,
  initialOpenDetail = false,
  selectedNodeId = null,
  onSelectNode,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  isLoading,
  isError,
  onRetry,
  workflowNodes = [],
  componentIconMap = {},
  onStatusFiltersChange,
}: RunsTabPanelProps) {
  const [sidebarView, setSidebarView] = useState<RunsSidebarView>(() =>
    initialOpenDetail && selectedRunId ? "detail" : "list",
  );

  const selectedRun = useMemo(() => runs.find((run) => run.id === selectedRunId) || null, [runs, selectedRunId]);

  const {
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
  const scrollRef = useRef<HTMLDivElement>(null);
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: hasNextPage,
    isLoading: isFetchingNextPage,
    onLoadMore,
  });
  const handleScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      loadMoreIfNeeded(event.currentTarget);
    },
    [loadMoreIfNeeded],
  );

  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [filteredRuns.length, loadMoreIfNeeded]);

  useEffect(() => {
    if (!selectedRunId) {
      setSidebarView("list");
    }
  }, [selectedRunId]);

  const handleRunSelect = useCallback(
    (runId: string) => {
      onSelectRun(runId);
      setSidebarView("detail");
    },
    [onSelectRun],
  );

  const handleBack = useCallback(() => {
    setSidebarView("list");
    onBackToRunList?.();
  }, [onBackToRunList]);

  const isDetailView = sidebarView === "detail" && !!selectedRun;

  return (
    <div className="relative min-h-0 flex-1 overflow-hidden">
      <div
        className={`absolute inset-0 flex min-h-0 flex-col bg-white transition-transform duration-300 ease-in-out ${
          isDetailView ? "-translate-x-full" : "translate-x-0"
        } ${isDetailView ? "pointer-events-none" : "pointer-events-auto"}`}
      >
        <RunsToolbar
          selectedStatuses={selectedStatuses}
          selectedTriggerIds={selectedTriggerIds}
          triggerOptions={triggerOptions}
          onToggleStatus={toggleStatus}
          onClearStatuses={clearStatuses}
          onToggleTrigger={toggleTrigger}
          onClearTriggers={clearTriggers}
        />

        <div
          ref={scrollRef}
          className="min-h-0 flex-1 overflow-y-auto"
          data-testid="runs-sidebar-scroll"
          onScroll={handleScroll}
        >
          <RunsList
            runs={runs}
            filteredRuns={filteredRuns}
            orderedRuns={orderedRuns}
            selectedRunId={selectedRunId}
            onSelectRun={handleRunSelect}
            componentIconMap={componentIconMap}
            isLoading={isLoading}
            isError={isError}
            onRetry={onRetry}
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
      </div>

      <div
        className={`absolute inset-0 flex min-h-0 flex-col bg-white transition-transform duration-300 ease-in-out ${
          isDetailView ? "translate-x-0" : "translate-x-full"
        } ${isDetailView ? "pointer-events-auto" : "pointer-events-none"}`}
      >
        {selectedRun ? (
          <RunDetailPanel
            canvasId={canvasId}
            run={selectedRun}
            workflowNodes={workflowNodes}
            componentIconMap={componentIconMap}
            selectedNodeId={selectedNodeId}
            onSelectNode={onSelectNode ?? (() => {})}
            onBack={handleBack}
          />
        ) : null}
      </div>
    </div>
  );
}
