import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { LiveCanvasSidebarRow } from "./LiveCanvasSidebarRow";
import { RunDetailPanel } from "./RunDetailPanel";
import { RunsTabListView } from "./RunsTabListView";
import { getAdjacentSidebarRunId, getRunSidebarNavigation } from "./runsSidebarNavigation";
import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";
import { useRunFilters } from "./useRunFilters";

export type RunsSidebarView = "list" | "detail";

export interface RunsTabPanelProps {
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
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

  const olderNavigationLoadRequestedRef = useRef(false);
  const filteredRunIdsLengthAtFetchStartRef = useRef(0);
  const pendingOlderNavigationRunIdRef = useRef<string | null>(null);
  const lastOlderFetchCompletedRef = useRef(false);
  const wasFetchingNextPageRef = useRef(false);
  const [olderNavigationSession, setOlderNavigationSession] = useState(0);
  const [olderFetchCompletedTick, setOlderFetchCompletedTick] = useState(0);
  const [pendingOlderNavigation, setPendingOlderNavigation] = useState(false);

  const handleNavigateRun = useCallback(
    (runId: string) => {
      (onNavigateRun ?? onSelectRun)(runId);
      setSidebarView("detail");
    },
    [onNavigateRun, onSelectRun],
  );

  const handleNavigateOlder = useCallback(() => {
    if (olderRunId) {
      handleNavigateRun(olderRunId);
      return;
    }

    if (!atOlderPaginationBoundary || !hasNextPage || filterState.hasAnyFilter) {
      return;
    }

    pendingOlderNavigationRunIdRef.current = selectedRunId;
    filteredRunIdsLengthAtFetchStartRef.current = sidebarRunIds.length;
    olderNavigationLoadRequestedRef.current = false;
    lastOlderFetchCompletedRef.current = false;
    setOlderNavigationSession((session) => session + 1);
    setPendingOlderNavigation(true);
  }, [
    atOlderPaginationBoundary,
    filterState.hasAnyFilter,
    handleNavigateRun,
    hasNextPage,
    olderRunId,
    selectedRunId,
    sidebarRunIds.length,
  ]);

  useEffect(() => {
    if (!pendingOlderNavigation || isFetchingNextPage || !onLoadMore) {
      return;
    }

    if (!selectedRunId || selectedRunId !== pendingOlderNavigationRunIdRef.current) {
      pendingOlderNavigationRunIdRef.current = null;
      olderNavigationLoadRequestedRef.current = false;
      lastOlderFetchCompletedRef.current = false;
      setPendingOlderNavigation(false);
      return;
    }

    const nextOlderRunId = getAdjacentSidebarRunId(sidebarRunIds, selectedRunId, "next");
    if (nextOlderRunId) {
      pendingOlderNavigationRunIdRef.current = null;
      olderNavigationLoadRequestedRef.current = false;
      lastOlderFetchCompletedRef.current = false;
      setPendingOlderNavigation(false);
      handleNavigateRun(nextOlderRunId);
      return;
    }

    if (!hasNextPage) {
      pendingOlderNavigationRunIdRef.current = null;
      olderNavigationLoadRequestedRef.current = false;
      lastOlderFetchCompletedRef.current = false;
      setPendingOlderNavigation(false);
      return;
    }

    if (lastOlderFetchCompletedRef.current && sidebarRunIds.length === filteredRunIdsLengthAtFetchStartRef.current) {
      pendingOlderNavigationRunIdRef.current = null;
      olderNavigationLoadRequestedRef.current = false;
      lastOlderFetchCompletedRef.current = false;
      setPendingOlderNavigation(false);
      return;
    }

    if (olderNavigationLoadRequestedRef.current) {
      return;
    }

    olderNavigationLoadRequestedRef.current = true;
    lastOlderFetchCompletedRef.current = false;
    onLoadMore();
  }, [
    handleNavigateRun,
    hasNextPage,
    isFetchingNextPage,
    olderFetchCompletedTick,
    olderNavigationSession,
    onLoadMore,
    pendingOlderNavigation,
    selectedRunId,
    sidebarRunIds,
  ]);

  useEffect(() => {
    const wasFetching = wasFetchingNextPageRef.current;
    wasFetchingNextPageRef.current = !!isFetchingNextPage;

    if (!wasFetching || isFetchingNextPage || !pendingOlderNavigation) {
      return;
    }

    lastOlderFetchCompletedRef.current = true;
    olderNavigationLoadRequestedRef.current = false;
    setOlderFetchCompletedTick((tick) => tick + 1);
  }, [isFetchingNextPage, pendingOlderNavigation]);

  const isDetailView = sidebarView === "detail" && !!selectedRun;
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
          className={`absolute inset-0 flex min-h-0 min-w-0 flex-col overflow-hidden bg-white transition-transform duration-300 ease-in-out ${
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
          ) : null}
        </div>
      </div>
    </div>
  );
}
