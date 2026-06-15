import { useEffect } from "react";
import { hasLoadedAllRuns, shouldClearStaleRunUrl } from "./runInspectionSync";

type InfiniteRunsQuery = {
  data?: { pages?: Array<{ runs?: unknown[]; totalCount?: number }> };
  isLoading: boolean;
  isFetchingNextPage: boolean;
  hasNextPage?: boolean;
  fetchNextPage: () => Promise<unknown>;
};

export function useStaleRunInspectionUrlCleanup({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  infiniteRunsQuery,
  onClear,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  infiniteRunsQuery: InfiniteRunsQuery;
  onClear: () => void;
}) {
  useEffect(() => {
    const pages = infiniteRunsQuery.data?.pages ?? [];
    const hasNextPage = !!infiniteRunsQuery.hasNextPage;

    if (
      shouldClearStaleRunUrl({
        selectedRunId,
        isRunInspectionMode,
        selectedRun,
        isRunsQueryLoading: infiniteRunsQuery.isLoading,
        isFetchingNextPage: infiniteRunsQuery.isFetchingNextPage,
        pages,
        hasNextPage,
      })
    ) {
      onClear();
      return;
    }

    if (!selectedRunId || !isRunInspectionMode || selectedRun) return;
    if (infiniteRunsQuery.isLoading || infiniteRunsQuery.isFetchingNextPage) return;
    if (infiniteRunsQuery.data === undefined) return;

    if (!hasLoadedAllRuns(pages, hasNextPage)) {
      void infiniteRunsQuery.fetchNextPage();
    }
  }, [infiniteRunsQuery, isRunInspectionMode, onClear, selectedRun, selectedRunId]);
}
