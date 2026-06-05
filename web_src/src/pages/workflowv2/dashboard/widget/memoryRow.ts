import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { getValueAtPath } from "./fieldPath";

/**
 * Hoist memory entry `values` to the row root while preserving server-side
 * metadata (`id`, `namespace`, `createdAt`, `updatedAt`). Stored values win
 * conflicts so author-defined keys (e.g. a memory record with a `createdAt`
 * column) still surface their own value.
 */
export function memoryEntryToRow(entry: CanvasMemoryEntry): Record<string, unknown> {
  const values =
    entry.values && typeof entry.values === "object" && !Array.isArray(entry.values)
      ? (entry.values as Record<string, unknown>)
      : {};
  return {
    id: entry.id,
    namespace: entry.namespace,
    createdAt: entry.createdAt,
    updatedAt: entry.updatedAt,
    ...values,
  };
}

export function flattenMemoryEntries(
  entries: CanvasMemoryEntry[],
  namespace: string,
  fieldPath?: string,
): Record<string, unknown>[] {
  const filtered = entries.filter((entry) => entry.namespace === namespace);
  if (!fieldPath) {
    return filtered.map(memoryEntryToRow);
  }
  const out: Record<string, unknown>[] = [];
  for (const entry of filtered) {
    const value = getValueAtPath(entry.values, fieldPath);
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item && typeof item === "object") {
          out.push({ id: entry.id, namespace: entry.namespace, ...(item as Record<string, unknown>) });
        } else if (item !== undefined) {
          out.push({ id: entry.id, namespace: entry.namespace, value: item });
        }
      }
    } else if (value !== undefined) {
      out.push({ id: entry.id, namespace: entry.namespace, value });
    }
  }
  return out;
}
