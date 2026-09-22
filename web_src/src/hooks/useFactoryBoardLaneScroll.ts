import { useCallback, useEffect, useLayoutEffect, useRef, type UIEvent } from "react";

const persistedScrollTops = new Map<string, number>();

export type FactoryBoardLaneId = "backlog" | "verify" | "done" | `step-${number}`;

export function factoryBoardLaneScrollKey(workspaceId: string, lineId: string, lane: FactoryBoardLaneId): string {
  return `${workspaceId}:${lineId}:${lane}`;
}

export function resetFactoryBoardLaneScrollPositions() {
  persistedScrollTops.clear();
}

export function useFactoryBoardLaneScroll(scrollPersistenceKey: string | undefined) {
  const scrollRef = useRef<HTMLUListElement>(null);

  const handleScroll = useCallback(
    (event: UIEvent<HTMLUListElement>) => {
      if (!scrollPersistenceKey) {
        return;
      }
      persistedScrollTops.set(scrollPersistenceKey, event.currentTarget.scrollTop);
    },
    [scrollPersistenceKey],
  );

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element || !scrollPersistenceKey) {
      return;
    }

    const scrollTop = persistedScrollTops.get(scrollPersistenceKey);
    if (scrollTop == null) {
      return;
    }

    element.scrollTop = scrollTop;
  }, [scrollPersistenceKey]);

  useEffect(() => {
    const element = scrollRef.current;

    return () => {
      if (!element || !scrollPersistenceKey) {
        return;
      }
      persistedScrollTops.set(scrollPersistenceKey, element.scrollTop);
    };
  }, [scrollPersistenceKey]);

  return { scrollRef, handleScroll };
}
