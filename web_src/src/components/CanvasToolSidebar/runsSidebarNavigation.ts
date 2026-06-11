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
