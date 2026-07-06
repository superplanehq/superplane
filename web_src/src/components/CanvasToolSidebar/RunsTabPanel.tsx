import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { LiveCanvasSidebarRow } from "./LiveCanvasSidebarRow";
import { RunDetailPanel } from "./RunDetailPanel";
import { RunsTabListView } from "./RunsTabListView";
import { getRunSidebarNavigation } from "./runsSidebarNavigation";
import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";
import { useOlderRunSidebarNavigation } from "./useOlderRunSidebarNavigation";
import { useRunFilters } from "./useRunFilters";

export type RunsSidebarView = "list" | "detail";

export interface RunsTabPanelProps {
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  selectedRun?: CanvasesCanvasRun | null;
  isSelectedRunLoading?: boolean;
  onSelectRun: (runId: string) => void;
  onNavigateRun?: (runId: string) => void;
  onSelectLiveCanvas: () => void;
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
  selectedRun: selectedRunProp = null,
  isSelectedRunLoading = false,
  onSelectRun,
  onNavigateRun,
  onSelectLiveCanvas,
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

  const selectedRun = useMemo(
    () => selectedRunProp ?? runs.find((run) => run.id === selectedRunId) ?? null,
    [selectedRunProp, runs, selectedRunId],
  );

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
    if (sidebarView === "detail" && selectedRunId) {
      return;
    }

    loadMoreIfNeeded(scrollRef.current);
  }, [filterState.filteredRuns.length, loadMoreIfNeeded, selectedRunId, sidebarView]);

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

  const handleSelectLiveCanvas = useCallback(() => {
    setSidebarView("list");
    onSelectLiveCanvas();
  }, [onSelectLiveCanvas]);

  const {
    runIds: sidebarRunIds,
    newerRunId,
    olderRunId,
    canNavigateOlder,
    atOlderPaginationBoundary,
  } = useMemo(
    () =>
      getRunSidebarNavigation(filterState.orderedRuns, selectedRunId, {
        hasNextPage: !!hasNextPage,
        hasActiveFilters: filterState.hasAnyFilter,
      }),
    [filterState.hasAnyFilter, filterState.orderedRuns, hasNextPage, selectedRunId],
  );

  const handleNavigateRun = useCallback(
    (runId: string) => {
      (onNavigateRun ?? onSelectRun)(runId);
      setSidebarView("detail");
    },
    [onNavigateRun, onSelectRun],
  );

  const { handleNavigateOlder } = useOlderRunSidebarNavigation({
    selectedRunId,
    sidebarRunIds,
    olderRunId,
    atOlderPaginationBoundary,
    hasNextPage,
    hasActiveFilters: filterState.hasAnyFilter,
    isFetchingNextPage,
    onLoadMore,
    onNavigateRun: handleNavigateRun,
  });

  const isDetailView = sidebarView === "detail" && (!!selectedRun || (!!selectedRunId && isSelectedRunLoading));
  const isLiveCanvasSelected = !selectedRunId;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <LiveCanvasSidebarRow isSelected={isLiveCanvasSelected} onSelect={handleSelectLiveCanvas} />

      <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
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
          className={`absolute inset-0 flex min-h-0 min-w-0 flex-col overflow-hidden bg-white transition-transform duration-300 ease-in-out dark:bg-gray-900 ${
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
              newerRunId={newerRunId}
              olderRunId={olderRunId}
              canNavigateOlder={canNavigateOlder}
              onNavigateRun={handleNavigateRun}
              onNavigateOlder={
                atOlderPaginationBoundary && hasNextPage && !filterState.hasAnyFilter ? handleNavigateOlder : undefined
              }
            />
          ) : isSelectedRunLoading ? (
            <div className="flex min-h-0 flex-1 items-center justify-center px-4 text-sm text-gray-500">
              Loading run…
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
