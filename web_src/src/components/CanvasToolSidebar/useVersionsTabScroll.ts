import { useCallback, useEffect, useLayoutEffect, useRef, type UIEvent } from "react";
import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";

const persistedScrollPositions = new Map<string, number>();

type UseVersionsTabScrollOptions = {
  scrollPersistenceKey?: string;
  hasMore: boolean;
  isLoading?: boolean;
  onLoadMore?: () => void;
  itemCount: number;
};

export function useVersionsTabScroll({
  scrollPersistenceKey,
  hasMore,
  isLoading,
  onLoadMore,
  itemCount,
}: UseVersionsTabScrollOptions) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore,
    isLoading,
    onLoadMore,
  });
  const handleScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      if (scrollPersistenceKey) {
        persistedScrollPositions.set(scrollPersistenceKey, event.currentTarget.scrollTop);
      }

      loadMoreIfNeeded(event.currentTarget);
    },
    [loadMoreIfNeeded, scrollPersistenceKey],
  );

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || !scrollPersistenceKey) return;

    const scrollTop = persistedScrollPositions.get(scrollPersistenceKey);
    if (scrollTop == null) return;

    element.scrollTop = scrollTop;
  }, [scrollPersistenceKey]);

  useEffect(() => {
    const element = scrollRef.current;

    return () => {
      if (!element || !scrollPersistenceKey) return;

      persistedScrollPositions.set(scrollPersistenceKey, element.scrollTop);
    };
  }, [scrollPersistenceKey]);

  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [itemCount, loadMoreIfNeeded]);

  return { scrollRef, handleScroll };
}
