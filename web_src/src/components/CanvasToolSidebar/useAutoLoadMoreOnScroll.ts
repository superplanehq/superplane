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

  useEffect(() => {
    if (!isLoading) {
      requestedRef.current = false;
    }
  }, [isLoading]);

  return useCallback(
    (element: HTMLElement | null) => {
      if (!element || !hasMore || isLoading || !onLoadMore || requestedRef.current) return;

      const remainingScroll = element.scrollHeight - element.scrollTop - element.clientHeight;
      if (remainingScroll > AUTO_LOAD_SCROLL_THRESHOLD_PX) return;

      requestedRef.current = true;
      onLoadMore();
    },
    [hasMore, isLoading, onLoadMore],
  );
}
