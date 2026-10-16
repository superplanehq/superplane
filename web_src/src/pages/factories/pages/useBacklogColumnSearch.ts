import type { FactoriesWorkOrder } from "@/api-client";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  backlogOrdersMatchingSearch,
  backlogSearchColumnState,
  shouldLoadNextBacklogSearchPage,
} from "../lib/backlogColumnSearch";

export type BacklogColumnPaging = {
  hasMore: boolean;
  isLoading: boolean;
  hasPageError: boolean;
  onLoadMore: () => void;
};

export function useBacklogColumnSearch(orders: FactoriesWorkOrder[], paging?: BacklogColumnPaging) {
  const [query, setQuery] = useState("");
  const visibleOrders = useMemo(() => backlogOrdersMatchingSearch(orders, query), [orders, query]);
  const page = backlogPageFlags(paging);
  const loadMoreRef = useRef(paging?.onLoadMore);
  loadMoreRef.current = paging?.onLoadMore;
  const view = backlogSearchColumnState({
    query,
    matchCount: visibleOrders.length,
    loadedCount: orders.length,
    ...page,
  });

  const { hasMore, isLoadingNextPage, hasPageError } = page;

  useEffect(() => {
    if (!shouldLoadNextBacklogSearchPage({ query, hasMore, isLoadingNextPage, hasPageError })) {
      return;
    }
    loadMoreRef.current?.();
  }, [hasMore, hasPageError, isLoadingNextPage, query]);

  return {
    query,
    setQuery,
    visibleOrders,
    ...view,
    retryLoadMore: () => {
      loadMoreRef.current?.();
    },
  };
}

function backlogPageFlags(paging?: BacklogColumnPaging) {
  return {
    hasMore: paging?.hasMore ?? false,
    isLoadingNextPage: paging?.isLoading ?? false,
    hasPageError: paging?.hasPageError ?? false,
  };
}
