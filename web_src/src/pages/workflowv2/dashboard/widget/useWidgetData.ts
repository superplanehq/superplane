import { useEffect, useMemo } from "react";

import { useCanvasMemoryEntries, useInfiniteCanvasEvents, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";

import { resolveDashboardNode, useDashboardContext } from "../DashboardContext";
import { getValueAtPath } from "./fieldPath";
import type { WidgetDataSource } from "./types";

export interface WidgetDataResult {
  rows: unknown[];
  isLoading: boolean;
  error?: string;
  /** Server-reported total for sources that expose one (currently `runs`). */
  totalCount?: number;
}

/**
 * Upper bound on how many event pages a widget will eagerly pull through
 * the infinite query before giving up. Each page returns 25 events, so 20
 * pages covers up to ~500 events — comfortably more than the row `limit`
 * any dashboard panel currently asks for, while bounding memory and
 * network cost for canvases with very long histories.
 */
const EXECUTIONS_MAX_EAGER_PAGES = 20;

/**
 * Reactively fetch the dataset for a widget based on its declared data source.
 * Returns a uniform `{ rows, isLoading, error }` so renderers don't have to
 * deal with the underlying query specifics.
 *
 * - `memory` reads from the canvas memory entries and filters to the requested
 *   namespace. When `fieldPath` is set, rows are flat-mapped to the value at
 *   that path (lists are spread, scalars become single-row entries).
 * - `runs` reads from the canvas runs query and exposes the API totalCount,
 *   which makes Number panels report the canvas run count without depending
 *   on how many pages have been loaded.
 * - `executions` reads from the infinite canvas events query and flattens
 *   their `executions[]` arrays. When a `node` reference is set, only matching
 *   executions are returned. Pages are fetched eagerly until we have enough
 *   rows to satisfy `limit` or the canvas has no more events — without this,
 *   the visible row count fluctuates non-monotonically when a manual run
 *   prepends a new (still-empty) event that pushes an older event-with-
 *   executions off the loaded first page.
 */
export function useWidgetData(canvasId: string, dataSource: WidgetDataSource): WidgetDataResult {
  const ctx = useDashboardContext();

  const memoryEnabled = dataSource.kind === "memory";
  const memoryQuery = useCanvasMemoryEntries(canvasId, memoryEnabled);

  const executionsEnabled = dataSource.kind === "executions";
  const eventsQuery = useInfiniteCanvasEvents(canvasId, executionsEnabled);
  const runsEnabled = dataSource.kind === "runs";
  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, runsEnabled);

  const executionLimit = dataSource.kind === "executions" ? (dataSource.limit ?? 50) : 0;
  const runsLimit = dataSource.kind === "runs" ? (dataSource.limit ?? 50) : 0;

  /**
   * Trigger additional infinite-query pages while we don't have enough
   * executions to satisfy the widget's `limit` and the canvas still has
   * more events to give us. We re-evaluate after each page arrives, which
   * naturally backs off once `executionRows` reaches the limit or
   * `hasNextPage` flips to false. Capped at `EXECUTIONS_MAX_EAGER_PAGES`
   * to bound work for very long canvas histories.
   */
  const {
    data: eventsData,
    isLoading: eventsIsLoading,
    isFetching: eventsIsFetching,
    isFetchingNextPage: eventsIsFetchingNextPage,
    hasNextPage: eventsHasNextPage,
    fetchNextPage: eventsFetchNextPage,
    error: eventsError,
  } = eventsQuery;

  const eagerPageCount = eventsData?.pages?.length ?? 0;
  const eagerExecutionCount = useMemo(() => {
    if (!executionsEnabled) return 0;
    const pages = eventsData?.pages ?? [];
    let count = 0;
    for (const page of pages) {
      for (const event of page?.events ?? []) {
        count += event.executions?.length ?? 0;
      }
    }
    return count;
  }, [executionsEnabled, eventsData]);

  useEffect(() => {
    if (!executionsEnabled) return;
    if (!eventsHasNextPage) return;
    if (eventsIsFetchingNextPage || eventsIsFetching) return;
    if (eagerExecutionCount >= executionLimit) return;
    if (eagerPageCount >= EXECUTIONS_MAX_EAGER_PAGES) return;
    void eventsFetchNextPage();
  }, [
    executionsEnabled,
    executionLimit,
    eagerExecutionCount,
    eagerPageCount,
    eventsHasNextPage,
    eventsIsFetchingNextPage,
    eventsIsFetching,
    eventsFetchNextPage,
  ]);

  const memoryRows = useMemo(() => {
    if (dataSource.kind !== "memory") return [];
    const entries = memoryQuery.data ?? [];
    const filtered = entries.filter((entry) => entry.namespace === dataSource.namespace);
    if (!dataSource.fieldPath) return filtered.map((entry) => entry.values ?? entry);
    const out: unknown[] = [];
    for (const entry of filtered) {
      const value = getValueAtPath(entry.values, dataSource.fieldPath);
      if (Array.isArray(value)) out.push(...value);
      else if (value !== undefined) out.push(value);
    }
    return out;
  }, [dataSource, memoryQuery.data]);

  const executionRows = useMemo(() => {
    if (dataSource.kind !== "executions") return [];
    const pages = eventsData?.pages ?? [];
    const targetNode = dataSource.node ? resolveDashboardNode(ctx, dataSource.node) : undefined;
    const targetNodeId = targetNode?.node.id;
    const targetLabel = targetNode?.label;
    return collectExecutionRows(pages, targetNodeId, targetLabel, executionLimit);
  }, [dataSource, eventsData, ctx, executionLimit]);

  const runRows = useMemo(() => {
    if (dataSource.kind !== "runs") return [];
    const rows: unknown[] = [];
    for (const page of runsQuery.data?.pages ?? []) {
      for (const run of page?.runs ?? []) {
        rows.push(run);
        if (rows.length >= runsLimit) return rows;
      }
    }
    return rows;
  }, [dataSource, runsQuery.data, runsLimit]);

  if (dataSource.kind === "memory") {
    return {
      rows: memoryRows,
      isLoading: memoryQuery.isLoading,
      error: memoryQuery.error ? String(memoryQuery.error) : undefined,
    };
  }
  if (dataSource.kind === "runs") {
    return {
      rows: runRows,
      isLoading: runsQuery.isLoading,
      error: runsQuery.error ? String(runsQuery.error) : undefined,
      totalCount: runsQuery.data?.pages?.[0]?.totalCount,
    };
  }
  // Treat ongoing eager pagination as part of the initial load so panels
  // (especially `count` aggregations) don't flash an intermediate value
  // before the rest of the pages settle.
  const executionsLoading =
    eventsIsLoading ||
    (executionsEnabled &&
      eventsHasNextPage === true &&
      eagerExecutionCount < executionLimit &&
      eagerPageCount < EXECUTIONS_MAX_EAGER_PAGES &&
      (eventsIsFetchingNextPage || eventsIsFetching));
  return {
    rows: executionRows,
    isLoading: executionsLoading,
    error: eventsError ? String(eventsError) : undefined,
  };
}

/**
 * Walk the loaded event pages and synthesize the row objects the dashboard's
 * table / chart / number renderers consume. Each row carries the raw
 * execution fields plus three derived conveniences:
 *
 * - `status`: lowercase canonical status string (see {@link deriveExecutionStatus}).
 * - `nodeName`: friendly node label when a target node is resolved.
 * - `durationMs`: created-to-updated elapsed time in milliseconds.
 *
 * Iteration stops as soon as `rows.length >= limit`.
 */
function collectExecutionRows(
  pages: Array<
    | {
        events?: Array<{
          executions?: Array<
            Record<string, unknown> & {
              nodeId?: string;
              state?: string;
              result?: string;
              createdAt?: string;
              updatedAt?: string;
            }
          >;
        }>;
      }
    | undefined
  >,
  targetNodeId: string | undefined,
  targetLabel: string | undefined,
  limit: number,
): unknown[] {
  const rows: unknown[] = [];
  for (const page of pages) {
    for (const event of page?.events ?? []) {
      for (const exec of event.executions ?? []) {
        if (targetNodeId && exec.nodeId !== targetNodeId) continue;
        rows.push({
          ...exec,
          status: deriveExecutionStatus(exec.state, exec.result),
          nodeName: targetLabel ?? exec.nodeId,
          durationMs:
            exec.updatedAt && exec.createdAt ? Date.parse(exec.updatedAt) - Date.parse(exec.createdAt) : undefined,
        });
        if (rows.length >= limit) return rows;
      }
    }
  }
  return rows;
}

/**
 * Collapse the API `state` / `result` enum pair into the lowercase status
 * vocabulary the rest of the dashboard speaks: `passed`, `failed`,
 * `cancelled`, `running`, `pending`, `unknown`. Matches the lookup tables in
 * `WidgetTable` (`STATUS_PILL_CLASS`) and `NodePanelCard` (`STATUS_CLASS`).
 */
function deriveExecutionStatus(
  state: string | undefined,
  result: string | undefined,
): "passed" | "failed" | "cancelled" | "running" | "pending" | "unknown" {
  if (state === "STATE_PENDING") return "pending";
  if (state === "STATE_STARTED") return "running";
  if (state === "STATE_FINISHED") {
    switch (result) {
      case "RESULT_PASSED":
        return "passed";
      case "RESULT_FAILED":
        return "failed";
      case "RESULT_CANCELLED":
        return "cancelled";
      default:
        return "unknown";
    }
  }
  return "unknown";
}
