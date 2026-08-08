import { useMemo } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { getRunSidebarNavigation } from "@/components/CanvasToolSidebar/runsSidebarNavigation";
import { useRunFilters } from "@/components/CanvasToolSidebar/useRunFilters";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";

interface UseRunSidebarNavigationStateParams {
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  hasNextPage?: boolean;
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  onStatusFiltersChange: (filters: RunStatusFilter[]) => void;
  onMyRunsOnlyChange: (enabled: boolean) => void;
}

export function useRunSidebarNavigationState({
  runs,
  selectedRunId,
  hasNextPage,
  workflowNodes,
  componentIconMap,
  onStatusFiltersChange,
  onMyRunsOnlyChange,
}: UseRunSidebarNavigationStateParams) {
  const runFilterState = useRunFilters({ runs, workflowNodes, componentIconMap, onStatusFiltersChange, onMyRunsOnlyChange });
  const runNavigation = useMemo(
    () =>
      getRunSidebarNavigation(runFilterState.orderedRuns, selectedRunId, {
        hasNextPage: !!hasNextPage,
        hasActiveFilters: runFilterState.hasAnyFilter,
      }),
    [hasNextPage, runFilterState.hasAnyFilter, runFilterState.orderedRuns, selectedRunId],
  );

  return { runFilterState, runNavigation };
}
