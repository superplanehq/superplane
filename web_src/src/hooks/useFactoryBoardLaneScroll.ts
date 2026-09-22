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
  const persistenceKeyRef = useRef(scrollPersistenceKey);

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
    const previousKey = persistenceKeyRef.current;

    if (element && previousKey && previousKey !== scrollPersistenceKey) {
      persistedScrollTops.set(previousKey, element.scrollTop);
    }

    persistenceKeyRef.current = scrollPersistenceKey;

    if (!element) {
      return;
    }

    if (!scrollPersistenceKey) {
      element.scrollTop = 0;
      return;
    }

    element.scrollTop = persistedScrollTops.get(scrollPersistenceKey) ?? 0;
  }, [scrollPersistenceKey]);

  useEffect(() => {
    const element = scrollRef.current;

    return () => {
      const key = persistenceKeyRef.current;
      if (!element || !key) {
        return;
      }
      persistedScrollTops.set(key, element.scrollTop);
    };
  }, [scrollPersistenceKey]);

  return { scrollRef, handleScroll };
}
