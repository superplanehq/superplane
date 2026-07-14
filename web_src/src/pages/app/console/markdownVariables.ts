/**
 * Type schema and validation for markdown-panel variables.
 *
 * Kept in its own module so `panelTypes.ts` stays focused on the per-panel
 * content shapes shared by every editor. The validators here are exposed via
 * `validateMarkdownVariables` and wired into `panelTypes.validatePanelContent`
 * for the markdown kind.
 */

import { RUN_STATUS_FILTER_IDS, isRunStatusFilter, type RunStatusFilter } from "@/ui/Runs/runStatusFilterVocab";

/** Variable identifiers must match this regex so they can appear in `{{ }}` CEL expressions. */
export const MARKDOWN_VARIABLE_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

/** Allowed selectors for `{ kind: "run" }` variables. */
export const MARKDOWN_RUN_SELECTS = ["latest", "latest_passed", "latest_failed"] as const;
export type MarkdownRunSelect = (typeof MARKDOWN_RUN_SELECTS)[number];

/** Sort direction for memory variables. */
export const MARKDOWN_VARIABLE_DIRECTIONS = ["asc", "desc"] as const;
export type MarkdownVariableDirection = (typeof MARKDOWN_VARIABLE_DIRECTIONS)[number];

/**
 * How a memory variable resolves the rows it selected:
 *  - `single` (default): return the first sorted row, so authors can write
 *    `{{ name.field }}` directly.
 *  - `list`: return the full sorted array, unlocking CEL list macros like
 *    `name.map(r, r.field)` / `name.filter(r, r.passed)` inside `{{ }}`.
 */
export const MARKDOWN_VARIABLE_MODES = ["single", "list"] as const;
export type MarkdownVariableMode = (typeof MARKDOWN_VARIABLE_MODES)[number];

/** One property-equality match clause for memory variable selection. */
export interface MarkdownVariableMatch {
  field: string;
  value: string;
}

export interface MarkdownMemoryVariableSource {
  kind: "memory";
  /** Required: the memory namespace to read from. */
  namespace: string;
  /** Optional field name to sort the namespace by before taking the first row. */
  orderBy?: string;
  /** Sort direction when `orderBy` is set. Defaults to `desc` (newest first). */
  direction?: MarkdownVariableDirection;
  /** Optional property-equality filter applied before sorting. */
  matches?: MarkdownVariableMatch[];
  /** Resolution mode — defaults to `single` when omitted for back-compat. */
  mode?: MarkdownVariableMode;
  /**
   * Maximum number of rows to expose when `mode === "list"`. Omitted (or
   * unset) means return every matching row. Ignored when `mode !== "list"`.
   */
  limit?: number;
}

export interface MarkdownRunVariableSource {
  kind: "run";
  /**
   * Which run to pick:
   *  - `latest` — the most recent run regardless of result
   *  - `latest_passed` — the most recent `RESULT_PASSED` run
   *  - `latest_failed` — the most recent `RESULT_FAILED` run
   */
  select: MarkdownRunSelect;
  /**
   * Optional status filter (running / passed / failed / cancelled) applied
   * on top of `select`. Empty or omitted means "all statuses". Applied
   * client-side after the underlying runs query returns so the same
   * `select` bucket is shared across variables regardless of filter.
   */
  statuses?: RunStatusFilter[];
  /**
   * Optional trigger filter — each entry references a trigger node by id
   * or name. Empty or omitted means "all triggers".
   */
  triggers?: string[];
}

export type MarkdownVariableSource = MarkdownMemoryVariableSource | MarkdownRunVariableSource;

export interface MarkdownVariable {
  /** Identifier used in the markdown body as `{{ name.field }}`. */
  name: string;
  source: MarkdownVariableSource;
}

function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

/**
 * Validate the body / title / variables shape used by both markdown and html
 * panels. Returns `null` when valid, a human-readable error otherwise. The
 * two panel types share this validator because their stored content is
 * identical; only the renderer differs.
 */
export function validateMarkdownContent(content: unknown): string | null {
  if (content === undefined || content === null) return null;
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  if (obj.title !== undefined && obj.title !== null && typeof obj.title !== "string") {
    return "content.title must be a string.";
  }
  if (obj.body !== undefined && obj.body !== null && typeof obj.body !== "string") {
    return "content.body must be a string.";
  }
  return validateMarkdownVariables(obj.variables);
}

/**
 * Validate the `variables` array on a markdown panel's content. Returns
 * `null` when valid (or unset) and a human-readable error otherwise. Kept
 * permissive: each individual variable is validated independently so an
 * editor can surface multiple problems through the YAML diff modal.
 */
export function validateMarkdownVariables(raw: unknown): string | null {
  if (raw === undefined || raw === null) return null;
  if (!Array.isArray(raw)) return "content.variables must be an array.";
  const names = new Set<string>();
  for (let i = 0; i < raw.length; i += 1) {
    const item = asObject(raw[i]);
    if (!item) return `content.variables[${i}] must be an object.`;
    if (typeof item.name !== "string" || !MARKDOWN_VARIABLE_NAME_RE.test(item.name)) {
      return `content.variables[${i}].name must be a valid identifier (letters, digits, underscore; not starting with a digit).`;
    }
    if (names.has(item.name)) {
      return `content.variables[${i}].name ${JSON.stringify(item.name)} is duplicated.`;
    }
    names.add(item.name);
    const sourceError = validateMarkdownVariableSource(item.source, i);
    if (sourceError) return sourceError;
  }
  return null;
}

function validateMarkdownVariableSource(raw: unknown, index: number): string | null {
  const source = asObject(raw);
  if (!source) return `content.variables[${index}].source must be an object.`;
  if (source.kind === "memory") return validateMarkdownMemorySource(source, index);
  if (source.kind === "run") return validateMarkdownRunSource(source, index);
  return `content.variables[${index}].source.kind must be "memory" or "run".`;
}

function validateMarkdownMemorySource(source: Record<string, unknown>, index: number): string | null {
  if (typeof source.namespace !== "string" || source.namespace.trim() === "") {
    return `content.variables[${index}].source.namespace must be a non-empty string.`;
  }
  if (source.orderBy !== undefined && source.orderBy !== null && typeof source.orderBy !== "string") {
    return `content.variables[${index}].source.orderBy must be a string.`;
  }
  const directionError = validateMemoryDirection(source.direction, index);
  if (directionError) return directionError;
  const matchesError = validateMemoryMatches(source.matches, index);
  if (matchesError) return matchesError;
  const modeError = validateMemoryMode(source.mode, index);
  if (modeError) return modeError;
  return validateMemoryLimit(source.limit, index);
}

function validateMemoryMode(mode: unknown, index: number): string | null {
  if (mode === undefined || mode === null) return null;
  if (typeof mode !== "string" || !(MARKDOWN_VARIABLE_MODES as readonly string[]).includes(mode)) {
    return `content.variables[${index}].source.mode must be "single" or "list".`;
  }
  return null;
}

function validateMemoryLimit(limit: unknown, index: number): string | null {
  if (limit === undefined || limit === null) return null;
  if (typeof limit !== "number" || !Number.isFinite(limit) || !Number.isInteger(limit) || limit <= 0) {
    return `content.variables[${index}].source.limit must be a positive integer.`;
  }
  return null;
}

function validateMemoryDirection(direction: unknown, index: number): string | null {
  if (direction === undefined || direction === null) return null;
  if (typeof direction !== "string" || !(MARKDOWN_VARIABLE_DIRECTIONS as readonly string[]).includes(direction)) {
    return `content.variables[${index}].source.direction must be "asc" or "desc".`;
  }
  return null;
}

function validateMemoryMatches(matches: unknown, index: number): string | null {
  if (matches === undefined || matches === null) return null;
  if (!Array.isArray(matches)) return `content.variables[${index}].source.matches must be an array.`;
  for (let j = 0; j < matches.length; j += 1) {
    const match = asObject(matches[j]);
    if (!match) return `content.variables[${index}].source.matches[${j}] must be an object.`;
    if (typeof match.field !== "string" || match.field.trim() === "") {
      return `content.variables[${index}].source.matches[${j}].field must be a non-empty string.`;
    }
    if (match.value !== undefined && match.value !== null && typeof match.value !== "string") {
      return `content.variables[${index}].source.matches[${j}].value must be a string.`;
    }
  }
  return null;
}

function validateMarkdownRunSource(source: Record<string, unknown>, index: number): string | null {
  if (typeof source.select !== "string" || !(MARKDOWN_RUN_SELECTS as readonly string[]).includes(source.select)) {
    return `content.variables[${index}].source.select must be one of ${MARKDOWN_RUN_SELECTS.join(", ")}.`;
  }
  const statusesError = validateRunStatusesField(source.statuses, index);
  if (statusesError) return statusesError;
  return validateRunTriggersField(source.triggers, index);
}

function validateRunStatusesField(raw: unknown, index: number): string | null {
  if (raw === undefined || raw === null) return null;
  if (!Array.isArray(raw)) return `content.variables[${index}].source.statuses must be an array.`;
  for (let j = 0; j < raw.length; j += 1) {
    const item = raw[j];
    if (!isRunStatusFilter(item)) {
      return `content.variables[${index}].source.statuses[${j}] must be one of ${RUN_STATUS_FILTER_IDS.join(", ")}.`;
    }
  }
  return null;
}

function validateRunTriggersField(raw: unknown, index: number): string | null {
  if (raw === undefined || raw === null) return null;
  if (!Array.isArray(raw)) return `content.variables[${index}].source.triggers must be an array.`;
  for (let j = 0; j < raw.length; j += 1) {
    const item = raw[j];
    if (typeof item !== "string" || item.trim() === "") {
      return `content.variables[${index}].source.triggers[${j}] must be a non-empty string.`;
    }
  }
  return null;
}

/**
 * Coerce a persisted statuses array into a typed subset of
 * {@link RunStatusFilter}. Returns `undefined` when the array is missing
 * or would end up empty so persistence stays clean (empty === "all").
 */
export function normalizeRunVariableStatuses(raw: unknown): RunStatusFilter[] | undefined {
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
 * Coerce a persisted triggers array into a normalized list (trimmed,
 * deduped). Returns `undefined` when the array is missing or empty so
 * persistence stays clean (empty === "all").
 */
export function normalizeRunVariableTriggers(raw: unknown): string[] | undefined {
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
