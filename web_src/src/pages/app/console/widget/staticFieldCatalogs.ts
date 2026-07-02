/**
 * Static field catalogs for the non-memory widget data sources.
 *
 * Memory rows are dynamic — we discover their shape by inspecting the live
 * canvas memory (see {@link useMemoryCatalog}). Executions and runs always
 * produce rows with the same set of fields, so we hard-code the catalog
 * here. The table panel form uses these lists to power the field dropdown,
 * the quick-add column buttons, and the payload editor preview, mirroring
 * the editor affordances memory-backed tables already have.
 *
 * The execution catalog mirrors the row shape built by `collectExecutionRows`
 * in `useWidgetData.ts` — the raw `CanvasesCanvasNodeExecution` fields plus
 * the three derived fields the widget appends to every row. The runs
 * catalog mirrors `CanvasesCanvasRun`.
 *
 * Each entry carries an optional `sample` so the payload-editor preview can
 * render a realistic `{{ field }}` interpolation without having to wait for
 * a live row to come back from the API.
 */

import type { MemoryFieldSummary } from "./useMemoryCatalog";

/**
 * Field catalog for rows produced by the `executions` data source.
 *
 * Keep this in sync with the row shape constructed by `collectExecutionRows`
 * in `widget/useWidgetData.ts` — the raw execution fields plus the derived
 * convenience fields (`status`, `nodeName`, `durationMs`, `payload`).
 * Entries are sorted alphabetically by `field` so the dropdown and
 * quick-add chips stay scannable as the catalog grows.
 */
export const EXECUTIONS_FIELDS: MemoryFieldSummary[] = sortFields([
  { field: "status", sample: "passed" },
  { field: "nodeName", sample: "deploy-prod" },
  { field: "state", sample: "STATE_FINISHED" },
  { field: "result", sample: "RESULT_PASSED" },
  { field: "resultReason", sample: "" },
  { field: "resultMessage", sample: "" },
  { field: "id", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "nodeId", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "canvasId", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "previousExecutionId", sample: "" },
  { field: "createdAt", sample: "2026-01-01T12:00:00Z" },
  { field: "updatedAt", sample: "2026-01-01T12:05:00Z" },
  { field: "durationMs", sample: "300000" },
  { field: "payload", sample: "{}" },
]);

/**
 * Field catalog for rows produced by the `runs` data source. Mirrors the
 * `CanvasesCanvasRun` shape returned by `useInfiniteCanvasRuns` plus the
 * derived convenience fields appended by `collectRunRows` (`status`,
 * `nodeName`, `payload`, `durationMs`, `$`) and the most useful root-event
 * fields. Entries are sorted alphabetically by `field` to match the
 * executions catalog.
 *
 * The `$` entry is a hint: each run row carries a `$` map keyed by node
 * display name with that node's full execution (with `outputs` and a
 * `data` shortcut). Authors drill in with `$["deploy-prod"].outputs.url`
 * — the per-canvas node names aren't enumerated here because executions
 * can fork and a given node may be empty for some runs.
 */
export const RUNS_FIELDS: MemoryFieldSummary[] = sortFields([
  { field: "state", sample: "STATE_STARTED" },
  { field: "result", sample: "RESULT_PASSED" },
  { field: "status", sample: "passed" },
  { field: "nodeName", sample: "deploy-prod" },
  { field: "payload", sample: "{}" },
  { field: "durationMs", sample: "300000" },
  { field: "$", sample: '{ "<node name>": { outputs, data, ... } }' },
  { field: "id", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "canvasId", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "versionId", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "createdAt", sample: "2026-01-01T12:00:00Z" },
  { field: "updatedAt", sample: "2026-01-01T12:05:00Z" },
  { field: "finishedAt", sample: "2026-01-01T12:05:00Z" },
  { field: "rootEvent.nodeId", sample: "00000000-0000-0000-0000-000000000000" },
  { field: "rootEvent.customName", sample: "" },
]);

/**
 * Stable alphabetical sort by `field`. Used by both static catalogs so the
 * dropdown and quick-add chips render in a predictable, scannable order
 * (matches the alphabetical sort `useMemoryCatalog` applies to discovered
 * memory fields — see `collectFieldKeys`).
 */
function sortFields(fields: MemoryFieldSummary[]): MemoryFieldSummary[] {
  return [...fields].sort((a, b) => a.field.localeCompare(b.field));
}

/**
 * Pick the right static catalog for a non-memory data source kind. Returns
 * an empty list for unrecognized kinds; the caller should treat that as
 * "no suggestions available" and fall back to free-text input.
 */
export function staticFieldsForDataSource(kind: "executions" | "runs" | string): MemoryFieldSummary[] {
  if (kind === "executions") return EXECUTIONS_FIELDS;
  if (kind === "runs") return RUNS_FIELDS;
  return [];
}
