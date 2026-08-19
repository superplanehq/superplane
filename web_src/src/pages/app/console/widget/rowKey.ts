/**
 * Stable per-row keys used by widget row-action locks. Extracted into its
 * own module so `WidgetRowActionButton.tsx` remains a components-only file
 * (Fast Refresh + eslint `react-refresh/only-export-components`).
 */

/**
 * Stable per-row key used to scope action locks. Prefers the row's `id`
 * when present (memory rows, executions, runs all expose one) and falls
 * back to a deterministic JSON encoding when the source rows don't carry
 * identifiers. The index is only used as a last-resort tiebreaker so
 * locks don't bleed across rows on re-render.
 */
export function rowKeyForRow(row: Record<string, unknown>, index: number): string {
  const id = row.id;
  if (typeof id === "string" && id.length > 0) return id;
  if (typeof id === "number") return String(id);
  if (typeof id === "bigint") return id.toString();
  try {
    return `row:${index}:${JSON.stringify(row, jsonBigIntReplacer)}`;
  } catch {
    return `row:${index}`;
  }
}

function jsonBigIntReplacer(_key: string, value: unknown): unknown {
  return typeof value === "bigint" ? value.toString() : value;
}
