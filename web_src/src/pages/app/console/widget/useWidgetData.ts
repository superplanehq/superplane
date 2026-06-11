import { useEffect, useMemo } from "react";

import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode } from "@/api-client";
import {
  useCanvasMemoryEntries,
  useEventExecutionsBatch,
  useInfiniteCanvasEvents,
  useInfiniteCanvasRuns,
} from "@/hooks/useCanvasData";

import { resolveConsoleNode, useConsoleContext } from "../ConsoleContext";
import { DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";
import { flattenMemoryEntries } from "./memoryRow";
import type { WidgetDataSource, WidgetRender } from "./types";

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
 */
export function useWidgetData(
  canvasId: string,
  dataSource: WidgetDataSource,
  needsNodeOutputs: boolean = true,
): WidgetDataResult {
  const ctx = useConsoleContext();

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

  useEagerExecutionPagination({
    executionsEnabled,
    executionLimit,
    eagerExecutionCount,
    eagerPageCount,
    eventsHasNextPage,
    eventsIsFetchingNextPage,
    eventsIsFetching,
    eventsFetchNextPage,
  });

  const memoryRows = useMemo(() => {
    if (dataSource.kind !== "memory") return [];
    return flattenMemoryEntries(memoryQuery.data ?? [], dataSource.namespace, dataSource.fieldPath);
  }, [dataSource, memoryQuery.data]);

  const executionRows = useMemo(() => {
    if (dataSource.kind !== "executions") return [];
    const pages = eventsData?.pages ?? [];
    const targetNode = dataSource.node ? resolveConsoleNode(ctx, dataSource.node) : undefined;
    const targetNodeId = targetNode?.node.id;
    const nodeNameById = buildNodeNameMap(ctx?.nodes);
    return collectExecutionRows(pages, targetNodeId, nodeNameById, executionLimit);
  }, [dataSource, eventsData, ctx, executionLimit]);

  // Collect unique root-event ids for the visible run page so we can lazy-
  // fetch their per-node executions (with `outputs`) via `ListEventExecutions`.
  // The runs API only returns lightweight execution refs without outputs, so
  // we have to side-load the full executions to support `$["node"].outputs`.
  const runRootEventIds = useMemo(() => {
    if (dataSource.kind !== "runs" || !needsNodeOutputs) return [] as string[];
    const seen = new Set<string>();
    const ids: string[] = [];
    let count = 0;
    for (const page of runsQuery.data?.pages ?? []) {
      for (const run of page?.runs ?? []) {
        if (count >= runsLimit) break;
        count++;
        const id = run.rootEvent?.id;
        if (id && !seen.has(id)) {
          seen.add(id);
          ids.push(id);
        }
      }
      if (count >= runsLimit) break;
    }
    return ids;
  }, [dataSource, runsQuery.data, runsLimit, needsNodeOutputs]);

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

  const runRows = useMemo(() => {
    if (dataSource.kind !== "runs") return [];
    const pages = runsQuery.data?.pages ?? [];
    const nodeNameById = buildNodeNameMap(ctx?.nodes);
    return collectRunRows(pages, nodeNameById, runsLimit, executionsByRootEventId);
  }, [dataSource, runsQuery.data, ctx, runsLimit, executionsByRootEventId]);

  // Treat ongoing eager pagination as part of the initial load so panels
  // (especially `count` aggregations) don't flash an intermediate value
  // before the rest of the pages settle.
  const executionsLoading = isExecutionQueryLoading({
    eventsIsLoading,
    executionsEnabled,
    eventsHasNextPage,
    eagerExecutionCount,
    executionLimit,
    eagerPageCount,
    eventsIsFetchingNextPage,
    eventsIsFetching,
  });

  return resultForDataSource({
    dataSource,
    memoryRows,
    memoryQuery,
    runRows,
    runsQuery,
    runExecutionsLoading,
    executionRows,
    executionsLoading,
    eventsError,
  });
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

function useEagerExecutionPagination({
  executionsEnabled,
  executionLimit,
  eagerExecutionCount,
  eagerPageCount,
  eventsHasNextPage,
  eventsIsFetchingNextPage,
  eventsIsFetching,
  eventsFetchNextPage,
}: {
  executionsEnabled: boolean;
  executionLimit: number;
  eagerExecutionCount: number;
  eagerPageCount: number;
  eventsHasNextPage: boolean | undefined;
  eventsIsFetchingNextPage: boolean;
  eventsIsFetching: boolean;
  eventsFetchNextPage: () => unknown;
}) {
  useEffect(() => {
    if (
      !shouldFetchNextExecutionPage({
        executionsEnabled,
        executionLimit,
        eagerExecutionCount,
        eagerPageCount,
        eventsHasNextPage,
        eventsIsFetchingNextPage,
        eventsIsFetching,
      })
    ) {
      return;
    }
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
}

function shouldFetchNextExecutionPage({
  executionsEnabled,
  executionLimit,
  eagerExecutionCount,
  eagerPageCount,
  eventsHasNextPage,
  eventsIsFetchingNextPage,
  eventsIsFetching,
}: {
  executionsEnabled: boolean;
  executionLimit: number;
  eagerExecutionCount: number;
  eagerPageCount: number;
  eventsHasNextPage: boolean | undefined;
  eventsIsFetchingNextPage: boolean;
  eventsIsFetching: boolean;
}): boolean {
  if (!executionsEnabled || !eventsHasNextPage) return false;
  if (eventsIsFetchingNextPage || eventsIsFetching) return false;
  if (eagerExecutionCount >= executionLimit) return false;
  return eagerPageCount < EXECUTIONS_MAX_EAGER_PAGES;
}

function isExecutionQueryLoading({
  eventsIsLoading,
  executionsEnabled,
  eventsHasNextPage,
  eagerExecutionCount,
  executionLimit,
  eagerPageCount,
  eventsIsFetchingNextPage,
  eventsIsFetching,
}: {
  eventsIsLoading: boolean;
  executionsEnabled: boolean;
  eventsHasNextPage: boolean | undefined;
  eagerExecutionCount: number;
  executionLimit: number;
  eagerPageCount: number;
  eventsIsFetchingNextPage: boolean;
  eventsIsFetching: boolean;
}): boolean {
  if (eventsIsLoading) return true;
  return (
    executionsEnabled &&
    eventsHasNextPage === true &&
    eagerExecutionCount < executionLimit &&
    eagerPageCount < EXECUTIONS_MAX_EAGER_PAGES &&
    (eventsIsFetchingNextPage || eventsIsFetching)
  );
}

function resultForDataSource({
  dataSource,
  memoryRows,
  memoryQuery,
  runRows,
  runsQuery,
  runExecutionsLoading,
  executionRows,
  executionsLoading,
  eventsError,
}: {
  dataSource: WidgetDataSource;
  memoryRows: unknown[];
  memoryQuery: { isLoading: boolean; error: unknown };
  runRows: unknown[];
  runsQuery: { isLoading: boolean; error: unknown; data?: { pages?: Array<{ totalCount?: number }> } };
  runExecutionsLoading: boolean;
  executionRows: unknown[];
  executionsLoading: boolean;
  eventsError: unknown;
}): WidgetDataResult {
  if (dataSource.kind === "memory") {
    return { rows: memoryRows, isLoading: memoryQuery.isLoading, error: errorMessage(memoryQuery.error) };
  }
  if (dataSource.kind === "runs") {
    // Treat the per-run execution side-loads as part of the initial load so
    // count/aggregate panels don't flash with empty `$` references before
    // the executions resolve.
    return {
      rows: runRows,
      isLoading: runsQuery.isLoading || runExecutionsLoading,
      error: errorMessage(runsQuery.error),
      totalCount: runsQuery.data?.pages?.[0]?.totalCount,
    };
  }
  return { rows: executionRows, isLoading: executionsLoading, error: errorMessage(eventsError) };
}

function errorMessage(error: unknown): string | undefined {
  return error ? String(error) : undefined;
}

/**
 * Walk the loaded event pages and synthesize the row objects the dashboard's
 * table / chart / number renderers consume. Each row carries the raw
 * execution fields plus four derived conveniences:
 *
 * - `status`: lowercase canonical status string (see {@link deriveExecutionStatus}).
 * - `nodeName`: friendly node label resolved per-row via `nodeNameById`,
 *   falling back to the raw `nodeId` when the canvas no longer contains
 *   that node (e.g. it was deleted after the execution ran).
 * - `durationMs`: created-to-updated elapsed time in milliseconds.
 * - `payload`: the data carried by the parent (root) event — i.e. the
 *   payload the node received. Shared by every execution under that event.
 *
 * Iteration stops as soon as `rows.length >= limit`.
 */
export function collectExecutionRows(
  pages: Array<
    | {
        events?: Array<{
          data?: Record<string, unknown>;
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
  nodeNameById: Map<string, string>,
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
          nodeName: (exec.nodeId && nodeNameById.get(exec.nodeId)) || exec.nodeId,
          durationMs:
            exec.updatedAt && exec.createdAt ? Date.parse(exec.updatedAt) - Date.parse(exec.createdAt) : undefined,
          payload: event.data,
        });
        if (rows.length >= limit) return rows;
      }
    }
  }
  return rows;
}

/**
 * Walk the loaded run pages and synthesize the row objects the dashboard's
 * widgets consume. Each row carries the raw `CanvasesCanvasRun` fields plus
 * a few derived conveniences mirroring what `RunsList` shows:
 *
 * - `status`: lowercase canonical status string (see {@link deriveRunStatus}).
 * - `nodeName`: friendly label of the node that initiated the run, resolved
 *   from `rootEvent.nodeId` via `nodeNameById`. Falls back to the raw
 *   `nodeId` when the canvas no longer contains that node.
 * - `payload`: alias for `rootEvent.data` — the initial payload that
 *   triggered the run. Exposed at the top level so authors don't have to
 *   type `rootEvent.data.*` for the common case.
 * - `durationMs`: created-to-finished elapsed time in milliseconds. Mirrors
 *   the executions row's `durationMs` so authors can write
 *   `field: durationMs, format: duration` for a friendly run-duration cell
 *   without having to write CEL date arithmetic.
 * - `$` / `DOLLAR_REWRITE_IDENTIFIER`: a map keyed by node display name
 *   pointing at each node's full execution (with `outputs` and a `data`
 *   shortcut for the latest output event). Lets authors write
 *   `$["deploy-prod"].outputs.url` in literal field paths and the same
 *   syntax in `{{ }}` CEL templates (the CEL compiler rewrites `$` to
 *   `__runNodes__` since cel-js doesn't accept `$` as an identifier).
 *
 * The raw `rootEvent`, `executions`, timestamps, etc. remain reachable via
 * dot paths (`getValueAtPath`) because we spread the full run into the row.
 *
 * Iteration stops as soon as `rows.length >= limit`.
 */
type RunRowSource = Record<string, unknown> & {
  state?: string;
  result?: string;
  createdAt?: string;
  finishedAt?: string;
  rootEvent?: {
    id?: string;
    nodeId?: string;
    data?: Record<string, unknown>;
  };
};

function buildRunRow(
  run: RunRowSource,
  nodeNameById: Map<string, string>,
  executionsByRootEventId?: Map<string, CanvasesCanvasNodeExecution[]>,
): unknown {
  const rootEvent = run.rootEvent;
  const nodeId = rootEvent?.nodeId;
  const executions = (rootEvent?.id && executionsByRootEventId?.get(rootEvent.id)) || undefined;
  const dollarNodes = buildDollarNodes(executions, nodeNameById);
  return {
    ...run,
    status: deriveRunStatus(run.state, run.result),
    nodeName: (nodeId && nodeNameById.get(nodeId)) || nodeId,
    payload: rootEvent?.data,
    durationMs: run.finishedAt && run.createdAt ? Date.parse(run.finishedAt) - Date.parse(run.createdAt) : undefined,
    $: dollarNodes,
    [DOLLAR_REWRITE_IDENTIFIER]: dollarNodes,
  };
}

export function collectRunRows(
  pages: Array<{ runs?: RunRowSource[] } | undefined>,
  nodeNameById: Map<string, string>,
  limit: number,
  executionsByRootEventId?: Map<string, CanvasesCanvasNodeExecution[]>,
): unknown[] {
  const rows: unknown[] = [];
  for (const page of pages) {
    for (const run of page?.runs ?? []) {
      rows.push(buildRunRow(run, nodeNameById, executionsByRootEventId));
      if (rows.length >= limit) return rows;
    }
  }
  return rows;
}

/**
 * Build the `$` map for a single run row. Keys are node display names so
 * authors can write `$["deploy-prod"]` in expressions; falls back to the
 * `nodeId` when the canvas no longer contains that node (e.g. it was
 * deleted). The value spreads the full execution and adds a `data` shortcut
 * mirroring the canvas-side `$['Node Name'].data` semantics.
 */
export function buildDollarNodes(
  executions: CanvasesCanvasNodeExecution[] | undefined,
  nodeNameById: Map<string, string>,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  if (!executions) return out;
  for (const exec of executions) {
    if (!exec.nodeId) continue;
    const name = nodeNameById.get(exec.nodeId) || exec.nodeId;
    out[name] = {
      ...exec,
      data: lastOutputData(exec.outputs),
    };
  }
  return out;
}

/**
 * Pick the most useful single payload from an execution's `outputs` map.
 * Mirrors how the canvas backend resolves `$['Node Name'].data`: prefer the
 * `default` channel, otherwise the first available channel; take the last
 * event in that channel (most recent emission). When the event itself is
 * an envelope-shaped object with a `.data` field, unwrap it; otherwise
 * return the event verbatim. Returns `undefined` for missing or empty
 * outputs so widget cells render `-`.
 */
export function lastOutputData(outputs: Record<string, unknown> | undefined): unknown {
  if (!outputs) return undefined;
  const channels = Object.keys(outputs);
  if (channels.length === 0) return undefined;
  const channel = channels.includes("default") ? "default" : channels[0];
  const events = outputs[channel];
  if (!Array.isArray(events) || events.length === 0) return undefined;
  const last = events[events.length - 1];
  if (last && typeof last === "object" && !Array.isArray(last) && "data" in last) {
    return (last as { data: unknown }).data;
  }
  return last;
}

/**
 * Build a `nodeId -> friendly name` lookup from the canvas nodes available
 * on the dashboard context. We index by id only (not name) because event
 * executions always carry `nodeId`. Falls back to the node id when the
 * canvas node has no `name`, so the widget never shows a blank label.
 */
export function buildNodeNameMap(nodes: SuperplaneComponentsNode[] | undefined): Map<string, string> {
  const map = new Map<string, string>();
  if (!nodes) return map;
  for (const node of nodes) {
    if (!node.id) continue;
    map.set(node.id, node.name || node.id);
  }
  return map;
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

/**
 * Collapse the run `state` / `result` enum pair into the lowercase status
 * vocabulary used across the dashboard and RunsList. Mirrors `getRunStatus`
 * in `ui/Runs/runPresentation.ts`. Runs have a smaller state machine than
 * executions — no separate `pending` step — so a started run that has not
 * yet produced a result maps to `running`.
 */
function deriveRunStatus(
  state: string | undefined,
  result: string | undefined,
): "passed" | "failed" | "cancelled" | "running" | "unknown" {
  if (state === "STATE_STARTED") return "running";
  if (result === "RESULT_FAILED") return "failed";
  if (result === "RESULT_CANCELLED") return "cancelled";
  if (result === "RESULT_PASSED" || state === "STATE_FINISHED") return "passed";
  return "unknown";
}
