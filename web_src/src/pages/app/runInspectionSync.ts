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
  isRunResolveLoading,
  isRunNotFound,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunResolveLoading: boolean;
  isRunNotFound: boolean;
}): boolean {
  if (!selectedRunId || !isRunInspectionMode) return false;
  if (isRunResolveLoading) return false;
  if (selectedRun) return false;
  return isRunNotFound;
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
