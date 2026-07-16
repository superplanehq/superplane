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

const AVATAR_FIELD_NAMES = new Set(["avatar", "avatar_url", "imageurl", "image_url", "photourl", "photo_url"]);

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

export function suggestColumnFormat(
  field: string,
): "status" | "relative" | "datetime" | "link" | "duration" | "avatar" | "text" {
  const lower = field.toLowerCase();
  if (lower === "status" || lower === "state" || lower === "health") return "status";
  if (lower.endsWith("_at") || lower.includes("created") || lower.includes("updated")) return "relative";
  // Avatar-like fields must be checked before the generic URL/link heuristic
  // so `avatar_url` / `avatarUrl` / `image_url` land on the avatar renderer
  // instead of the plain link one.
  if (AVATAR_FIELD_NAMES.has(lower) || lower.endsWith("avatarurl") || lower.endsWith("_avatar_url")) {
    return "avatar";
  }
  if (lower === "url" || lower === "link" || lower === "href") return "link";
  // `durationMs` (and any `*Ms` numeric field) renders best with the duration
  // formatter, which turns ms counts into "5m 30s" style strings.
  if (lower === "durationms" || lower.endsWith("durationms")) return "duration";
  return "text";
}

/**
 * Build a sample row by pairing each discovered field with the first observed
 * sample value. Dotted paths (`rootEvent.nodeId`) nest into sub-objects so
 * they resolve through the same walker (`getValueAtPath`) that widget CEL
 * uses at runtime; without this, `pathOrRaw` previews would miss real field
 * paths whenever no live row is available.
 */
export function sampleRowFromFields(fields: MemoryFieldSummary[]): Record<string, unknown> {
  const row: Record<string, unknown> = {};
  // Shallow paths first so a leaf write for `rootEvent.nodeId` never clobbers
  // an already-nested `rootEvent` object built by an earlier deeper entry.
  const sorted = [...fields].sort((a, b) => depthOf(a.field) - depthOf(b.field));
  for (const f of sorted) {
    setSamplePath(row, f.field, f.sample ?? "");
  }
  return row;
}

function depthOf(path: string): number {
  return path.split(".").length;
}

function setSamplePath(target: Record<string, unknown>, path: string, value: unknown): void {
  const segments = path.split(".").filter((s) => s.length > 0);
  if (segments.length === 0) return;
  let cursor: Record<string, unknown> = target;
  for (let i = 0; i < segments.length - 1; i++) {
    const key = segments[i];
    const existing = cursor[key];
    if (existing == null || typeof existing !== "object" || Array.isArray(existing)) {
      cursor[key] = {};
    }
    cursor = cursor[key] as Record<string, unknown>;
  }
  cursor[segments[segments.length - 1]] = value;
}
