import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { useCanvasMemoryEntries, useEventExecutionsBatch, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";

import { resolveConsoleNode, useConsoleContext, type ConsoleContextValue } from "../ConsoleContext";
import { flattenMemoryEntries } from "./memoryRow";
import type { WidgetDataSource, WidgetRender } from "./types";
import {
  LOAD_MORE_STEP,
  computeDisplaySlice,
  computeEffectiveLimit,
  computeInitialDisplayCount,
  computeWidgetHasMore,
  isWidgetQueryLoading,
  useEagerInfinitePagination,
} from "./widgetPagination";
import { buildNodeNameMap, collectExecutionRows, collectRunRows } from "./widgetRowCollection";

// Re-export the row-collection and pagination helpers from this module so the
// existing spec imports (and any external callers) keep working after the
// split. Keep this section narrow — new helpers should be added to their
// home modules and imported from there directly.
export {
  DEFAULT_AGGREGATE_LIMIT,
  INITIAL_EAGER_ROWS,
  LOAD_MORE_STEP,
  WIDGET_MAX_EAGER_PAGES,
  computeDisplaySlice,
  computeEffectiveLimit,
  computeInitialDisplayCount,
  computeWidgetHasMore,
  isWidgetQueryLoading,
  shouldFetchNextWidgetPage,
} from "./widgetPagination";
export {
  buildDollarNodes,
  buildNodeNameMap,
  collectExecutionRows,
  collectRunRows,
  lastOutputData,
} from "./widgetRowCollection";

export interface WidgetDataResult {
  rows: unknown[];
  isLoading: boolean;
  error?: string;
  /** Server-reported total for sources that expose one (currently `runs`). */
  totalCount?: number;
  /**
   * Whether more rows can be revealed by calling `loadMore()`. Only meaningful
   * for progressive callers (the table widget). `false` for chart/number
   * panels that always render against the full configured limit.
   */
  hasMore?: boolean;
  /**
   * `true` while a `loadMore()`-triggered (or scroll-triggered) fetch is in
   * flight, distinct from the initial fill which is reported via `isLoading`.
   */
  isFetchingMore?: boolean;
  /**
   * Grow the per-widget display window by `LOAD_MORE_STEP` rows (capped at
   * the configured limit, if any). No-op for non-progressive callers.
   */
  loadMore?: () => void;
}

/**
 * Reactively fetch the dataset for a widget based on its declared data source.
 * Returns a uniform `{ rows, isLoading, error }` so renderers don't have to
 * deal with the underlying query specifics.
 *
 * - `memory` reads from the canvas memory entries and filters to the requested
 *   namespace. When `fieldPath` is set, rows are flat-mapped to the value at
 *   that path (lists are spread, scalars become single-row entries).
 * - `runs` reads from the canvas runs infinite query and exposes the API
 *   totalCount, which makes Number panels report the canvas run count without
 *   depending on how many pages have been loaded.
 * - `executions` reads from the infinite canvas events query and flattens
 *   their `executions[]` arrays. When a `node` reference is set, only matching
 *   executions are returned.
 *
 * Both `runs` and `executions` eagerly fetch pages until the per-widget
 * display window is satisfied or `WIDGET_MAX_EAGER_PAGES` is reached. Without
 * this, the runs widget would silently piggyback on whichever other consumer
 * (e.g. the canvas Runs sidebar) happened to paginate the shared infinite-
 * query cache, so the visible row count would jump non-monotonically.
 *
 * `needsNodeOutputs` controls the `runs` per-node side-load. Resolving
 * `$["node"].outputs` requires a `ListEventExecutions` call per visible run
 * (the runs API only returns lightweight execution refs). When the render
 * never references `$`, callers pass `false` so we skip those calls entirely
 * and, crucially, don't gate the panel's loading state on them — otherwise a
 * count KPI that only reads the API `totalCount` would spin until every
 * side-load resolved even though it never reads per-node data. Defaults to
 * `true` so a caller that forgets to pass it keeps the safe (fully-loaded)
 * behavior.
 *
 * `progressive` opts a caller into the per-widget pagination UX: the widget
 * starts at `INITIAL_EAGER_ROWS` and grows via `loadMore()` (button or scroll)
 * up to the configured `limit` (or unbounded when blank). Charts and numbers
 * stay non-progressive — they always aggregate against the full configured
 * limit at once, since partial aggregates would flash incorrect KPIs.
 */
export function useWidgetData(
  canvasId: string,
  dataSource: WidgetDataSource,
  needsNodeOutputs: boolean = true,
  progressive: boolean = false,
): WidgetDataResult {
  const ctx = useConsoleContext();
  const rawLimit = dataSource.kind === "memory" ? undefined : dataSource.limit;
  const effectiveLimit = computeEffectiveLimit(rawLimit, progressive);
  const initialFillTarget = computeInitialDisplayCount(effectiveLimit, progressive);

  const { displayCount, displaySlice, loadMore } = useDisplayWindow({
    dataSourceKind: dataSource.kind,
    progressive,
    effectiveLimit,
  });

  const memoryResult = useMemoryDataSourceResult(canvasId, dataSource);
  const executionsResult = useExecutionsDataSourceResult({
    canvasId,
    dataSource,
    ctx,
    progressive,
    displayCount,
    displaySlice,
    effectiveLimit,
    initialFillTarget,
    loadMore,
  });
  const runsResult = useRunsDataSourceResult({
    canvasId,
    dataSource,
    ctx,
    needsNodeOutputs,
    progressive,
    displayCount,
    displaySlice,
    effectiveLimit,
    initialFillTarget,
    loadMore,
  });

  if (dataSource.kind === "memory") return memoryResult;
  if (dataSource.kind === "runs") return runsResult;
  return executionsResult;
}

/**
 * Matches a canvas-style per-node reference (`$["node"]` / `$['node']`,
 * with optional whitespace before the bracket) inside a render config. We
 * key off the `$[` shape rather than a bare `$` so currency/label literals
 * like `prefix: "R$"` or `label: "Total $"` don't force the per-node
 * side-load.
 */
const RUN_NODE_REF_RE = /\$\s*\[/;

/**
 * Whether a widget render reads per-node run outputs via the `$["node"]`
 * syntax — in a literal field path or inside a `{{ }}` CEL template. Only the
 * `runs` data source side-loads per-node executions to populate the row's `$`
 * map, so when the render never references `$` the caller can pass the result
 * as `needsNodeOutputs={false}` to skip the `ListEventExecutions` batch and
 * avoid gating the panel's loading state on it. Scans the whole render via
 * `JSON.stringify` so every field/expression-bearing string (columns,
 * filters, sort, series, sparkline, row actions, …) is covered without
 * enumerating each render shape.
 */
export function renderNeedsRunNodeOutputs(render: WidgetRender | undefined): boolean {
  if (!render) return false;
  return RUN_NODE_REF_RE.test(JSON.stringify(render));
}

/**
 * Per-widget display window state. Tracks the current visible row count,
 * resets it on a meaningful "widget identity" change (data source kind or
 * `progressive` toggle), and exposes a `loadMore()` that bumps the count
 * up to the configured limit. Limit edits intentionally do not reset the
 * progressive window — the slice clamps via `computeDisplaySlice` and we'd
 * rather not yank the user back to the first page just because they bumped
 * the limit input.
 *
 * Non-progressive callers (chart, number) have no "Load more" UX, so they
 * always render against the full `effectiveLimit`. We deliberately bypass
 * the `displayCount` state for them: the state only resets on an identity
 * change, so reusing it would clamp the slice to a stale (smaller) value
 * when the author raises the limit live — making the panel silently ignore
 * the increase until it remounts.
 */
export function useDisplayWindow({
  dataSourceKind,
  progressive,
  effectiveLimit,
}: {
  dataSourceKind: WidgetDataSource["kind"];
  progressive: boolean;
  effectiveLimit: number;
}) {
  const [displayCount, setDisplayCount] = useState(() => computeInitialDisplayCount(effectiveLimit, progressive));
  const lastIdentityRef = useRef<{ kind: WidgetDataSource["kind"]; progressive: boolean }>({
    kind: dataSourceKind,
    progressive,
  });
  useEffect(() => {
    const last = lastIdentityRef.current;
    if (last.kind === dataSourceKind && last.progressive === progressive) return;
    lastIdentityRef.current = { kind: dataSourceKind, progressive };
    setDisplayCount(computeInitialDisplayCount(effectiveLimit, progressive));
  }, [dataSourceKind, progressive, effectiveLimit]);

  const loadMore = useCallback(() => {
    if (!progressive) return;
    setDisplayCount((c) => {
      const next = c + LOAD_MORE_STEP;
      return Number.isFinite(effectiveLimit) ? Math.min(next, effectiveLimit) : next;
    });
  }, [progressive, effectiveLimit]);

  if (!progressive) {
    return { displayCount: effectiveLimit, displaySlice: effectiveLimit, loadMore };
  }

  return {
    displayCount,
    displaySlice: computeDisplaySlice(displayCount, effectiveLimit),
    loadMore,
  };
}

function useMemoryDataSourceResult(canvasId: string, dataSource: WidgetDataSource): WidgetDataResult {
  const enabled = dataSource.kind === "memory";
  const query = useCanvasMemoryEntries(canvasId, enabled);
  const rows = useMemo(() => {
    if (dataSource.kind !== "memory") return [];
    return flattenMemoryEntries(query.data ?? [], dataSource.namespace, dataSource.fieldPath);
  }, [dataSource, query.data]);
  return { rows, isLoading: query.isLoading, error: errorMessage(query.error) };
}

function useExecutionsDataSourceResult({
  canvasId,
  dataSource,
  ctx,
  progressive,
  displayCount,
  displaySlice,
  effectiveLimit,
  initialFillTarget,
  loadMore,
}: {
  canvasId: string;
  dataSource: WidgetDataSource;
  ctx: ConsoleContextValue | undefined;
  progressive: boolean;
  displayCount: number;
  displaySlice: number;
  effectiveLimit: number;
  initialFillTarget: number;
  loadMore: () => void;
}): WidgetDataResult {
  const enabled = dataSource.kind === "executions";
  const query = useInfiniteCanvasRuns(canvasId, {}, enabled);
  // Memoize `pages` so empty-array fallback identity stays stable across
  // renders — keeps the downstream `useMemo` deps from busting every cycle.
  const pages = useMemo(() => query.data?.pages ?? [], [query.data]);
  const pageCount = pages.length;

  const targetNodeId = useMemo(() => {
    if (dataSource.kind !== "executions" || !dataSource.node) return undefined;
    return resolveConsoleNode(ctx, dataSource.node)?.node.id;
  }, [dataSource, ctx]);

  // Count only the executions that survive the node filter so the eager
  // fill target and the "Load more" affordance track the *visible* rows.
  // Counting every loaded execution would otherwise show a "Load more"
  // button that reveals nothing for a node with sparse executions.
  const loadedRowCount = useMemo(() => {
    if (!enabled) return 0;
    let count = 0;
    for (const page of pages) {
      for (const run of page?.runs ?? []) {
        for (const exec of run.executions ?? []) {
          if (targetNodeId && exec.nodeId !== targetNodeId) continue;
          count++;
        }
      }
    }
    return count;
  }, [enabled, pages, targetNodeId]);

  useEagerInfinitePagination({
    enabled,
    fillTarget: displaySlice,
    loadedRowCount,
    pageCount,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetching: query.isFetching,
    fetchNextPage: query.fetchNextPage,
  });

  const rows = useMemo(() => {
    if (dataSource.kind !== "executions") return [];
    const nodeNameById = buildNodeNameMap(ctx?.nodes);
    return collectExecutionRows(pages, targetNodeId, nodeNameById, displaySlice);
  }, [dataSource, pages, ctx, targetNodeId, displaySlice]);

  const isLoading = isWidgetQueryLoading({
    queryIsLoading: query.isLoading,
    enabled,
    hasNextPage: query.hasNextPage,
    loadedRowCount,
    fillTarget: initialFillTarget,
    pageCount,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetching: query.isFetching,
  });

  const hasMore = computeWidgetHasMore({
    progressive,
    displayCount,
    effectiveLimit,
    loadedRowCount,
    hasNextPage: query.hasNextPage,
    pageCount,
  });

  const isFetchingMore = enabled && query.isFetchingNextPage && !isLoading && hasMore;
  const paginationFields = progressive ? { hasMore, isFetchingMore, loadMore } : {};
  return { rows, isLoading, error: errorMessage(query.error), ...paginationFields };
}

function useRunsDataSourceResult({
  canvasId,
  dataSource,
  ctx,
  needsNodeOutputs,
  progressive,
  displayCount,
  displaySlice,
  effectiveLimit,
  initialFillTarget,
  loadMore,
}: {
  canvasId: string;
  dataSource: WidgetDataSource;
  ctx: ConsoleContextValue | undefined;
  needsNodeOutputs: boolean;
  progressive: boolean;
  displayCount: number;
  displaySlice: number;
  effectiveLimit: number;
  initialFillTarget: number;
  loadMore: () => void;
}): WidgetDataResult {
  const enabled = dataSource.kind === "runs";
  const query = useInfiniteCanvasRuns(canvasId, {}, enabled);
  // Memoize `pages` so empty-array fallback identity stays stable across
  // renders — keeps the downstream `useMemo` deps from busting every cycle.
  const pages = useMemo(() => query.data?.pages ?? [], [query.data]);
  const pageCount = pages.length;

  const loadedRowCount = useMemo(() => {
    if (!enabled) return 0;
    let count = 0;
    for (const page of pages) count += page?.runs?.length ?? 0;
    return count;
  }, [enabled, pages]);

  useEagerInfinitePagination({
    enabled,
    fillTarget: displaySlice,
    loadedRowCount,
    pageCount,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetching: query.isFetching,
    fetchNextPage: query.fetchNextPage,
  });

  // Collect unique root-event ids for the visible run page so we can lazy-
  // fetch their per-node executions (with `outputs`) via `ListEventExecutions`.
  // The runs API only returns lightweight execution refs without outputs, so
  // we have to side-load the full executions to support `$["node"].outputs`.
  const runRootEventIds = useMemo(() => {
    if (dataSource.kind !== "runs" || !needsNodeOutputs) return [] as string[];
    const seen = new Set<string>();
    const ids: string[] = [];
    let count = 0;
    for (const page of pages) {
      for (const run of page?.runs ?? []) {
        if (count >= displaySlice) break;
        count++;
        const id = run.rootEvent?.id;
        if (id && !seen.has(id)) {
          seen.add(id);
          ids.push(id);
        }
      }
      if (count >= displaySlice) break;
    }
    return ids;
  }, [dataSource, pages, displaySlice, needsNodeOutputs]);

  const { queries: runExecutionQueries, isLoading: runExecutionsLoading } = useEventExecutionsBatch(
    canvasId,
    runRootEventIds,
  );

  const executionsByRootEventId = useMemo(() => {
    const map = new Map<string, CanvasesCanvasNodeExecution[]>();
    runRootEventIds.forEach((eventId, index) => {
      const data = runExecutionQueries[index]?.data;
      if (!data?.executions) return;
      map.set(eventId, data.executions as CanvasesCanvasNodeExecution[]);
    });
    return map;
  }, [runRootEventIds, runExecutionQueries]);

  const rows = useMemo(() => {
    if (dataSource.kind !== "runs") return [];
    const nodeNameById = buildNodeNameMap(ctx?.nodes);
    return collectRunRows(pages, nodeNameById, displaySlice, executionsByRootEventId);
  }, [dataSource, pages, ctx, displaySlice, executionsByRootEventId]);

  const initialFillLoading = isWidgetQueryLoading({
    queryIsLoading: query.isLoading,
    enabled,
    hasNextPage: query.hasNextPage,
    loadedRowCount,
    fillTarget: initialFillTarget,
    pageCount,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetching: query.isFetching,
  });

  // In progressive mode the table stays mounted across `Load more` clicks,
  // each of which appends more root-event ids to `runRootEventIds` and
  // re-flips `runExecutionsLoading` to true. Gating `isLoading` on it would
  // hide the already-rendered rows behind a spinner every time the user
  // grows the window, so progressive callers absorb side-loads as
  // background work — the affected `$["node"]` cells render `-` until they
  // resolve. Non-progressive callers (chart/number) keep the existing
  // strict gate so partial aggregates don't flash.
  const sideLoadsBlock = !progressive && runExecutionsLoading;
  const isLoading = initialFillLoading || sideLoadsBlock;

  const hasMore = computeWidgetHasMore({
    progressive,
    displayCount,
    effectiveLimit,
    loadedRowCount,
    hasNextPage: query.hasNextPage,
    pageCount,
  });

  const isFetchingMore = enabled && query.isFetchingNextPage && !isLoading && hasMore;
  const paginationFields = progressive ? { hasMore, isFetchingMore, loadMore } : {};
  return {
    rows,
    isLoading,
    error: errorMessage(query.error),
    totalCount: query.data?.pages?.[0]?.totalCount,
    ...paginationFields,
  };
}

function errorMessage(error: unknown): string | undefined {
  return error ? String(error) : undefined;
}
