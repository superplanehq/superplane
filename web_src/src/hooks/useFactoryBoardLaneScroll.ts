import { useCallback, useEffect, useLayoutEffect, useRef } from "react";

const persistedScrollTops = new Map<string, number>();

export type FactoryBoardLaneId = "backlog" | "verify" | "done" | `step-${number}`;

export function factoryBoardLaneScrollKey(workspaceId: string, lineId: string, lane: FactoryBoardLaneId): string {
  return `${workspaceId}:${lineId}:${lane}`;
}

export function resetFactoryBoardLaneScrollPositions() {
  persistedScrollTops.clear();
}

export function useFactoryBoardLaneScroll(scrollPersistenceKey: string | undefined, ready = true) {
  const scrollRef = useRef<HTMLUListElement>(null);
  const persistenceKeyRef = useRef(scrollPersistenceKey);
  const readyRef = useRef(ready);
  const lastReadyRef = useRef(ready);
  readyRef.current = ready;

  const handleScroll = useCallback(
    (element: HTMLElement) => {
      if (!scrollPersistenceKey || !readyRef.current) {
        return;
      }
      persistedScrollTops.set(scrollPersistenceKey, element.scrollTop);
    },
    [scrollPersistenceKey],
  );

  useLayoutEffect(() => {
    const element = scrollRef.current;
    const previousKey = persistenceKeyRef.current;
    const previousReady = lastReadyRef.current;

    if (element && previousKey && previousKey !== scrollPersistenceKey && previousReady && ready) {
      persistedScrollTops.set(previousKey, element.scrollTop);
    }

    persistenceKeyRef.current = scrollPersistenceKey;
    lastReadyRef.current = ready;

    if (!element || !ready) {
      return;
    }

    if (!scrollPersistenceKey) {
      element.scrollTop = 0;
      return;
    }

    element.scrollTop = persistedScrollTops.get(scrollPersistenceKey) ?? 0;
  }, [ready, scrollPersistenceKey]);

  useEffect(() => {
    const element = scrollRef.current;

    return () => {
      const key = persistenceKeyRef.current;
      if (!element || !key || !readyRef.current) {
        return;
      }
      persistedScrollTops.set(key, element.scrollTop);
    };
  }, [ready, scrollPersistenceKey]);

  return { scrollRef, handleScroll };
}
