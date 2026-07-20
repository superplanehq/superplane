/**
 * Vocabulary-only module for the four run-status filter categories used
 * across the app (runs sidebar, console runs datasource, console run
 * variables). Kept independent of `runPresentation.ts` — which pulls in
 * page-level mappers and utils — so the schema files under
 * `web_src/src/pages/app/console/` can validate filter values without
 * dragging heavy dependencies (and creating import cycles).
 */

/** The four filterable outcomes shown in every runs filter surface. */
export const RUN_STATUS_FILTER_IDS = ["running", "passed", "failed", "cancelled"] as const;

/** Type produced by {@link RUN_STATUS_FILTER_IDS} — the union of allowed filter ids. */
export type RunStatusFilter = (typeof RUN_STATUS_FILTER_IDS)[number];

/** Fast membership test for {@link RUN_STATUS_FILTER_IDS}. */
export const RUN_STATUS_FILTER_SET: ReadonlySet<RunStatusFilter> = new Set(RUN_STATUS_FILTER_IDS);

/** True when `value` is one of the allowed filter ids. */
export function isRunStatusFilter(value: unknown): value is RunStatusFilter {
  return typeof value === "string" && RUN_STATUS_FILTER_SET.has(value as RunStatusFilter);
}
