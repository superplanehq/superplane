import { useCallback, useEffect, useRef, useState, type MutableRefObject } from "react";
import { getAdjacentSidebarRunId } from "./runsSidebarNavigation";

function resetPendingOlderNavigation(
  pendingRunIdRef: MutableRefObject<string | null>,
  loadRequestedRef: MutableRefObject<boolean>,
  lastFetchCompletedRef: MutableRefObject<boolean>,
  setPendingOlderNavigation: (value: boolean) => void,
) {
  pendingRunIdRef.current = null;
  loadRequestedRef.current = false;
  lastFetchCompletedRef.current = false;
  setPendingOlderNavigation(false);
}

export function useOlderRunSidebarNavigation({
  selectedRunId,
  sidebarRunIds,
  olderRunId,
  atOlderPaginationBoundary,
  hasNextPage,
  hasActiveFilters,
  isFetchingNextPage,
  onLoadMore,
  onNavigateRun,
}: {
  selectedRunId: string | null;
  sidebarRunIds: string[];
  olderRunId: string | null;
  atOlderPaginationBoundary: boolean;
  hasNextPage?: boolean;
  hasActiveFilters: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  onNavigateRun: (runId: string) => void;
}) {
  const loadRequestedRef = useRef(false);
  const filteredRunIdsLengthAtFetchStartRef = useRef(0);
  const pendingOlderNavigationRunIdRef = useRef<string | null>(null);
  const lastOlderFetchCompletedRef = useRef(false);
  const wasFetchingNextPageRef = useRef(false);
  const [olderNavigationSession, setOlderNavigationSession] = useState(0);
  const [olderFetchCompletedTick, setOlderFetchCompletedTick] = useState(0);
  const [pendingOlderNavigation, setPendingOlderNavigation] = useState(false);

  const handleNavigateOlder = useCallback(() => {
    if (olderRunId) {
      onNavigateRun(olderRunId);
      return;
    }

    if (!atOlderPaginationBoundary || !hasNextPage || hasActiveFilters) {
      return;
    }

    pendingOlderNavigationRunIdRef.current = selectedRunId;
    filteredRunIdsLengthAtFetchStartRef.current = sidebarRunIds.length;
    loadRequestedRef.current = false;
    lastOlderFetchCompletedRef.current = false;
    setOlderNavigationSession((session) => session + 1);
    setPendingOlderNavigation(true);
  }, [
    atOlderPaginationBoundary,
    hasActiveFilters,
    hasNextPage,
    olderRunId,
    onNavigateRun,
    selectedRunId,
    sidebarRunIds.length,
  ]);

  useEffect(() => {
    if (!pendingOlderNavigation || isFetchingNextPage || !onLoadMore) {
      return;
    }

    if (!selectedRunId || selectedRunId !== pendingOlderNavigationRunIdRef.current) {
      resetPendingOlderNavigation(
        pendingOlderNavigationRunIdRef,
        loadRequestedRef,
        lastOlderFetchCompletedRef,
        setPendingOlderNavigation,
      );
      return;
    }

    const nextOlderRunId = getAdjacentSidebarRunId(sidebarRunIds, selectedRunId, "next");
    if (nextOlderRunId) {
      resetPendingOlderNavigation(
        pendingOlderNavigationRunIdRef,
        loadRequestedRef,
        lastOlderFetchCompletedRef,
        setPendingOlderNavigation,
      );
      onNavigateRun(nextOlderRunId);
      return;
    }

    if (!hasNextPage) {
      resetPendingOlderNavigation(
        pendingOlderNavigationRunIdRef,
        loadRequestedRef,
        lastOlderFetchCompletedRef,
        setPendingOlderNavigation,
      );
      return;
    }

    if (lastOlderFetchCompletedRef.current && sidebarRunIds.length === filteredRunIdsLengthAtFetchStartRef.current) {
      resetPendingOlderNavigation(
        pendingOlderNavigationRunIdRef,
        loadRequestedRef,
        lastOlderFetchCompletedRef,
        setPendingOlderNavigation,
      );
      return;
    }

    if (loadRequestedRef.current) {
      return;
    }

    loadRequestedRef.current = true;
    lastOlderFetchCompletedRef.current = false;
    onLoadMore();
  }, [
    hasNextPage,
    isFetchingNextPage,
    olderFetchCompletedTick,
    olderNavigationSession,
    onLoadMore,
    onNavigateRun,
    pendingOlderNavigation,
    selectedRunId,
    sidebarRunIds,
  ]);

  useEffect(() => {
    const wasFetching = wasFetchingNextPageRef.current;
    wasFetchingNextPageRef.current = !!isFetchingNextPage;

    if (!wasFetching || isFetchingNextPage || !pendingOlderNavigation) {
      return;
    }

    lastOlderFetchCompletedRef.current = true;
    loadRequestedRef.current = false;
    setOlderFetchCompletedTick((tick) => tick + 1);
  }, [isFetchingNextPage, pendingOlderNavigation]);

  return { handleNavigateOlder };
}
