import { useCallback, useMemo } from "react";

import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { useEventExecutionsBatch, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import {
  hasRunStatusTriggerFilters,
  runMatchesStatusTriggerFilters,
  triggerFilterCanMatch,
  type RunStatusTriggerFilters,
  type TriggerFilterMatchOptions,
  type TriggerReferenceResolver,
} from "@/ui/Runs/runStatusTriggerFilter";

import { resolveConsoleTrigger, type ConsoleContextValue } from "../ConsoleContext";
import { makeRunsFlightKey } from "./runsWidgetQuery";
import type { WidgetDataResult, WidgetDataSource } from "./types";
import {
  computeRunsDataSourceLoading,
  computeTrendCollectLimit,
  computeWidgetHasMore,
  useEagerInfinitePagination,
} from "./widgetPagination";
import { buildNodeNameMap, collectRunRows } from "./widgetRowCollection";

export function useConsoleTriggerMatch(ctx: ConsoleContextValue | undefined) {
  const resolveTrigger = useCallback((reference: string) => resolveConsoleTrigger(ctx, reference)?.node.id, [ctx]);
  const triggerMatchOptions = useMemo(
    (): TriggerFilterMatchOptions => ({ nodeCatalogSize: ctx?.nodes?.length ?? 0 }),
    [ctx?.nodes?.length],
  );
  return { resolveTrigger, triggerMatchOptions };
}

export function useRunsDataSourceResult({
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
  skipEagerFill,
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
  skipEagerFill: boolean;
}): WidgetDataResult {
  const enabled = dataSource.kind === "runs";
  const query = useInfiniteCanvasRuns(canvasId, {}, enabled);
  const pages = useMemo(() => query.data?.pages ?? [], [query.data]);
  const pageCount = pages.length;

  const loadedRowCount = useMemo(() => countLoadedRuns(pages, enabled), [enabled, pages]);
  const runsFilters = useMemo(() => runsFiltersFromDataSource(dataSource), [dataSource]);
  const filtersActive = hasRunStatusTriggerFilters(runsFilters);
  const { resolveTrigger, triggerMatchOptions } = useConsoleTriggerMatch(ctx);
  const collectLimit = computeRunsCollectLimit({
    filtersActive,
    progressive,
    displaySlice,
    loadedRowCount,
    effectiveLimit,
  });
  // Side-load only needs executions for rows that survive filtering (and the
  // display / trend window). Reusing `collectLimit` when filters are active
  // would issue ListEventExecutions for every unfiltered loaded run.
  const sideloadLimit = computeRunsSideloadLimit({
    filtersActive,
    progressive,
    displaySlice,
    collectLimit,
  });

  const { executionsByRootEventId, runExecutionsLoading } = useRunsExecutionSideload({
    canvasId,
    enabled: dataSource.kind === "runs" && needsNodeOutputs,
    pages,
    collectLimit: sideloadLimit,
    filters: runsFilters,
    resolveTrigger,
    triggerMatchOptions,
  });

  const rows = useMemo(
    () =>
      buildRunsDataSourceRows({
        dataSource,
        pages,
        ctx,
        collectLimit,
        executionsByRootEventId,
        runsFilters,
        triggerMatchOptions,
        resultLimit: progressive ? undefined : displaySlice,
      }),
    [
      dataSource,
      pages,
      ctx,
      collectLimit,
      executionsByRootEventId,
      runsFilters,
      triggerMatchOptions,
      progressive,
      displaySlice,
    ],
  );
  const triggersMatchable = triggerFilterCanMatch(runsFilters?.triggers, resolveTrigger, triggerMatchOptions);
  const fillRowCount = filtersActive ? rows.length : loadedRowCount;
  const eagerEnabled = enabled && triggersMatchable && !skipEagerFill;

  const runsFlightKey = useMemo(() => makeRunsFlightKey(canvasId, {}), [canvasId]);
  useEagerInfinitePagination({
    enabled: eagerEnabled,
    fillTarget: displaySlice,
    loadedRowCount: fillRowCount,
    pageCount,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetching: query.isFetching,
    isError: query.isError,
    isFetchNextPageError: query.isFetchNextPageError,
    fetchNextPage: query.fetchNextPage,
    flightKey: runsFlightKey,
  });

  return finalizeRunsWidgetResult({
    query,
    rows,
    eagerEnabled,
    fillRowCount,
    initialFillTarget,
    pageCount,
    filtersActive,
    triggersMatchable,
    progressive,
    runExecutionsLoading,
    displayCount,
    effectiveLimit,
    loadMore,
    displaySlice,
  });
}

function finalizeRunsWidgetResult(args: {
  query: {
    hasNextPage: boolean | undefined;
    isFetchingNextPage: boolean;
    isLoading: boolean;
    isFetching: boolean;
    isError: boolean;
    isFetchNextPageError: boolean;
    error: unknown;
    data?: { pages?: { totalCount?: number }[] };
  };
  rows: unknown[];
  eagerEnabled: boolean;
  fillRowCount: number;
  initialFillTarget: number;
  pageCount: number;
  filtersActive: boolean;
  triggersMatchable: boolean;
  progressive: boolean;
  runExecutionsLoading: boolean;
  displayCount: number;
  effectiveLimit: number;
  loadMore: () => void;
  displaySlice: number;
}): WidgetDataResult {
  const isLoading = computeRunsDataSourceLoading({
    query: args.query,
    enabled: args.eagerEnabled,
    fillRowCount: args.fillRowCount,
    initialFillTarget: args.initialFillTarget,
    pageCount: args.pageCount,
    filtersActive: args.filtersActive,
    triggersMatchable: args.triggersMatchable,
    progressive: args.progressive,
    runExecutionsLoading: args.runExecutionsLoading,
  });
  const hasMore = computeWidgetHasMore({
    progressive: args.progressive,
    displayCount: args.displayCount,
    effectiveLimit: args.effectiveLimit,
    loadedRowCount: args.fillRowCount,
    hasNextPage: args.triggersMatchable ? args.query.hasNextPage : false,
    pageCount: args.pageCount,
  });
  const isFetchingMore = args.eagerEnabled && args.query.isFetchingNextPage && !isLoading && hasMore;
  const paginationFields = args.progressive
    ? { hasMore, isFetchingMore, loadMore: args.loadMore, displayCount: args.displaySlice }
    : {};
  const totalCount = args.filtersActive ? undefined : args.query.data?.pages?.[0]?.totalCount;
  return {
    rows: args.rows,
    isLoading,
    error: args.query.error ? String(args.query.error) : undefined,
    totalCount,
    ...paginationFields,
  };
}

function countLoadedRuns(pages: { runs?: unknown[] }[], enabled: boolean): number {
  if (!enabled) return 0;
  let count = 0;
  for (const page of pages) count += page?.runs?.length ?? 0;
  return count;
}

function runsFiltersFromDataSource(dataSource: WidgetDataSource) {
  if (dataSource.kind !== "runs") return undefined;
  return { statuses: dataSource.statuses, triggers: dataSource.triggers };
}

function computeRunsCollectLimit(args: {
  filtersActive: boolean;
  progressive: boolean;
  displaySlice: number;
  loadedRowCount: number;
  effectiveLimit: number;
}): number {
  // Filters: materialize every loaded run so matches aren't dropped pre-filter.
  if (args.filtersActive) return args.loadedRowCount;
  if (args.progressive) {
    return computeTrendCollectLimit(args.displaySlice, args.loadedRowCount, args.effectiveLimit);
  }
  return args.displaySlice;
}

/**
 * Cap for execution side-load queries. When status/trigger filters are active
 * on a non-progressive panel, only the post-filter display window needs `$`
 * data — not every already-loaded (and soon-to-be-dropped) run.
 */
function computeRunsSideloadLimit(args: {
  filtersActive: boolean;
  progressive: boolean;
  displaySlice: number;
  collectLimit: number;
}): number {
  if (args.filtersActive && !args.progressive) return args.displaySlice;
  return args.collectLimit;
}

function useRunsExecutionSideload(args: {
  canvasId: string;
  enabled: boolean;
  pages: { runs?: CanvasesCanvasRun[] }[];
  collectLimit: number;
  filters?: RunStatusTriggerFilters;
  resolveTrigger?: TriggerReferenceResolver;
  triggerMatchOptions?: TriggerFilterMatchOptions;
}) {
  const runRootEventIds = useMemo(() => {
    if (!args.enabled) return [] as string[];
    return collectRunRootEventIdsFromPages(
      args.pages,
      args.collectLimit,
      args.filters,
      args.resolveTrigger,
      args.triggerMatchOptions,
    );
  }, [args.enabled, args.pages, args.collectLimit, args.filters, args.resolveTrigger, args.triggerMatchOptions]);

  const { queries: runExecutionQueries, isLoading: runExecutionsLoading } = useEventExecutionsBatch(
    args.canvasId,
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

  return { executionsByRootEventId, runExecutionsLoading };
}

function buildRunsDataSourceRows(args: {
  dataSource: WidgetDataSource;
  pages: Parameters<typeof collectRunRows>[0];
  ctx: ConsoleContextValue | undefined;
  collectLimit: number;
  executionsByRootEventId: Map<string, CanvasesCanvasNodeExecution[]>;
  runsFilters: ReturnType<typeof runsFiltersFromDataSource>;
  triggerMatchOptions?: TriggerFilterMatchOptions;
  /** Cap applied after filtering for non-progressive panels (chart/number). */
  resultLimit?: number;
}): unknown[] {
  if (args.dataSource.kind !== "runs") return [];
  const nodeNameById = buildNodeNameMap(args.ctx?.nodes);
  const collected = collectRunRows(args.pages, nodeNameById, args.collectLimit, args.executionsByRootEventId);
  const filtered = hasRunStatusTriggerFilters(args.runsFilters)
    ? collected.filter((row) =>
        runMatchesStatusTriggerFilters(
          row as CanvasesCanvasRun,
          args.runsFilters,
          (reference) => resolveConsoleTrigger(args.ctx, reference)?.node.id,
          args.triggerMatchOptions,
        ),
      )
    : collected;
  if (args.resultLimit != null && Number.isFinite(args.resultLimit) && filtered.length > args.resultLimit) {
    return filtered.slice(0, args.resultLimit);
  }
  return filtered;
}

/**
 * Collect root-event ids for execution side-load. When filters are provided,
 * only matching runs count toward `collectLimit` so tight status/trigger
 * filters do not fan out ListEventExecutions across the unfiltered page set.
 */
export function collectRunRootEventIdsFromPages(
  pages: { runs?: CanvasesCanvasRun[] }[],
  collectLimit: number,
  filters?: RunStatusTriggerFilters,
  resolveTrigger?: TriggerReferenceResolver,
  triggerMatchOptions?: TriggerFilterMatchOptions,
): string[] {
  const filterActive = hasRunStatusTriggerFilters(filters);
  const seen = new Set<string>();
  const ids: string[] = [];
  let count = 0;
  for (const page of pages) {
    for (const run of page?.runs ?? []) {
      if (filterActive && !runMatchesStatusTriggerFilters(run, filters, resolveTrigger, triggerMatchOptions)) continue;
      if (count >= collectLimit) return ids;
      count += 1;
      const id = run.rootEvent?.id;
      if (id && !seen.has(id)) {
        seen.add(id);
        ids.push(id);
      }
    }
  }
  return ids;
}
