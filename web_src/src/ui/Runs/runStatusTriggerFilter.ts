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
 * Options for trigger-filter matchability / row matching when a resolver
 * is in play. Distinguishes "canvas nodes not loaded yet" from "refs are
 * permanently stale".
 */
export interface TriggerFilterMatchOptions {
  /**
   * How many canvas nodes are available for id/name resolution. When `0`
   * (the empty fallback while `canvas?.spec?.nodes` is still loading),
   * unresolved trigger refs are treated as inconclusive rather than
   * permanently unmatchable — otherwise widgets skip eager paging and
   * markdown variables flash "No run matched…" for valid filters.
   */
  nodeCatalogSize?: number;
}

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
  options?: TriggerFilterMatchOptions,
): boolean {
  if (!filters) return true;

  const statuses = filters.statuses;
  if (statuses && statuses.length > 0) {
    const status = getRunStatus(run);
    if (status === "unknown" || !statuses.includes(status)) return false;
  }

  const triggers = filters.triggers;
  if (triggers && triggers.length > 0) {
    if (!runMatchesTriggerFilter(run.rootEvent?.nodeId, triggers, resolveTriggerReference, options)) return false;
  }

  return true;
}

/**
 * Match a run's root-event node id against a trigger-filter list.
 * Extracted from {@link runMatchesStatusTriggerFilters} to keep complexity
 * under the lint budget and to isolate the empty-catalog fallback.
 */
function runMatchesTriggerFilter(
  triggerNodeId: string | undefined,
  triggers: readonly string[],
  resolveTriggerReference?: TriggerReferenceResolver,
  options?: TriggerFilterMatchOptions,
): boolean {
  if (!triggerNodeId) return false;
  const canonicalIds = resolveTriggerFilterIds(triggers, resolveTriggerReference);
  if (canonicalIds.size === 0) {
    // Empty set: every persisted ref failed to resolve. When the node catalog
    // is still empty that is inconclusive (canvas may be loading) — fall back
    // to raw-ref comparison so UUID filters still match. Once nodes are
    // present, empty means permanently stale.
    if (resolveTriggerReference && (options?.nodeCatalogSize ?? 0) === 0) {
      return triggers.some((reference) => reference.trim() === triggerNodeId);
    }
    return false;
  }
  return canonicalIds.has(triggerNodeId);
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
 *
 * When a resolver is provided but the node catalog is still empty
 * (`nodeCatalogSize === 0`), returns `true` — resolution is inconclusive
 * until nodes arrive, so callers must not treat the filter as permanently
 * stale.
 */
export function triggerFilterCanMatch(
  triggers: readonly string[] | undefined,
  resolveTriggerReference?: TriggerReferenceResolver,
  options?: TriggerFilterMatchOptions,
): boolean {
  if (!triggers || triggers.length === 0) return true;
  if (resolveTriggerReference && (options?.nodeCatalogSize ?? 0) === 0) return true;
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

/**
 * Select buckets used by markdown/html `kind: "run"` variables. `latest_passed`
 * / `latest_failed` hit server-filtered ListRuns feeds, so a client status
 * filter that excludes the bucket's only possible status can never match.
 */
export type RunSelectBucket = "latest" | "latest_passed" | "latest_failed";

/**
 * True when the optional status filter can still match a run from the given
 * select bucket. Empty / omitted statuses always can. Used to skip eager
 * pagination for impossible combos like `latest_passed` + `statuses: [failed]`.
 */
export function runSelectStatusFilterCanMatch(
  select: RunSelectBucket,
  statuses: readonly RunStatusFilter[] | undefined,
): boolean {
  if (!statuses || statuses.length === 0) return true;
  if (select === "latest_passed") return statuses.includes("passed");
  if (select === "latest_failed") return statuses.includes("failed");
  return true;
}

/**
 * Drop status selections that can never appear in the given select bucket.
 * Used when authors switch the Run dropdown so persisted YAML stays coherent.
 */
export function statusesCompatibleWithRunSelect(
  select: RunSelectBucket,
  statuses: readonly RunStatusFilter[] | undefined,
): RunStatusFilter[] | undefined {
  if (!statuses || statuses.length === 0) return undefined;
  if (select === "latest") return [...statuses];
  const required: RunStatusFilter = select === "latest_passed" ? "passed" : "failed";
  const kept = statuses.filter((status) => status === required);
  return kept.length > 0 ? kept : undefined;
}
