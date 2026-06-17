type RunsPage = { runs?: unknown[]; totalCount?: number } | undefined;

export function hasLoadedAllRuns(pages: RunsPage[], hasNextPage: boolean): boolean {
  const loadedCount = pages.reduce((acc, page) => acc + (page?.runs?.length ?? 0), 0);
  const totalCount = pages[0]?.totalCount;
  if (totalCount !== undefined) {
    return loadedCount >= totalCount || !hasNextPage;
  }
  return !hasNextPage;
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
  runCanvasSettled,
}: {
  runDetailNodeId: string | null;
  participantNodeIds: string[];
  runCanvasLoading: boolean;
  runCanvasSettled: boolean;
}): boolean {
  if (!runDetailNodeId || runCanvasLoading || !runCanvasSettled) return false;
  if (participantNodeIds.length === 0) return true;
  return !participantNodeIds.includes(runDetailNodeId);
}

export function clearRunDetailNodeSearchParams(searchParams: URLSearchParams, nodeId: string): URLSearchParams {
  const next = new URLSearchParams(searchParams);
  if (next.get("sidebar") === "1" && next.get("node") === nodeId) {
    next.delete("sidebar");
    next.delete("node");
  }
  return next;
}
