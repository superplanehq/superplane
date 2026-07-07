import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useCallback, useEffect, useRef, type UIEvent } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { LiveCanvasSidebarRow } from "./LiveCanvasSidebarRow";
import { RunsTabListView } from "./RunsTabListView";
import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";
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
  runs,
  selectedRunId,
  onSelectRun,
  onSelectLiveCanvas,
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
  const filterState = useRunFilters({ runs, workflowNodes, componentIconMap, onStatusFiltersChange });
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
  }, [filterState.filteredRuns.length, loadMoreIfNeeded]);

  const handleRunSelect = useCallback(
    (runId: string) => {
      onSelectRun(runId);
    },
    [onSelectRun],
  );

  const handleSelectLiveCanvas = useCallback(() => {
    onSelectLiveCanvas();
  }, [onSelectLiveCanvas]);

  const isLiveCanvasSelected = !selectedRunId;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <LiveCanvasSidebarRow isSelected={isLiveCanvasSelected} onSelect={handleSelectLiveCanvas} />

      <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <RunsTabListView
          isActive
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
          searchQuery={filterState.searchQuery}
          triggerOptions={filterState.triggerOptions}
          onToggleStatus={filterState.toggleStatus}
          onClearStatuses={filterState.clearStatuses}
          onToggleTrigger={filterState.toggleTrigger}
          onClearTriggers={filterState.clearTriggers}
          onSearchQueryChange={filterState.setSearchQuery}
        />
      </div>
    </div>
  );
}
