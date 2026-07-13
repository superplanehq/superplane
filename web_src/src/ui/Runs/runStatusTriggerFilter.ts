/**
 * Shared client-side status/trigger filter for runs.
 *
 * Used by both the canvas runs sidebar and the console run datasource, so
 * the filter semantics stay identical across surfaces. Empty / omitted
 * `statuses` and `triggers` mean "no filter" (pass everything).
 */

import type { CanvasesCanvasRun } from "@/api-client";
import { getRunStatus, type RunStatusFilter } from "./runPresentation";

export interface RunStatusTriggerFilters {
  statuses?: readonly RunStatusFilter[];
  triggers?: readonly string[];
}

/**
 * Resolve a raw trigger reference (id or name) to a canonical id. When
 * omitted, the reference is compared as-is against `rootEvent.nodeId`.
 */
export type TriggerReferenceResolver = (reference: string) => string | undefined;

/**
 * Return true when the run passes the (optional) status and trigger
 * filters. Both filter dimensions are ORed within a dimension and ANDed
 * across dimensions: "any of the selected statuses AND any of the
 * selected triggers".
 *
 * Empty arrays (or undefined) short-circuit to a match so callers can
 * pass filter shapes straight from persisted state without checking.
 */
export function runMatchesStatusTriggerFilters(
  run: CanvasesCanvasRun,
  filters: RunStatusTriggerFilters | undefined,
  resolveTriggerReference?: TriggerReferenceResolver,
): boolean {
  if (!filters) return true;

  const statuses = filters.statuses;
  if (statuses && statuses.length > 0) {
    const status = getRunStatus(run);
    if (status === "unknown" || !statuses.includes(status)) return false;
  }

  const triggers = filters.triggers;
  if (triggers && triggers.length > 0) {
    const triggerNodeId = run.rootEvent?.nodeId;
    if (!triggerNodeId) return false;
    const canonicalIds = new Set<string>();
    for (const raw of triggers) {
      const resolved = resolveTriggerReference?.(raw) ?? raw;
      if (resolved) canonicalIds.add(resolved);
    }
    if (!canonicalIds.has(triggerNodeId)) return false;
  }

  return true;
}

/**
 * True when either dimension has a non-empty selection. Handy for
 * conditionally applying filters or short-circuiting count queries that
 * rely on server-side totals.
 */
export function hasRunStatusTriggerFilters(filters: RunStatusTriggerFilters | undefined): boolean {
  if (!filters) return false;
  const statuses = filters.statuses;
  const triggers = filters.triggers;
  return (statuses?.length ?? 0) > 0 || (triggers?.length ?? 0) > 0;
}
