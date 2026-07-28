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

  // React Router keeps this component mounted across canvases (the
  // `apps/:id` route matches both), so `activePageId` would otherwise
  // survive a canvas switch and leak the previous canvas's tab into
  // the new one. When that stale id happens to coincide with a page
  // id on the new canvas (e.g., the implicit `main`), the last-visited
  // effect below then writes it into `localStorage[<newCanvasId>]`,
  // overwriting the new canvas's real preference.
  //
  // Standard React "reset state on prop change during render" pattern
  // (https://react.dev/reference/react/useState#storing-information-from-previous-renders):
  // clear to `null` rather than re-resolving here, because on the
  // render where `canvasId` first changes `persistedPageIds` typically
  // still holds the previous canvas's data (RQ has not swapped caches
  // yet) — a re-resolve against that stale list can return the wrong
  // canvas's page. Case 2a in the sync effect hydrates against the
  // new canvas's pages once they arrive, and the last-visited effect
  // stays inert while `activePageId` is `null`.
  const [trackedCanvasId, setTrackedCanvasId] = useState(canvasId);
  if (trackedCanvasId !== canvasId) {
    setTrackedCanvasId(canvasId);
    setActivePageId(null);
  }

  return { activePageId, setActivePageId, rawPageParam, persistedIdsMemo };
}

type ReconcileInputs = {
  canvasId: string | undefined;
  activePageId: string | null;
  rawPageParam: string | null;
  liveIds: string[];
  persistedIds: string[];
  paramChanged: boolean;
};

/**
 * Compute the desired `activePageId` for the current render, or `null`
 * to indicate "leave state alone; fall through to URL projection". The
 * three adoption cases are mutually exclusive: at most one runs per
 * render, and each moves the system strictly closer to a steady state
 * where the URL param and `activePageId` agree.
 */
function nextActivePageIdFromReconciliation({
  canvasId,
  activePageId,
  rawPageParam,
  liveIds,
  persistedIds,
  paramChanged,
}: ReconcileInputs): string | null {
  const available = liveIds.length > 0 ? liveIds : persistedIds;

  // Case 1: URL param changed (external navigation, deep link, back
  // button, or our own prior projection write catching up). Adopt
  // when the resolved id differs from state; otherwise fall through
  // so a stale/invalid URL (`?page=ghost`) gets rewritten to match
  // the fallback state instead of being left in place.
  if (paramChanged) {
    const next = resolveActiveConsolePage({
      canvasId: canvasId ?? "",
      pageParam: rawPageParam,
      availablePageIds: available,
    });
    if (next !== activePageId) return next;
  }

  // Case 2a: async load — `useConsoleActivePageInitial` returned `null`
  // because the console query was still loading; adopt now that pages
  // are known, even though the URL param never *changed*.
  //
  // Uses the same `available` fallback (live → persisted) as Cases 1
  // and 2b so we can hydrate off `persistedPageIds` on the render
  // where the query has already populated but `useConsolePagesState`
  // has not yet mirrored the committed pages into its local `pages`
  // state. Without this, a multi-page console can briefly render the
  // empty state or the wrong tab in that one-tick window.
  if (!activePageId && available.length > 0) {
    return resolveActiveConsolePage({ canvasId: canvasId ?? "", pageParam: rawPageParam, availablePageIds: available });
  }

  // Case 2b: active id is no longer present in the authoritative page
  // list (page removed / renamed / stale query populated). Uses the
  // same `available` fallback as Cases 1 and 2a so that on the
  // transitional render where the query has just populated
  // (`persistedIds` non-empty) but local state has not yet mirrored it
  // (`liveIds` still empty), a valid `activePageId` from persisted is
  // not spuriously treated as stale.
  if (activePageId && available.length > 0 && !available.includes(activePageId)) {
    const next = resolveActiveConsolePage({
      canvasId: canvasId ?? "",
      pageParam: rawPageParam,
      availablePageIds: available,
    });
    if (next !== activePageId) return next;
  }

  return null;
}

function writePageParam(
  setSearchParams: (updater: (prev: URLSearchParams) => URLSearchParams, options: { replace: boolean }) => void,
  value: string | null,
) {
  setSearchParams(
    (prev) => {
      const next = new URLSearchParams(prev);
      if (value === null) next.delete("page");
      else next.set("page", value);
      return next;
    },
    { replace: true },
  );
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
  const [, setSearchParams] = useSearchParams();
  const liveIdsMemo = useMemo(() => livePageIds, [livePageIds]);
  const persistedIdsMemo = useMemo(() => persistedPageIds, [persistedPageIds]);
  const liveCount = liveIdsMemo.length;
  // `effectiveCount` mirrors the `available` fallback used inside the
  // reconciliation effect: prefer live, fall back to persisted. Case 3
  // and the last-visited persistence effect both key off this so we do
  // not remove the `?page=` param on the transitional render where
  // live is still empty but persisted has already populated.
  const effectiveCount = liveCount > 0 ? liveCount : persistedIdsMemo.length;

  const previousParamRef = useRef<string | null>(rawPageParam);
  useEffect(() => {
    // Reconcile URL <-> state in a single effect so the two directions
    // do not race each other. The prior implementation split adoption
    // (URL -> state) from projection (state -> URL) into two effects,
    // but during the render where a URL change was mid-adoption the
    // projection effect closed over the pre-adoption `activePageId`
    // and wrote the old value back to the URL — which then re-triggered
    // adoption on the next render and looped indefinitely.
    const paramChanged = rawPageParam !== previousParamRef.current;
    previousParamRef.current = rawPageParam;

    const resolved = nextActivePageIdFromReconciliation({
      canvasId,
      activePageId,
      rawPageParam,
      liveIds: liveIdsMemo,
      persistedIds: persistedIdsMemo,
      paramChanged,
    });
    if (resolved !== null) {
      setActivePageId(resolved);
      return;
    }

    // No adoption fired — project state into the URL. Use
    // `effectiveCount` (live → persisted fallback) so the URL param is
    // not stripped on the transitional render where live is empty but
    // persisted has already populated.
    if (!activePageId) return;
    const shouldWriteParam = effectiveCount > 1;
    if (shouldWriteParam && rawPageParam !== activePageId) {
      writePageParam(setSearchParams, activePageId);
      return;
    }
    if (!shouldWriteParam && rawPageParam !== null) {
      writePageParam(setSearchParams, null);
    }
  }, [
    activePageId,
    canvasId,
    effectiveCount,
    liveIdsMemo,
    persistedIdsMemo,
    rawPageParam,
    setActivePageId,
    setSearchParams,
  ]);

  useEffect(() => {
    if (!canvasId || !activePageId) return;
    // The implicit single-page id is never worth restoring; the same
    // fallback resolves it just as fast on the next visit.
    if (effectiveCount <= 1) return;

    // Membership guard: only persist a page id that we can confirm
    // belongs to the current canvas. React Router keeps this
    // component mounted across canvases, so on a canvas switch the
    // in-memory `activePageId` may briefly hold the previous canvas's
    // page id — either because the render-time reset in
    // `useConsoleActivePageInitial` has not run yet, or because Case
    // 1 in the reconciliation effect adopted a URL param against the
    // previous canvas's `persistedPageIds` (RQ has not swapped caches
    // yet). Without this check the write below would overwrite the
    // new canvas's real last-visited entry with a foreign page id.
    // The reconciliation effect will re-resolve `activePageId` once
    // the new canvas's pages arrive, at which point the write fires
    // with the correct value.
    const availableForCurrentCanvas = liveIdsMemo.length > 0 ? liveIdsMemo : persistedIdsMemo;
    if (!availableForCurrentCanvas.includes(activePageId)) return;

    recordLastVisitedConsolePage(canvasId, activePageId);
  }, [canvasId, activePageId, effectiveCount, liveIdsMemo, persistedIdsMemo]);
}
