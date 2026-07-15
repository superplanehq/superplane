import { useEffect, useMemo } from "react";

import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import {
  useCanvasMemoryEntries,
  useEventExecutionsBatch,
  useInfiniteCanvasRuns,
  type CanvasMemoryEntry,
} from "@/hooks/useCanvasData";
import {
  hasRunStatusTriggerFilters,
  runMatchesStatusTriggerFilters,
  runSelectStatusFilterCanMatch,
  triggerFilterCanMatch,
  type TriggerFilterMatchOptions,
} from "@/ui/Runs/runStatusTriggerFilter";

import { resolveConsoleTrigger, useConsoleContext } from "./ConsoleContext";
import { markdownTemplateReferencesRunNode } from "./markdownInterpolation";
import { DOLLAR_REWRITE_IDENTIFIER } from "./widget/celExpr";
import { memoryEntryToRow } from "./widget/memoryRow";
import { makeRunsFlightKey } from "./widget/runsWidgetQuery";
import { buildDollarNodes, buildNodeNameMap } from "./widget/useWidgetData";
import { claimEagerPaginationFetch, WIDGET_MAX_EAGER_PAGES } from "./widget/widgetPagination";
import { applySort } from "./widget/widgetData";
import type { MarkdownMemoryVariableSource, MarkdownRunVariableSource, MarkdownVariable } from "./panelTypes";

/**
 * Shape we read off a `useInfiniteCanvasRuns` result. Defined locally so the
 * memoized run-id collector doesn't reach for `typeof latestQuery.data`,
 * which the eslint react-hooks plugin tracks as a dependency on the entire
 * `latestQuery` object. Every field the run resolver consumes is optional
 * so we can share the same shape between the sideload id collector and the
 * status/trigger filter — both operate on the same runtime data.
 */
interface RunsPage {
  runs?: RunRow[];
}
interface RunsQueryData {
  pages?: RunsPage[];
}

export interface MarkdownVariableError {
  /** The variable name that failed to resolve, or `null` for global errors. */
  name: string | null;
  message: string;
}

export interface MarkdownVariablesResult {
  /** Map of variable name -> resolved object (or `null` when not found). */
  vars: Record<string, unknown>;
  /**
   * `true` while base memory/run queries or the execution side-load are still
   * loading. Does **not** include per-variable filter search — use
   * {@link searchingNames} for that so previews can gate per name.
   */
  isLoading: boolean;
  /**
   * `true` while the base memory / run queries are still loading. Excludes the
   * per-run execution side-load so callers can gate each piece of templated
   * text precisely (see {@link markdownTextIsLoading}).
   */
  baseLoading: boolean;
  /** `true` while the per-run execution side-load (for `$["Node"]` refs) loads. */
  sideloadLoading: boolean;
  /**
   * Names of run variables that are still eagerly paging for a
   * status/trigger filter match. Callers pass this to
   * {@link markdownTextIsLoading} so only templates that reference those
   * names stay gated — siblings can render in the meantime.
   */
  searchingNames: string[];
  /** Per-variable resolution errors and any global error (e.g. missing canvas). */
  errors: MarkdownVariableError[];
}

/**
 * Resolve a markdown panel's declared variables into a `{ name: value }` map
 * suitable for use as CEL globals in `{{ name.field }}` interpolation.
 *
 * Two source kinds are supported, each picking a single object:
 *
 *  - `memory` — reads from {@link useCanvasMemoryEntries}, filters by
 *    namespace, applies optional property-equality `matches`, sorts via the
 *    shared `applySort` helper, and takes the first row. The returned value
 *    spreads the entry's stored fields plus the metadata exposed on memory
 *    rows (`id`, `namespace`, `createdAt`, `updatedAt`). The default order
 *    when `orderBy` is omitted is `createdAt desc` so "newest record" is
 *    one click away.
 *
 *  - `run` — reads from {@link useInfiniteCanvasRuns} with the appropriate
 *    `results` filter (or none for `latest`) and takes the first run from
 *    the first page. Run values are enriched with the same `$` map the
 *    table widget builds, so authors can write `{{ run.$["Deploy"].data.x }}`.
 *    The per-node execution side-load only happens when the interpolated text
 *    actually references `$[` — pass the panel text (title + body, since both
 *    are interpolated) via `textForRunSideload` so the hook can skip the extra
 *    request when no template needs it.
 *
 * The hook is intentionally hook-stable: it issues a fixed set of queries
 * (one memory query and up to three run queries, deduped by select) so it
 * never violates the rules of hooks when authors edit the variables list.
 */
export function useMarkdownVariables(
  canvasId: string,
  variables: MarkdownVariable[] | undefined,
  textForRunSideload: string | undefined,
): MarkdownVariablesResult {
  const ctx = useConsoleContext();
  const list = useMemo(() => variables ?? [], [variables]);
  const runSelects = useRunSelectFlags(list);

  // Fixed-shape query calls so we never break the rules of hooks regardless
  // of what variables list the panel was edited into.
  const memoryQuery = useCanvasMemoryEntries(canvasId, runSelects.hasMemoryVar);
  const latestQuery = useInfiniteCanvasRuns(canvasId, {}, runSelects.wantLatest);
  const passedQuery = useInfiniteCanvasRuns(canvasId, { results: ["RESULT_PASSED"] }, runSelects.wantLatestPassed);
  const failedQuery = useInfiniteCanvasRuns(canvasId, { results: ["RESULT_FAILED"] }, runSelects.wantLatestFailed);

  const needsRunSideload = useMemo(() => markdownTemplateReferencesRunNode(textForRunSideload), [textForRunSideload]);

  const latestRunsData = latestQuery.data as RunsQueryData | undefined;
  const passedRunsData = passedQuery.data as RunsQueryData | undefined;
  const failedRunsData = failedQuery.data as RunsQueryData | undefined;

  const runRootEventIds = useMemo(() => {
    if (!needsRunSideload) return [] as string[];
    return collectRunVariableRootEventIds({
      list,
      ctx,
      latestRunsData,
      passedRunsData,
      failedRunsData,
    });
  }, [needsRunSideload, list, ctx, latestRunsData, passedRunsData, failedRunsData]);

  // When any variable in a `select` bucket carries a status/trigger filter,
  // the first fetched page may not contain a matching run. Eagerly page
  // through the shared infinite query until a match surfaces, the source
  // runs out of pages, or we hit `WIDGET_MAX_EAGER_PAGES`.
  useEagerFilterPagination({
    query: latestQuery,
    list,
    select: "latest",
    data: latestRunsData,
    ctx,
    flightKey: makeRunsFlightKey(canvasId, {}),
  });
  useEagerFilterPagination({
    query: passedQuery,
    list,
    select: "latest_passed",
    data: passedRunsData,
    ctx,
    flightKey: makeRunsFlightKey(canvasId, { results: ["RESULT_PASSED"] }),
  });
  useEagerFilterPagination({
    query: failedQuery,
    list,
    select: "latest_failed",
    data: failedRunsData,
    ctx,
    flightKey: makeRunsFlightKey(canvasId, { results: ["RESULT_FAILED"] }),
  });

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

  const memoryEntries = useMemo(() => memoryQuery.data ?? [], [memoryQuery.data]);
  const memoryLoading = runSelects.hasMemoryVar && memoryQuery.isLoading;
  const nodeNameById = useMemo(() => buildNodeNameMap(ctx?.nodes), [ctx?.nodes]);

  const { vars, errors, searchingNames } = useMemo(() => {
    const out: Record<string, unknown> = {};
    const errs: MarkdownVariableError[] = [];
    const searching: string[] = [];
    // Mirror `normalizeDraftVariables` (first-wins) so the preview resolves the
    // same row that save persists.
    const seen = new Set<string>();
    const resolveTrigger = (reference: string) => resolveConsoleTrigger(ctx, reference)?.node.id;
    const triggerMatchOptions = {
      nodeCatalogLoading: ctx?.nodesLoading ?? false,
    };
    for (const variable of list) {
      if (!variable?.name || !variable?.source) continue;
      if (seen.has(variable.name)) continue;
      seen.add(variable.name);
      const resolved = resolveVariable(variable, {
        memoryEntries,
        memoryLoading,
        latestQuery,
        passedQuery,
        failedQuery,
        executionsByRootEventId,
        nodeNameById,
        resolveTrigger,
        triggerMatchOptions,
      });
      out[variable.name] = resolved.value;
      if (resolved.error) errs.push({ name: variable.name, message: resolved.error });
      if (resolved.searching) searching.push(variable.name);
    }
    return { vars: out, errors: errs, searchingNames: searching };
  }, [
    list,
    memoryEntries,
    memoryLoading,
    latestQuery,
    passedQuery,
    failedQuery,
    executionsByRootEventId,
    nodeNameById,
    ctx,
  ]);

  const baseLoading =
    memoryLoading ||
    (runSelects.wantLatest && latestQuery.isLoading) ||
    (runSelects.wantLatestPassed && passedQuery.isLoading) ||
    (runSelects.wantLatestFailed && failedQuery.isLoading);
  const sideloadLoading = needsRunSideload && runExecutionsLoading;
  // Searching is tracked separately via `searchingNames` so variable previews
  // and panel text can gate per-name. Do not fold it into `isLoading` — that
  // made every VariablePreview wait while any filtered run var was paging.
  const isLoading = baseLoading || sideloadLoading;

  return { vars, isLoading, baseLoading, sideloadLoading, searchingNames, errors };
}

function useRunSelectFlags(list: MarkdownVariable[]) {
  return useMemo(
    () => ({
      hasMemoryVar: list.some((v) => v.source?.kind === "memory"),
      wantLatest: list.some((v) => v.source?.kind === "run" && v.source.select === "latest"),
      wantLatestPassed: list.some((v) => v.source?.kind === "run" && v.source.select === "latest_passed"),
      wantLatestFailed: list.some((v) => v.source?.kind === "run" && v.source.select === "latest_failed"),
    }),
    [list],
  );
}

function collectRunVariableRootEventIds(args: {
  list: MarkdownVariable[];
  ctx: ReturnType<typeof useConsoleContext>;
  latestRunsData: RunsQueryData | undefined;
  passedRunsData: RunsQueryData | undefined;
  failedRunsData: RunsQueryData | undefined;
}): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  const resolveTrigger = (reference: string) => resolveConsoleTrigger(args.ctx, reference)?.node.id;
  const triggerMatchOptions = {
    nodeCatalogLoading: args.ctx?.nodesLoading ?? false,
  };
  for (const variable of args.list) {
    const source = variable?.source;
    if (source?.kind !== "run") continue;
    const data = pickRunQueryData({
      source,
      latestRunsData: args.latestRunsData,
      passedRunsData: args.passedRunsData,
      failedRunsData: args.failedRunsData,
    });
    const run = firstMatchingRun(data, source, resolveTrigger, triggerMatchOptions);
    const id = run?.rootEvent?.id;
    if (id && !seen.has(id)) {
      seen.add(id);
      ids.push(id);
    }
  }
  return ids;
}

/**
 * Pick the right runs query data based on the variable's `select` value.
 * Extracted so the sideload id collector and the resolver share the same
 * bucket routing without duplicating the switch.
 */
function pickRunQueryData(args: {
  source: MarkdownRunVariableSource;
  latestRunsData: RunsQueryData | undefined;
  passedRunsData: RunsQueryData | undefined;
  failedRunsData: RunsQueryData | undefined;
}): RunsQueryData | undefined {
  if (args.source.select === "latest_passed") return args.passedRunsData;
  if (args.source.select === "latest_failed") return args.failedRunsData;
  return args.latestRunsData;
}

interface InfiniteRunsQuery {
  hasNextPage?: boolean;
  isFetching: boolean;
  isFetchingNextPage: boolean;
  isError: boolean;
  isFetchNextPageError: boolean;
  fetchNextPage: () => Promise<unknown> | unknown;
}

/**
 * Fetch additional pages of the shared `select` bucket until every
 * filter-carrying variable in that bucket resolves to a match — or the
 * source runs out of pages / we hit the widget page cap. No-op when no
 * variable in the bucket has filters set so unfiltered variables keep
 * their fast single-page behavior.
 *
 * Joins the same `claimEagerPaginationFetch` single-flight lock widgets
 * use so markdown/html panels and runs widgets sharing a cache entry
 * cannot race duplicate `before=` fetches in one commit.
 */
function useEagerFilterPagination(args: {
  query: InfiniteRunsQuery;
  list: MarkdownVariable[];
  select: MarkdownRunVariableSource["select"];
  data: RunsQueryData | undefined;
  ctx: ReturnType<typeof useConsoleContext>;
  flightKey: string;
}) {
  const { query, list, select, data, ctx, flightKey } = args;
  const filteredVariables = useMemo(
    () =>
      list.filter(
        (variable): variable is MarkdownVariable & { source: MarkdownRunVariableSource } =>
          variable?.source?.kind === "run" &&
          variable.source.select === select &&
          hasRunStatusTriggerFilters({
            statuses: variable.source.statuses,
            triggers: variable.source.triggers,
          }),
      ),
    [list, select],
  );

  const pageCount = data?.pages?.length ?? 0;
  const anyUnresolved = useMemo(() => {
    if (filteredVariables.length === 0) return false;
    const resolveTrigger = (reference: string) => resolveConsoleTrigger(ctx, reference)?.node.id;
    const triggerMatchOptions = {
      nodeCatalogLoading: ctx?.nodesLoading ?? false,
    };
    return filteredVariables.some((variable) => {
      if (!runSelectStatusFilterCanMatch(variable.source.select, variable.source.statuses)) return false;
      if (!triggerFilterCanMatch(variable.source.triggers, resolveTrigger, triggerMatchOptions)) return false;
      return firstMatchingRun(data, variable.source, resolveTrigger, triggerMatchOptions) === undefined;
    });
  }, [filteredVariables, data, ctx]);

  // Break the query object into primitives so the effect only re-runs when
  // pagination-relevant fields actually change — the full query object is
  // a fresh identity on every render.
  const { hasNextPage, isFetching, isFetchingNextPage, isError, isFetchNextPageError, fetchNextPage } = query;
  useEffect(() => {
    if (!anyUnresolved) return;
    if (!hasNextPage) return;
    // A failed page-1 or fetchNextPage leaves hasNextPage true without
    // advancing pageCount — stop rather than retrying forever / sticking
    // the variable in "searching".
    if (isError || isFetchNextPageError) return;
    if (isFetching || isFetchingNextPage) return;
    if (pageCount >= WIDGET_MAX_EAGER_PAGES) return;
    claimEagerPaginationFetch(flightKey, fetchNextPage);
  }, [
    anyUnresolved,
    pageCount,
    hasNextPage,
    isError,
    isFetchNextPageError,
    isFetching,
    isFetchingNextPage,
    fetchNextPage,
    flightKey,
  ]);
}

/**
 * Scan every already-loaded page and return the first run that matches
 * the variable's optional status + trigger filters. Falls back to
 * `pages[0].runs[0]` when no filters are configured so behavior is
 * identical to the pre-filter world for existing variables.
 */
function firstMatchingRun(
  data: RunsQueryData | undefined,
  source: MarkdownRunVariableSource,
  resolveTrigger: (reference: string) => string | undefined,
  triggerMatchOptions?: TriggerFilterMatchOptions,
): RunRow | undefined {
  const pages = data?.pages ?? [];
  const filters = { statuses: source.statuses, triggers: source.triggers };
  const active = hasRunStatusTriggerFilters(filters);
  for (const page of pages) {
    for (const run of page?.runs ?? []) {
      if (!active) return run;
      if (runMatchesStatusTriggerFilters(run as CanvasesCanvasRun, filters, resolveTrigger, triggerMatchOptions)) {
        return run;
      }
    }
  }
  return undefined;
}

interface ResolveContext {
  memoryEntries: CanvasMemoryEntry[];
  memoryLoading: boolean;
  latestQuery: RunQuerySnapshot;
  passedQuery: RunQuerySnapshot;
  failedQuery: RunQuerySnapshot;
  executionsByRootEventId: Map<string, CanvasesCanvasNodeExecution[]>;
  nodeNameById: Map<string, string>;
  resolveTrigger: (reference: string) => string | undefined;
  triggerMatchOptions?: TriggerFilterMatchOptions;
}

interface RunQuerySnapshot {
  isLoading: boolean;
  isFetchingNextPage: boolean;
  isError: boolean;
  isFetchNextPageError: boolean;
  hasNextPage?: boolean;
  data?: RunsQueryData;
  error?: Error | null;
}

/** Resolve a single variable to a `{ value, error?, searching? }` pair. */
function resolveVariable(
  variable: MarkdownVariable,
  ctx: ResolveContext,
): { value: unknown; error?: string; searching?: boolean } {
  if (variable.source.kind === "memory") {
    return resolveMemoryVariable(ctx.memoryEntries, variable.source, ctx.memoryLoading);
  }
  return resolveRunVariable(variable.source, ctx);
}

interface RunRow extends Record<string, unknown> {
  state?: string;
  result?: string;
  createdAt?: string;
  finishedAt?: string;
  rootEvent?: { id?: string; nodeId?: string; data?: Record<string, unknown> };
}

function pickRunQuery(select: MarkdownRunVariableSource["select"], ctx: ResolveContext): RunQuerySnapshot {
  if (select === "latest_passed") return ctx.passedQuery;
  if (select === "latest_failed") return ctx.failedQuery;
  return ctx.latestQuery;
}

function resolveRunVariable(
  source: MarkdownRunVariableSource,
  ctx: ResolveContext,
): { value: unknown; error?: string; searching?: boolean } {
  const query = pickRunQuery(source.select, ctx);
  const firstRun = firstMatchingRun(query.data, source, ctx.resolveTrigger, ctx.triggerMatchOptions);
  if (firstRun) {
    return { value: buildRunVariableValue(firstRun, ctx) };
  }
  // Prefer a concrete load failure over "not found" / endless searching —
  // a failed fetchNextPage leaves hasNextPage true without advancing pages.
  if (isRunsQueryFailed(query)) {
    return { value: null, error: runsLoadErrorMessage(query) };
  }
  // Stay quiet while the initial page or eager filter pagination is still
  // searching — otherwise the preview flashes "No run matched…" between pages.
  if (isRunQueryStillSearching(query, source, ctx.resolveTrigger, ctx.triggerMatchOptions)) {
    return { value: null, searching: true };
  }
  return { value: null, error: noRunFoundMessage(source) };
}

function isRunsQueryFailed(query: RunQuerySnapshot): boolean {
  return query.isError || query.isFetchNextPageError;
}

function runsLoadErrorMessage(query: RunQuerySnapshot): string {
  const message = query.error?.message?.trim();
  if (message) return `Failed to load runs: ${message}`;
  return "Failed to load runs.";
}

/**
 * True while a run variable should defer its "not found" error: either the
 * query hasn't settled, the next filtered page is in flight, or eager
 * pagination still has pages left to try before giving up.
 *
 * Returns `false` once a runs query has failed so callers can surface the
 * load error instead of sticking the panel in a perpetual loading state.
 */
export function isRunQueryStillSearching(
  query: RunQuerySnapshot,
  source: MarkdownRunVariableSource,
  resolveTrigger?: (reference: string) => string | undefined,
  triggerMatchOptions?: TriggerFilterMatchOptions,
): boolean {
  if (isRunsQueryFailed(query)) return false;
  if (query.isLoading || query.isFetchingNextPage) return true;
  if (!hasRunStatusTriggerFilters({ statuses: source.statuses, triggers: source.triggers })) return false;
  if (!runSelectStatusFilterCanMatch(source.select, source.statuses)) return false;
  if (!triggerFilterCanMatch(source.triggers, resolveTrigger, triggerMatchOptions)) return false;
  if (isTriggerCatalogPending(source.triggers, resolveTrigger, triggerMatchOptions)) return true;
  const pageCount = query.data?.pages?.length ?? 0;
  return query.hasNextPage === true && pageCount < WIDGET_MAX_EAGER_PAGES;
}

function isTriggerCatalogPending(
  triggers: readonly string[] | undefined,
  resolveTrigger: ((reference: string) => string | undefined) | undefined,
  options: TriggerFilterMatchOptions | undefined,
): boolean {
  return (triggers?.length ?? 0) > 0 && resolveTrigger !== undefined && options?.nodeCatalogLoading === true;
}

function buildRunVariableValue(firstRun: RunRow, ctx: ResolveContext): Record<string, unknown> {
  const rootEvent = firstRun.rootEvent;
  const nodeId = rootEvent?.nodeId;
  const executions = (rootEvent?.id && ctx.executionsByRootEventId.get(rootEvent.id)) || undefined;
  const dollarNodes = buildDollarNodes(executions, ctx.nodeNameById);
  return {
    ...firstRun,
    status: deriveRunStatus(firstRun.state, firstRun.result),
    nodeName: (nodeId && ctx.nodeNameById.get(nodeId)) || nodeId,
    payload: rootEvent?.data,
    durationMs:
      firstRun.finishedAt && firstRun.createdAt
        ? Date.parse(firstRun.finishedAt) - Date.parse(firstRun.createdAt)
        : undefined,
    $: dollarNodes,
    [DOLLAR_REWRITE_IDENTIFIER]: dollarNodes,
  };
}

function noRunFoundMessage(source: MarkdownRunVariableSource): string {
  const filtered = hasRunStatusTriggerFilters({ statuses: source.statuses, triggers: source.triggers });
  if (filtered) return "No run matched the configured filters yet.";
  switch (source.select) {
    case "latest_passed":
      return "No successful run found yet.";
    case "latest_failed":
      return "No failed run found yet.";
    default:
      return "No runs found yet.";
  }
}

/**
 * Resolve a single memory variable: filter by namespace + matches, sort, and
 * return the first row. Falls back to `createdAt desc` when no `orderBy` is
 * given, matching how the backend stores memory rows (newest first). Returns
 * `value: null` and a human-readable error string when no row matches so the
 * editor preview can surface a clear empty state without crashing the
 * markdown render.
 *
 * While the canvas memory query is still in flight (`loading`) we suppress the
 * "no rows" error and return `value: null` so the editor preview doesn't flash
 * a false empty state before the fetch settles — mirroring how run variables
 * wait on their query's `isLoading`.
 */
export function resolveMemoryVariable(
  entries: CanvasMemoryEntry[],
  source: MarkdownMemoryVariableSource,
  loading: boolean,
): { value: unknown; error?: string } {
  const namespace = source.namespace?.trim();
  if (!namespace) {
    return { value: null, error: "Missing namespace." };
  }
  const isList = source.mode === "list";
  const filtered = entries.filter((entry) => entry.namespace === namespace);
  if (filtered.length === 0) {
    // While the backing query is in flight, resolve to `null` for both modes
    // (not `[]` for list) so the editor preview shows its shared "Loading
    // preview…" state — gated on `value == null` — instead of flashing an
    // empty list ("List · 0 items"). The settled empty state (`[]` for list,
    // an error for single) only applies once loading completes.
    if (loading) return { value: null };
    if (isList) return { value: [] };
    return { value: null, error: `No memory rows in namespace ${JSON.stringify(namespace)}.` };
  }
  const rows = filtered.map(memoryEntryToRow);
  const matched = applyMatches(rows, source.matches);
  if (matched.length === 0) {
    if (isList) return { value: [] };
    return { value: null, error: "No memory row matched the filters." };
  }
  const sortField = source.orderBy?.trim() || "createdAt";
  const sortOrder = source.direction ?? "desc";
  const sorted = applySort(matched, { field: sortField, order: sortOrder });
  return { value: pickMemoryRows(sorted, source) };
}

/**
 * Reduce the sorted, match-filtered row set down to whatever the variable
 * should expose to CEL. Kept as a tiny helper so `resolveMemoryVariable`
 * stays under the ESLint complexity budget and so list-mode tests can drive
 * the picker directly.
 */
export function pickMemoryRows(sorted: Record<string, unknown>[], source: MarkdownMemoryVariableSource): unknown {
  if (source.mode !== "list") return sorted[0];
  // Only a positive integer caps the list. A fractional limit would otherwise
  // be floored by `Array.prototype.slice` (e.g. 1.5 -> 1 row), so fail soft to
  // "no cap" and let the validator surface the real error on save.
  if (typeof source.limit === "number" && Number.isInteger(source.limit) && source.limit > 0) {
    return sorted.slice(0, source.limit);
  }
  return sorted;
}

/**
 * Apply property-equality matches to memory rows. Each match is a
 * `field === value` comparison; values are compared as strings so authors
 * don't have to know whether the stored value is numeric or string-typed.
 * Matches with an empty `field` are ignored to mirror the lenient behavior
 * the form editor leans on while users are typing.
 */
function applyMatches(rows: Record<string, unknown>[], matches: { field: string; value: string }[] | undefined) {
  if (!matches || matches.length === 0) return rows;
  return rows.filter((row) =>
    matches.every((match) => {
      const field = match?.field?.trim();
      if (!field) return true;
      const expected = match?.value ?? "";
      const actual = row[field];
      if (actual === undefined || actual === null) return expected === "";
      return String(actual) === String(expected);
    }),
  );
}

/**
 * Mirror of `deriveRunStatus` in `useWidgetData.ts`. Inlined here to keep
 * variable values consistent with table-row shapes without exporting an
 * additional symbol.
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
