import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

/**
 * Counts distinct namespace buckets in a canvas-memory entry list, matching the
 * grouping used by the Memory view (entries with an empty/whitespace namespace
 * collapse into a single "(no namespace)" bucket).
 */
export function countMemoryNamespaces(entries: CanvasMemoryEntry[]): number {
  const namespaces = new Set<string>();
  for (const entry of entries) {
    const ns = entry.namespace?.trim() ? entry.namespace.trim() : "(no namespace)";
    namespaces.add(ns);
  }
  return namespaces.size;
}
