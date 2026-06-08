import { useMemo } from "react";

import { useCanvasMemoryEntries, type CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { memoryEntryToRow } from "./memoryRow";

export interface MemoryNamespaceSummary {
  namespace: string;
  count: number;
}

export interface MemoryFieldSummary {
  field: string;
  sample?: string;
}

const META_KEYS = new Set(["id", "namespace"]);

function collectFieldKeys(entries: CanvasMemoryEntry[], namespace: string): MemoryFieldSummary[] {
  const keys = new Set<string>();
  const samples = new Map<string, unknown>();
  for (const entry of entries) {
    if (entry.namespace !== namespace) continue;
    const row = memoryEntryToRow(entry);
    for (const [key, value] of Object.entries(row)) {
      if (META_KEYS.has(key)) continue;
      keys.add(key);
      if (!samples.has(key) && value !== undefined && value !== "") {
        samples.set(key, value);
      }
    }
  }
  return Array.from(keys)
    .sort((a, b) => a.localeCompare(b))
    .map((field) => {
      const raw = samples.get(field);
      const sample = raw == null ? undefined : String(raw).slice(0, 48);
      return { field, sample };
    });
}

export function useMemoryCatalog(canvasId: string | undefined, namespace?: string) {
  const query = useCanvasMemoryEntries(canvasId ?? "", Boolean(canvasId));

  const namespaces = useMemo((): MemoryNamespaceSummary[] => {
    const counts = new Map<string, number>();
    for (const entry of query.data ?? []) {
      const ns = entry.namespace || "";
      if (!ns) continue;
      counts.set(ns, (counts.get(ns) ?? 0) + 1);
    }
    return Array.from(counts.entries())
      .map(([ns, count]) => ({ namespace: ns, count }))
      .sort((a, b) => a.namespace.localeCompare(b.namespace));
  }, [query.data]);

  const fields = useMemo((): MemoryFieldSummary[] => {
    if (!namespace?.trim() || !query.data) return [];
    return collectFieldKeys(query.data, namespace.trim());
  }, [namespace, query.data]);

  return {
    namespaces,
    fields,
    isLoading: query.isLoading,
    isEmpty: (query.data?.length ?? 0) === 0,
  };
}

export function suggestColumnFormat(field: string): "status" | "relative" | "datetime" | "link" | "duration" | "text" {
  const lower = field.toLowerCase();
  if (lower === "status" || lower === "state" || lower === "health") return "status";
  if (lower.endsWith("_at") || lower.includes("created") || lower.includes("updated")) return "relative";
  if (lower === "url" || lower === "link" || lower === "href") return "link";
  // `durationMs` (and any `*Ms` numeric field) renders best with the duration
  // formatter, which turns ms counts into "5m 30s" style strings.
  if (lower === "durationms" || lower.endsWith("durationms")) return "duration";
  return "text";
}

/**
 * Build a sample row by pairing each discovered field with the first observed
 * sample value. Used by the payload editor preview to show how `{{ expr }}`
 * resolves for a concrete row without needing live memory.
 */
export function sampleRowFromFields(fields: MemoryFieldSummary[]): Record<string, unknown> {
  const row: Record<string, unknown> = {};
  for (const f of fields) {
    row[f.field] = f.sample ?? "";
  }
  return row;
}
