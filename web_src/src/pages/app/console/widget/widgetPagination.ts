/**
 * Per-widget pagination helpers shared across the runs/executions data
 * sources. Keeps `useWidgetData` focused on orchestration while the pure
 * limit/slice/should-fetch math lives here (and stays trivially unit-testable
 * via `useWidgetData.pagination.spec.ts`).
 */

import { useEffect } from "react";

/**
 * Upper bound on how many infinite-query pages a widget will eagerly pull
 * before giving up. Each page returns 25 rows, so 20 pages covers up to ~500
 * rows — more than any dashboard panel currently asks for, while bounding
 * memory and network cost for canvases with very long histories.
 *
 * Applied to both `executions` (events feed) and `runs` (canvas runs feed)
 * data sources so the widget never falls back on shared-cache pagination
 * driven by other consumers (e.g. the canvas Runs sidebar).
 */
export const WIDGET_MAX_EAGER_PAGES = 20;

/**
 * How many rows the progressive table widget eagerly fills on first render
 * before showing the "Load more" affordance. Picked to comfortably exceed a
 * typical viewport so the user sees a scrollbar without having to click.
 */
export const INITIAL_EAGER_ROWS = 100;

/**
 * Step size used by `loadMore()` to grow the per-widget display window.
 */
export const LOAD_MORE_STEP = 100;

/**
 * Default cap applied to non-progressive callers (chart, number) when the
 * data source's `limit` is left undefined. Keeps aggregations bounded so a
 * blank limit on a chart doesn't silently pull every page in the canvas.
 */
export const DEFAULT_AGGREGATE_LIMIT = 100;

/**
 * Resolve the data source's user-configured limit to a concrete row cap.
 *
 * - `progressive` (the table widget): a missing/non-positive limit means
 *   "load all rows on demand", expressed as `Infinity`. The widget pages in
 *   chunks via `loadMore` so this never translates to an unbounded fetch on
 *   its own — `WIDGET_MAX_EAGER_PAGES` still caps total work.
 * - non-progressive (chart, number): a missing/non-positive limit falls
 *   back to `DEFAULT_AGGREGATE_LIMIT` so aggregated panels stay bounded.
 */
export function computeEffectiveLimit(rawLimit: number | undefined, progressive: boolean): number {
  if (typeof rawLimit === "number" && rawLimit > 0) return rawLimit;
  return progressive ? Number.POSITIVE_INFINITY : DEFAULT_AGGREGATE_LIMIT;
}

/**
 * Initial value for the per-widget display window. Non-progressive callers
 * always render against the full effective limit (no `loadMore` UI), so
 * their `displayCount` is fixed at `effectiveLimit` from the start.
 */
export function computeInitialDisplayCount(effectiveLimit: number, progressive: boolean): number {
  if (!progressive) return effectiveLimit;
  if (!Number.isFinite(effectiveLimit)) return INITIAL_EAGER_ROWS;
  return Math.min(INITIAL_EAGER_ROWS, effectiveLimit);
}

/**
 * The number of rows to slice out of the loaded pages for the current
 * render. Always clamps to `effectiveLimit` even when `displayCount`
 * temporarily exceeds it (e.g. the user lowered the limit live).
 */
export function computeDisplaySlice(displayCount: number, effectiveLimit: number): number {
  return Math.min(displayCount, effectiveLimit);
}

/**
 * How many source rows to materialize in progressive mode. Collect every
 * already-loaded row (capped by the configured limit) so the table can
 * filter+sort the full loaded set, then slice the display window — trend
 * baselines must see neighbors that sort into place after the visible rows.
 */
export function computeTrendCollectLimit(
  displaySlice: number,
  loadedRowCount: number,
  effectiveLimit: number = Number.POSITIVE_INFINITY,
): number {
  const loaded = Math.max(displaySlice, loadedRowCount);
  if (!Number.isFinite(effectiveLimit)) return loaded;
  return Math.min(loaded, effectiveLimit);
}

/**
 * Split a collected row list into the visible display window and the optional
 * first already-loaded row beyond it.
 *
 * @deprecated Progressive tables now pass the full loaded set plus
 * `displayCount` so filter/sort run before the display slice. Kept for specs
 * that still exercise the helper.
 */
export function splitDisplayRowsWithTrendPeek<T>(
  collected: T[],
  displaySlice: number,
): { rows: T[]; nextLoadedRow: T | undefined } {
  if (collected.length <= displaySlice) {
    return { rows: collected, nextLoadedRow: undefined };
  }
  return {
    rows: collected.slice(0, displaySlice),
    nextLoadedRow: collected[displaySlice],
  };
}

/**
 * Whether the progressive widget should expose a "Load more" affordance:
 * either there are already-loaded rows we haven't shown yet, or there are
 * more pages to fetch and we still have eager-page budget. Returns `false`
 * for non-progressive callers (they always fill to their configured limit).
 */
export function computeWidgetHasMore({
  progressive,
  displayCount,
  effectiveLimit,
  loadedRowCount,
  hasNextPage,
  pageCount,
}: {
  progressive: boolean;
  displayCount: number;
  effectiveLimit: number;
  loadedRowCount: number;
  hasNextPage: boolean | undefined;
  pageCount: number;
}): boolean {
  if (!progressive) return false;
  if (displayCount >= effectiveLimit) return false;
  const canShowMoreLoaded = loadedRowCount > displayCount;
  const canFetchMore = hasNextPage === true && pageCount < WIDGET_MAX_EAGER_PAGES;
  return canShowMoreLoaded || canFetchMore;
}

/**
 * Module-level set of `flightKey`s that currently have a `fetchNextPage`
 * call in flight. React Query dedupes network requests per query key, but
 * it flips `isFetchingNextPage` asynchronously — so N widgets that pass
 * `shouldFetchNextWidgetPage` in the same commit will each call
 * `fetchNextPage()` before the flag reaches them, producing duplicate
 * `before=` requests for the same cursor (live capture on the SaaS console
 * showed 4× duplicates on the first pagination steps).
 *
 * This set gives us a synchronous single-flight guarantee across all widget
 * effects that share the same infinite-query key. It is intentionally
 * module-scoped (per browser tab) — pagination is a global concern of the
 * shared cache, not a per-widget one.
 */
const inFlightEagerPagination = new Set<string>();

/**
 * Test-only escape hatch: clears the module-level in-flight set. Guarded
 * so it stays a no-op in production bundles.
 */
export function __resetEagerPaginationInFlight(): void {
  inFlightEagerPagination.clear();
}

/**
 * Drives widget-owned eager pagination of an infinite query: fetches pages
 * until the current `fillTarget` is reached, the source runs out of pages,
 * or `WIDGET_MAX_EAGER_PAGES` is hit. Re-runs after each page arrives so
 * `loadMore` (bumping `fillTarget`) flows through the same mechanism with
 * no special-case fetch path.
 *
 * `flightKey` is a stable string identifying the shared infinite-query
 * cache entry (typically `JSON.stringify(queryKey)`). Widget effects on the
 * same key coordinate through the module-level `inFlightEagerPagination`
 * set so only one `fetchNextPage()` call is dispatched per cursor advance
 * even when multiple widgets satisfy `shouldFetchNextWidgetPage` in the
 * same React commit.
 */
export function useEagerInfinitePagination({
  enabled,
  fillTarget,
  loadedRowCount,
  pageCount,
  hasNextPage,
  isFetchingNextPage,
  isFetching,
  fetchNextPage,
  flightKey,
}: {
  enabled: boolean;
  fillTarget: number;
  loadedRowCount: number;
  pageCount: number;
  hasNextPage: boolean | undefined;
  isFetchingNextPage: boolean;
  isFetching: boolean;
  fetchNextPage: () => Promise<unknown> | unknown;
  flightKey: string;
}) {
  useEffect(() => {
    if (
      !shouldFetchNextWidgetPage({
        enabled,
        fillTarget,
        loadedRowCount,
        pageCount,
        hasNextPage,
        isFetchingNextPage,
        isFetching,
      })
    ) {
      return;
    }
    if (inFlightEagerPagination.has(flightKey)) return;
    inFlightEagerPagination.add(flightKey);
    const clear = () => {
      inFlightEagerPagination.delete(flightKey);
    };
    let result: Promise<unknown> | unknown;
    try {
      result = fetchNextPage();
    } catch {
      clear();
      return;
    }
    if (result && typeof (result as Promise<unknown>).then === "function") {
      (result as Promise<unknown>).then(clear, clear);
      return;
    }
    clear();
  }, [
    enabled,
    fillTarget,
    loadedRowCount,
    pageCount,
    hasNextPage,
    isFetchingNextPage,
    isFetching,
    fetchNextPage,
    flightKey,
  ]);
}

export function shouldFetchNextWidgetPage({
  enabled,
  fillTarget,
  loadedRowCount,
  pageCount,
  hasNextPage,
  isFetchingNextPage,
  isFetching,
}: {
  enabled: boolean;
  fillTarget: number;
  loadedRowCount: number;
  pageCount: number;
  hasNextPage: boolean | undefined;
  isFetchingNextPage: boolean;
  isFetching: boolean;
}): boolean {
  if (!enabled || !hasNextPage) return false;
  if (isFetchingNextPage || isFetching) return false;
  if (loadedRowCount >= fillTarget) return false;
  return pageCount < WIDGET_MAX_EAGER_PAGES;
}

export function isWidgetQueryLoading({
  queryIsLoading,
  enabled,
  hasNextPage,
  loadedRowCount,
  fillTarget,
  pageCount,
  isFetchingNextPage,
  isFetching,
}: {
  queryIsLoading: boolean;
  enabled: boolean;
  hasNextPage: boolean | undefined;
  loadedRowCount: number;
  fillTarget: number;
  pageCount: number;
  isFetchingNextPage: boolean;
  isFetching: boolean;
}): boolean {
  if (queryIsLoading) return true;
  return (
    enabled &&
    hasNextPage === true &&
    loadedRowCount < fillTarget &&
    pageCount < WIDGET_MAX_EAGER_PAGES &&
    (isFetchingNextPage || isFetching)
  );
}
