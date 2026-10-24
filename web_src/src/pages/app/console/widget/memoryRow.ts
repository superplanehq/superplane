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
      for (let i = 0; i < value.length; i++) {
        const item = value[i];
        if (item && typeof item === "object") {
          const itemRecord = item as Record<string, unknown>;
          const itemId =
            typeof itemRecord.id === "string" && itemRecord.id.length > 0
              ? itemRecord.id
              : typeof itemRecord.id === "number" || typeof itemRecord.id === "bigint"
                ? String(itemRecord.id)
                : `${entry.id}:${i}`;
          out.push({ id: itemId, namespace: entry.namespace, ...itemRecord });
        } else if (item !== undefined) {
          out.push({ id: `${entry.id}:${i}`, namespace: entry.namespace, value: item });
        }
      }
    } else if (value !== undefined) {
      out.push({ id: entry.id, namespace: entry.namespace, value });
    }
  }
  return out;
}
