type RunsPage = { runs?: unknown[]; totalCount?: number } | undefined;

export function hasLoadedAllRuns(pages: RunsPage[], hasNextPage: boolean): boolean {
  const loadedCount = pages.reduce((acc, page) => acc + (page?.runs?.length ?? 0), 0);
  const totalCount = pages[0]?.totalCount ?? 0;
  return loadedCount >= totalCount || !hasNextPage;
}

export function shouldClearStaleRunUrl({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  isRunsQueryLoading,
  isFetchingNextPage,
  pages,
  hasNextPage,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunsQueryLoading: boolean;
  isFetchingNextPage: boolean;
  pages: RunsPage[];
  hasNextPage: boolean;
}): boolean {
  if (!selectedRunId || !isRunInspectionMode) return false;
  if (isRunsQueryLoading || isFetchingNextPage) return false;
  if (selectedRun) return false;
  return hasLoadedAllRuns(pages, hasNextPage);
}

export function shouldClearRunDetailNode({
  runDetailNodeId,
  participantNodeIds,
  runCanvasLoading,
}: {
  runDetailNodeId: string | null;
  participantNodeIds: string[];
  runCanvasLoading: boolean;
}): boolean {
  if (!runDetailNodeId || runCanvasLoading) return false;
  if (participantNodeIds.length === 0) return false;
  return !participantNodeIds.includes(runDetailNodeId);
}
