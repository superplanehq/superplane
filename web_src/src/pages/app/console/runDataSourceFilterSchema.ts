/**
 * Validate and normalize optional status / trigger filters on runs
 * datasources (widget panels and markdown / html run variables).
 */

import { RUN_STATUS_FILTER_IDS, isRunStatusFilter, type RunStatusFilter } from "@/ui/Runs/runStatusFilterVocab";

/**
 * Validate a persisted runs status filter array. Accepts undefined /
 * null / empty (meaning "all statuses") and any subset of the shared
 * {@link RunStatusFilter} vocabulary; anything else is rejected with a
 * message listing the allowed values.
 */
export function validateRunStatusesArray(value: unknown, fieldPath: string): string | null {
  if (value == null) return null;
  if (!Array.isArray(value)) return `${fieldPath} must be an array.`;
  for (let i = 0; i < value.length; i += 1) {
    const item = value[i];
    if (!isRunStatusFilter(item)) {
      return `${fieldPath}[${i}] must be one of ${RUN_STATUS_FILTER_IDS.join(", ")}.`;
    }
  }
  return null;
}

/**
 * Validate a persisted runs trigger filter array. Accepts undefined /
 * null / empty (meaning "all triggers") and any list of non-empty
 * strings; individual entries are matched at runtime against the
 * canvas nodes so unknown ids simply fail to match rather than fail
 * validation.
 */
export function validateRunTriggersArray(value: unknown, fieldPath: string): string | null {
  if (value == null) return null;
  if (!Array.isArray(value)) return `${fieldPath} must be an array.`;
  for (let i = 0; i < value.length; i += 1) {
    const item = value[i];
    if (typeof item !== "string" || item.trim() === "") {
      return `${fieldPath}[${i}] must be a non-empty string.`;
    }
  }
  return null;
}

/**
 * Coerce a persisted statuses array into a typed subset of the shared
 * {@link RunStatusFilter} vocabulary. Returns `undefined` when the result
 * would be empty so YAML round-trips stay clean (empty === "all").
 */
export function normalizeRunStatuses(raw: unknown): RunStatusFilter[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: RunStatusFilter[] = [];
  const seen = new Set<RunStatusFilter>();
  for (const item of raw) {
    if (!isRunStatusFilter(item)) continue;
    if (seen.has(item)) continue;
    seen.add(item);
    out.push(item);
  }
  return out.length > 0 ? out : undefined;
}

/**
 * Coerce a persisted triggers array into a normalized list of non-empty
 * strings (trimmed, deduped). Returns `undefined` when the result would
 * be empty so YAML round-trips stay clean (empty === "all").
 */
export function normalizeRunTriggers(raw: unknown): string[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of raw) {
    if (typeof item !== "string") continue;
    const trimmed = item.trim();
    if (!trimmed || seen.has(trimmed)) continue;
    seen.add(trimmed);
    out.push(trimmed);
  }
  return out.length > 0 ? out : undefined;
}

export type NormalizedRunsDataSource = {
  kind: "runs";
  limit?: number;
  statuses?: RunStatusFilter[];
  triggers?: string[];
};

export type NormalizedExecutionsDataSource = {
  kind: "executions";
  node?: string;
  limit?: number;
};

export type NormalizedMemoryDataSource = {
  kind: "memory";
  namespace: string;
  fieldPath?: string;
};

export type NormalizedTableDataSource =
  | NormalizedRunsDataSource
  | NormalizedExecutionsDataSource
  | NormalizedMemoryDataSource;

export function normalizeRunsDataSource(ds: Record<string, unknown>): NormalizedRunsDataSource {
  const statuses = normalizeRunStatuses(ds.statuses);
  const triggers = normalizeRunTriggers(ds.triggers);
  return {
    kind: "runs",
    limit: optionalNumber(ds.limit),
    ...(statuses ? { statuses } : {}),
    ...(triggers ? { triggers } : {}),
  };
}

export function normalizeTableDataSource(raw: unknown): NormalizedTableDataSource {
  const ds = asObject(raw);
  if (ds?.kind === "executions") {
    return {
      kind: "executions",
      node: stringOrUndefined(ds.node),
      limit: optionalNumber(ds.limit),
    };
  }
  if (ds?.kind === "runs") return normalizeRunsDataSource(ds);
  if (ds?.kind === "memory") {
    return {
      kind: "memory",
      namespace: typeof ds.namespace === "string" ? ds.namespace : "",
      fieldPath: stringOrUndefined(ds.fieldPath),
    };
  }
  return { kind: "memory", namespace: "" };
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function stringOrUndefined(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function asObject(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}
