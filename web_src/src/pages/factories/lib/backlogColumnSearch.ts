import type { FactoriesWorkOrder } from "@/api-client";

import { applyWorkOrderSearch, buildWorkOrderListEntries } from "./workOrderListModel";

export const BACKLOG_COLUMN_SEARCH_COPY = {
  placeholder: "Search tasks",
  noMatch: "No tasks match this search.",
  loadingMore: "Loading more",
  loadError: "SuperPlane could not load more tasks.",
  retry: "Try again",
} as const;

export function shouldLoadNextBacklogSearchPage(args: {
  query: string;
  hasMore: boolean;
  isLoadingNextPage: boolean;
  hasPageError: boolean;
}): boolean {
  if (!args.query.trim() || !args.hasMore || args.isLoadingNextPage || args.hasPageError) {
    return false;
  }
  return true;
}

export function backlogSearchColumnState(args: {
  query: string;
  matchCount: number;
  loadedCount: number;
  hasMore: boolean;
  isLoadingNextPage: boolean;
  hasPageError: boolean;
}): {
  searchActive: boolean;
  columnCount: number;
  showLoadingMore: boolean;
  showLoadError: boolean;
  showNoMatch: boolean;
} {
  if (!args.query.trim()) {
    return {
      searchActive: false,
      columnCount: args.loadedCount,
      showLoadingMore: false,
      showLoadError: false,
      showNoMatch: false,
    };
  }

  const showLoadingMore =
    args.isLoadingNextPage ||
    shouldLoadNextBacklogSearchPage({
      query: args.query,
      hasMore: args.hasMore,
      isLoadingNextPage: args.isLoadingNextPage,
      hasPageError: args.hasPageError,
    });
  const showLoadError = args.hasPageError && !args.isLoadingNextPage;
  return {
    searchActive: true,
    columnCount: args.matchCount,
    showLoadingMore,
    showLoadError,
    showNoMatch: !showLoadingMore && !showLoadError && !args.hasMore && args.matchCount === 0,
  };
}

export function backlogOrdersMatchingSearch(orders: FactoriesWorkOrder[], query: string): FactoriesWorkOrder[] {
  if (!query.trim()) {
    return orders;
  }
  const matchedIds = new Set(
    applyWorkOrderSearch(buildWorkOrderListEntries(orders, undefined), query).map((entry) => entry.id),
  );
  return orders.filter((order) => Boolean(order.id) && matchedIds.has(order.id ?? ""));
}
