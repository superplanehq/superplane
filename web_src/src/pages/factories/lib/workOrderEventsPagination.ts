import type { FactoriesListWorkOrderEventsResponse, FactoriesWorkOrderEvent } from "@/api-client";

export const WORK_ORDER_EVENTS_PAGE_LIMIT = 50;

export function flattenWorkOrderEventsPages(
  pages: Array<FactoriesListWorkOrderEventsResponse | undefined> | undefined,
): FactoriesWorkOrderEvent[] {
  return pages?.flatMap((page) => page?.events ?? []) ?? [];
}

export function getWorkOrderEventsNextPageParam(
  lastPage: FactoriesListWorkOrderEventsResponse | undefined,
  allPages: Array<FactoriesListWorkOrderEventsResponse | undefined>,
): string | undefined {
  const loadedCount = allPages.reduce((count, page) => count + (page?.events?.length ?? 0), 0);
  const totalCount = lastPage?.totalCount ?? 0;

  if (loadedCount >= totalCount) {
    return undefined;
  }

  if (!lastPage?.hasNextPage || !lastPage.lastTimestamp) {
    return undefined;
  }

  return lastPage.lastTimestamp;
}
