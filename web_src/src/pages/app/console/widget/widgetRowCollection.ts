/**
 * Row collection + derived-field helpers shared by the table, chart, and
 * number renderers. Pure functions over loaded infinite-query pages — kept
 * separate from `useWidgetData` so each can be exercised in isolation by
 * its spec file.
 */

import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode } from "@/api-client";

import { DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";

/**
 * Walk the loaded run pages and synthesize execution row objects for the
 * dashboard's table / chart / number renderers. Each row carries the raw
 * execution fields plus derived conveniences:
 *
 * - `status`: lowercase canonical status string (see {@link deriveExecutionStatus}).
 * - `nodeName`: friendly node label resolved per-row via `nodeNameById`.
 * - `durationMs`: created-to-updated elapsed time in milliseconds.
 * - `payload`: the data carried by the run's root event — i.e. the payload
 *   the node received.
 *
 * Iteration stops as soon as `rows.length >= limit`.
 */
export function collectExecutionRows(
  pages: Array<
    | {
        runs?: Array<{
          rootEvent?: { data?: Record<string, unknown> };
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
    for (const run of page?.runs ?? []) {
      for (const exec of run.executions ?? []) {
        if (targetNodeId && exec.nodeId !== targetNodeId) continue;
        rows.push({
          ...exec,
          status: deriveExecutionStatus(exec.state, exec.result),
          nodeName: (exec.nodeId && nodeNameById.get(exec.nodeId)) || exec.nodeId,
          durationMs:
            exec.updatedAt && exec.createdAt ? Date.parse(exec.updatedAt) - Date.parse(exec.createdAt) : undefined,
          payload: run.rootEvent?.data,
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
export type RunRowSource = Record<string, unknown> & {
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
