/**
 * Shared low-level validation helpers used across the console panel content
 * validators. Kept in its own module so `panelTypes.ts` and
 * `nodesPanelContent.ts` (and any future per-panel validator) can consume the
 * same rules without duplicating them.
 */

/** Narrow an unknown value to a plain object; returns null for arrays / non-objects. */
export function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

/** Error string for an optional field that, when present, must be a string. */
export function optionalStringError(field: string, value: unknown): string | null {
  if (value !== undefined && value !== null && typeof value !== "string") {
    return `${field} must be a string.`;
  }
  return null;
}

/** Error string for an optional field that, when present, must be a boolean. */
export function optionalBooleanError(field: string, value: unknown): string | null {
  if (value !== undefined && typeof value !== "boolean") {
    return `${field} must be a boolean.`;
  }
  return null;
}
