import { useCallback, useEffect, useRef } from "react";

const AUTO_LOAD_SCROLL_THRESHOLD_PX = 160;

export function useAutoLoadMoreOnScroll({
  hasMore,
  isLoading,
  onLoadMore,
}: {
  hasMore?: boolean;
  isLoading?: boolean;
  onLoadMore?: () => void;
}) {
  const requestedRef = useRef(false);
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (isLoading) {
      return;
    }
    requestedRef.current = false;
  }, [hasMore, isLoading]);

  useEffect(() => {
    return () => {
      if (resetTimerRef.current == null) {
        return;
      }
      clearTimeout(resetTimerRef.current);
    };
  }, []);

  return useCallback(
    (element: HTMLElement | null) => {
      if (!element || !hasMore || isLoading || !onLoadMore || requestedRef.current) return;

      const remainingScroll = element.scrollHeight - element.scrollTop - element.clientHeight;
      if (remainingScroll > AUTO_LOAD_SCROLL_THRESHOLD_PX) return;

      requestedRef.current = true;
      onLoadMore();
      if (isLoading !== undefined) {
        return;
      }
      if (resetTimerRef.current != null) {
        clearTimeout(resetTimerRef.current);
      }
      resetTimerRef.current = setTimeout(() => {
        requestedRef.current = false;
        resetTimerRef.current = null;
      }, 0);
    },
    [hasMore, isLoading, onLoadMore],
  );
}
