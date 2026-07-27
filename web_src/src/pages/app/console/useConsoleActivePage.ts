import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { recordLastVisitedConsolePage, resolveActiveConsolePage } from "@/lib/lastVisitedConsolePage";

/**
 * Owns the "which console page is active?" state, and reflects it back
 * to the URL and to per-canvas last-visited storage.
 *
 * The hook is intentionally split into two effect groups so the caller
 * can insert `useConsolePagesState` between them:
 *
 * 1. Initial resolution — sets `activePageId` from the priority chain
 *    (URL `?page=` → last-visited → first persisted page). Runs on
 *    mount and re-runs when the URL param changes.
 * 2. Post-render sync — driven off `livePageIds` (the current in-memory
 *    page list, including unsaved adds), so:
 *      - Writing the `?page=` URL param and the last-visited entry
 *        happens with the actual live count, not just the persisted
 *        count. Without this, adding a second page briefly writes the
 *        wrong URL and the next persisted refetch resets the tab back
 *        to the first page.
 *      - Re-resolving when the active id no longer exists in the live
 *        page list falls back gracefully (a removed page, a page id
 *        that no longer matches after a rename, etc).
 *
 * Callers use `useConsoleActivePageInitial` first, then invoke
 * `useConsoleActivePageSync` after `useConsolePagesState` has produced
 * the current `localPages`.
 */
export function useConsoleActivePageInitial({
  canvasId,
  persistedPageIds,
}: {
  canvasId: string | undefined;
  persistedPageIds: string[];
}) {
  const [searchParams] = useSearchParams();
  const rawPageParam = searchParams.get("page");
  const persistedIdsMemo = useMemo(() => persistedPageIds, [persistedPageIds]);
  const [activePageId, setActivePageId] = useState<string | null>(() =>
    resolveActiveConsolePage({
      canvasId: canvasId ?? "",
      pageParam: rawPageParam,
      availablePageIds: persistedIdsMemo,
    }),
  );
  return { activePageId, setActivePageId, rawPageParam, persistedIdsMemo };
}

/**
 * Runs after `useConsolePagesState` so URL / storage sync decisions can
 * use the *live* page list. Never called with a stale `livePageIds`.
 */
export function useConsoleActivePageSync({
  canvasId,
  livePageIds,
  activePageId,
  setActivePageId,
  rawPageParam,
  persistedPageIds,
}: {
  canvasId: string | undefined;
  livePageIds: string[];
  activePageId: string | null;
  setActivePageId: (next: string | null) => void;
  rawPageParam: string | null;
  persistedPageIds: string[];
}) {
  const [searchParams, setSearchParams] = useSearchParams();
  const liveIdsMemo = useMemo(() => livePageIds, [livePageIds]);
  const persistedIdsMemo = useMemo(() => persistedPageIds, [persistedPageIds]);
  const liveCount = liveIdsMemo.length;

  const previousParamRef = useRef<string | null>(rawPageParam);
  useEffect(() => {
    // Re-resolve on two independent triggers:
    //   1. The URL `?page=` param changed. This includes back/forward
    //      navigation, deep links, and manual URL edits. When the new
    //      value matches a live page we adopt it, even if the current
    //      active id is still valid — otherwise the grid gets stuck on
    //      the previous tab while the URL says otherwise.
    //   2. The current active id is no longer present in the live page
    //      list (a page was removed, a stale query populated, etc.).
    // Adopting our own URL writes is a no-op because the resolved value
    // will already match `activePageId`.
    const paramChanged = rawPageParam !== previousParamRef.current;
    const activeStale = !!activePageId && !liveIdsMemo.includes(activePageId);
    previousParamRef.current = rawPageParam;

    if (!paramChanged && !activeStale && activePageId) return;

    const next = resolveActiveConsolePage({
      canvasId: canvasId ?? "",
      pageParam: rawPageParam,
      availablePageIds: liveIdsMemo.length > 0 ? liveIdsMemo : persistedIdsMemo,
    });
    if (next !== activePageId) setActivePageId(next);
  }, [activePageId, canvasId, liveIdsMemo, persistedIdsMemo, rawPageParam, setActivePageId]);

  useEffect(() => {
    if (!canvasId || !activePageId) return;
    // The implicit single-page id is never worth restoring; the same
    // fallback resolves it just as fast on the next visit.
    if (liveCount <= 1) return;
    recordLastVisitedConsolePage(canvasId, activePageId);
  }, [canvasId, activePageId, liveCount]);

  useEffect(() => {
    if (!activePageId) return;
    const currentParam = searchParams.get("page");
    const shouldWriteParam = liveCount > 1;

    if (shouldWriteParam && currentParam !== activePageId) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set("page", activePageId);
          return next;
        },
        { replace: true },
      );
      return;
    }

    if (!shouldWriteParam && currentParam !== null) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.delete("page");
          return next;
        },
        { replace: true },
      );
    }
  }, [activePageId, liveCount, searchParams, setSearchParams]);
}
