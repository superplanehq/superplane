import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { RunDetailPanel } from "./RunDetailPanel";
import { RunsTabListView } from "./RunsTabListView";
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
  detailDismissedForRunId?: string | null;
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
  detailDismissedForRunId = null,
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

  const filterState = useRunFilters({ runs, workflowNodes, componentIconMap, onStatusFiltersChange });
  const scrollRef = useRef<HTMLDivElement>(null);
  const previousSelectedRunIdRef = useRef<string | null>(selectedRunId);
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
    if (!selectedRunId) {
      setSidebarView("list");
      previousSelectedRunIdRef.current = null;
      return;
    }

    const previousRunId = previousSelectedRunIdRef.current;
    const runIdChanged = previousRunId !== selectedRunId;

    if (initialOpenDetail && selectedRunId !== detailDismissedForRunId) {
      setSidebarView("detail");
    } else if (previousRunId !== null && runIdChanged && selectedRunId !== detailDismissedForRunId) {
      setSidebarView("detail");
    }

    previousSelectedRunIdRef.current = selectedRunId;
  }, [detailDismissedForRunId, initialOpenDetail, selectedRunId]);

  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [filterState.filteredRuns.length, loadMoreIfNeeded]);

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
      <RunsTabListView
        isActive={!isDetailView}
        scrollRef={scrollRef}
        onScroll={handleScroll}
        runs={runs}
        filteredRuns={filterState.filteredRuns}
        orderedRuns={filterState.orderedRuns}
        selectedRunId={selectedRunId}
        onSelectRun={handleRunSelect}
        componentIconMap={componentIconMap}
        isLoading={isLoading}
        isError={isError}
        onRetry={onRetry}
        onClearFilters={filterState.clearFilters}
        hasAnyFilter={filterState.hasAnyFilter}
        selectedStatuses={filterState.selectedStatuses}
        selectedTriggerIds={filterState.selectedTriggerIds}
        triggerOptions={filterState.triggerOptions}
        onToggleStatus={filterState.toggleStatus}
        onClearStatuses={filterState.clearStatuses}
        onToggleTrigger={filterState.toggleTrigger}
        onClearTriggers={filterState.clearTriggers}
      />

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
