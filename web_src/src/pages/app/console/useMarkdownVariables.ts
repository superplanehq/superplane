import { useMemo } from "react";

import type { CanvasesCanvasNodeExecution } from "@/api-client";
import {
  useCanvasMemoryEntries,
  useEventExecutionsBatch,
  useInfiniteCanvasRuns,
  type CanvasMemoryEntry,
} from "@/hooks/useCanvasData";

import { useConsoleContext } from "./ConsoleContext";
import { markdownTemplateReferencesRunNode } from "./markdownInterpolation";
import { DOLLAR_REWRITE_IDENTIFIER } from "./widget/celExpr";
import { memoryEntryToRow } from "./widget/memoryRow";
import { buildDollarNodes, buildNodeNameMap } from "./widget/useWidgetData";
import { applySort } from "./widget/widgetData";
import type { MarkdownMemoryVariableSource, MarkdownRunVariableSource, MarkdownVariable } from "./panelTypes";

/**
 * Shape we read off a `useInfiniteCanvasRuns` result. Defined locally so the
 * memoized run-id collector doesn't reach for `typeof latestQuery.data`,
 * which the eslint react-hooks plugin tracks as a dependency on the entire
 * `latestQuery` object.
 */
interface RunsPage {
  runs?: Array<{
    rootEvent?: { id?: string };
  }>;
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
  /** Hierarchical loading flag — `true` while any backing query hasn't settled yet. */
  isLoading: boolean;
  /**
   * `true` while the base memory / run queries are still loading. Excludes the
   * per-run execution side-load so callers can gate each piece of templated
   * text precisely (see {@link markdownTextIsLoading}).
   */
  baseLoading: boolean;
  /** `true` while the per-run execution side-load (for `$["Node"]` refs) loads. */
  sideloadLoading: boolean;
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

  const hasMemoryVar = list.some((v) => v.source?.kind === "memory");
  const wantLatest = list.some((v) => v.source?.kind === "run" && v.source.select === "latest");
  const wantLatestPassed = list.some((v) => v.source?.kind === "run" && v.source.select === "latest_passed");
  const wantLatestFailed = list.some((v) => v.source?.kind === "run" && v.source.select === "latest_failed");

  // Fixed-shape query calls so we never break the rules of hooks regardless
  // of what variables list the panel was edited into.
  const memoryQuery = useCanvasMemoryEntries(canvasId, hasMemoryVar);
  const latestQuery = useInfiniteCanvasRuns(canvasId, {}, wantLatest);
  const passedQuery = useInfiniteCanvasRuns(canvasId, { results: ["RESULT_PASSED"] }, wantLatestPassed);
  const failedQuery = useInfiniteCanvasRuns(canvasId, { results: ["RESULT_FAILED"] }, wantLatestFailed);

  const needsRunSideload = useMemo(() => markdownTemplateReferencesRunNode(textForRunSideload), [textForRunSideload]);

  const latestRunsData = latestQuery.data as RunsQueryData | undefined;
  const passedRunsData = passedQuery.data as RunsQueryData | undefined;
  const failedRunsData = failedQuery.data as RunsQueryData | undefined;

  // Pick the candidate root-event ids for each enabled run query so the
  // execution side-load fetches only the run that will actually be used as
  // a variable value. Dedupes ids so the same root event is fetched once.
  const runRootEventIds = useMemo(() => {
    if (!needsRunSideload) return [] as string[];
    const candidates: Array<RunsQueryData | undefined> = [];
    if (wantLatest) candidates.push(latestRunsData);
    if (wantLatestPassed) candidates.push(passedRunsData);
    if (wantLatestFailed) candidates.push(failedRunsData);
    return collectFirstRunRootEventIds(candidates);
  }, [
    needsRunSideload,
    wantLatest,
    wantLatestPassed,
    wantLatestFailed,
    latestRunsData,
    passedRunsData,
    failedRunsData,
  ]);

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
  const memoryLoading = hasMemoryVar && memoryQuery.isLoading;
  const nodeNameById = useMemo(() => buildNodeNameMap(ctx?.nodes), [ctx?.nodes]);

  const { vars, errors } = useMemo(() => {
    const out: Record<string, unknown> = {};
    const errs: MarkdownVariableError[] = [];
    // Mirror `normalizeDraftVariables` (first-wins) so the preview resolves the
    // same row that save persists. Without this, duplicate names resolve
    // last-wins here while save keeps the first, letting authors preview one
    // value and save a different one.
    const seen = new Set<string>();
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
      });
      out[variable.name] = resolved.value;
      if (resolved.error) errs.push({ name: variable.name, message: resolved.error });
    }
    return { vars: out, errors: errs };
  }, [
    list,
    memoryEntries,
    memoryLoading,
    latestQuery,
    passedQuery,
    failedQuery,
    executionsByRootEventId,
    nodeNameById,
  ]);

  const baseLoading =
    (hasMemoryVar && memoryQuery.isLoading) ||
    (wantLatest && latestQuery.isLoading) ||
    (wantLatestPassed && passedQuery.isLoading) ||
    (wantLatestFailed && failedQuery.isLoading);
  const sideloadLoading = needsRunSideload && runExecutionsLoading;
  const isLoading = baseLoading || sideloadLoading;

  return { vars, isLoading, baseLoading, sideloadLoading, errors };
}

/**
 * Collect unique `rootEvent.id` values from the first run of each provided
 * runs query. Pure helper extracted from `useMarkdownVariables` so the
 * memoized version stays inside the eslint complexity budget.
 */
function collectFirstRunRootEventIds(candidates: Array<RunsQueryData | undefined>): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const data of candidates) {
    const id = data?.pages?.[0]?.runs?.[0]?.rootEvent?.id;
    if (id && !seen.has(id)) {
      seen.add(id);
      ids.push(id);
    }
  }
  return ids;
}

interface ResolveContext {
  memoryEntries: CanvasMemoryEntry[];
  memoryLoading: boolean;
  latestQuery: { isLoading: boolean; data?: RunsQueryDataWithRoot };
  passedQuery: { isLoading: boolean; data?: RunsQueryDataWithRoot };
  failedQuery: { isLoading: boolean; data?: RunsQueryDataWithRoot };
  executionsByRootEventId: Map<string, CanvasesCanvasNodeExecution[]>;
  nodeNameById: Map<string, string>;
}

/** Resolve a single variable to a `{ value, error? }` pair. */
function resolveVariable(variable: MarkdownVariable, ctx: ResolveContext): { value: unknown; error?: string } {
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

interface RunsPageWithRoot {
  runs?: RunRow[];
}
interface RunsQueryDataWithRoot {
  pages?: RunsPageWithRoot[];
}

function pickRunQuery(select: MarkdownRunVariableSource["select"], ctx: ResolveContext) {
  if (select === "latest_passed") return ctx.passedQuery;
  if (select === "latest_failed") return ctx.failedQuery;
  return ctx.latestQuery;
}

function resolveRunVariable(
  source: MarkdownRunVariableSource,
  ctx: ResolveContext,
): { value: unknown; error?: string } {
  const query = pickRunQuery(source.select, ctx);
  const firstRun = query.data?.pages?.[0]?.runs?.[0];
  if (!firstRun) {
    if (query.isLoading) return { value: null };
    return { value: null, error: noRunFoundMessage(source.select) };
  }
  return { value: buildRunVariableValue(firstRun, ctx) };
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

function noRunFoundMessage(select: MarkdownRunVariableSource["select"]): string {
  switch (select) {
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
