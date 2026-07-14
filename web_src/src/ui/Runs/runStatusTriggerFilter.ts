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
    const canonicalIds = resolveTriggerFilterIds(triggers, resolveTriggerReference);
    // Empty set means every persisted ref failed to resolve (stale/deleted
    // node) — no run can match, so stop rather than comparing raw strings
    // that will never equal a live node id.
    if (canonicalIds.size === 0) return false;
    if (!canonicalIds.has(triggerNodeId)) return false;
  }

  return true;
}

/**
 * Resolve a trigger-filter list to canonical node ids. When a resolver is
 * provided, unresolved references are dropped (they cannot match any run).
 * Without a resolver, references are compared as-is against `rootEvent.nodeId`.
 */
export function resolveTriggerFilterIds(
  triggers: readonly string[],
  resolveTriggerReference?: TriggerReferenceResolver,
): Set<string> {
  const canonicalIds = new Set<string>();
  for (const raw of triggers) {
    if (!raw) continue;
    if (resolveTriggerReference) {
      const resolved = resolveTriggerReference(raw);
      if (resolved) canonicalIds.add(resolved);
      continue;
    }
    canonicalIds.add(raw);
  }
  return canonicalIds;
}

/**
 * True when a trigger filter can possibly match a run. Returns `true` when
 * there is no trigger filter, or when at least one reference resolves (or
 * no resolver is provided). Used to skip eager pagination that can never
 * find a match for fully-stale trigger YAML.
 */
export function triggerFilterCanMatch(
  triggers: readonly string[] | undefined,
  resolveTriggerReference?: TriggerReferenceResolver,
): boolean {
  if (!triggers || triggers.length === 0) return true;
  return resolveTriggerFilterIds(triggers, resolveTriggerReference).size > 0;
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
