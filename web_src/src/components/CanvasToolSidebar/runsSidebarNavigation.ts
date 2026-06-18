type OrderedRuns = {
  active: Array<{ run: { id?: string } }>;
  rest: Array<{ run: { id?: string } }>;
};

export function buildSidebarRunIds(orderedRuns: OrderedRuns): string[] {
  return [...orderedRuns.active, ...orderedRuns.rest]
    .map((item) => item.run.id)
    .filter((id): id is string => Boolean(id));
}

export function getAdjacentSidebarRunId(
  runIds: string[],
  currentRunId: string,
  direction: "prev" | "next",
): string | null {
  const currentIndex = runIds.indexOf(currentRunId);
  if (currentIndex === -1) return null;

  const nextIndex = direction === "prev" ? currentIndex - 1 : currentIndex + 1;
  return runIds[nextIndex] ?? null;
}

export function isAtOlderRunPaginationBoundary(runIds: string[], currentRunId: string): boolean {
  if (runIds.length === 0) {
    return false;
  }

  if (getAdjacentSidebarRunId(runIds, currentRunId, "next")) {
    return false;
  }

  return runIds.indexOf(currentRunId) === runIds.length - 1;
}

export function canNavigateToOlderRun(
  runIds: string[],
  currentRunId: string,
  hasNextPage = false,
  allowPagedNavigation = true,
): boolean {
  if (getAdjacentSidebarRunId(runIds, currentRunId, "next")) {
    return true;
  }

  return allowPagedNavigation && hasNextPage && isAtOlderRunPaginationBoundary(runIds, currentRunId);
}

export function getRunSidebarNavigation(
  orderedRuns: OrderedRuns,
  currentRunId: string | null,
  options: { hasNextPage?: boolean; hasActiveFilters?: boolean } = {},
) {
  const hasNextPage = options.hasNextPage ?? false;
  const allowPagedNavigation = !options.hasActiveFilters;
  const runIds = buildSidebarRunIds(orderedRuns);

  if (!currentRunId) {
    return {
      runIds,
      newerRunId: null as string | null,
      olderRunId: null as string | null,
      canNavigateOlder: false,
      atOlderPaginationBoundary: false,
    };
  }

  const newerRunId = getAdjacentSidebarRunId(runIds, currentRunId, "prev");
  const olderRunId = getAdjacentSidebarRunId(runIds, currentRunId, "next");
  const atOlderPaginationBoundary = isAtOlderRunPaginationBoundary(runIds, currentRunId);

  return {
    runIds,
    newerRunId,
    olderRunId,
    canNavigateOlder: canNavigateToOlderRun(runIds, currentRunId, hasNextPage, allowPagedNavigation),
    atOlderPaginationBoundary,
  };
}
