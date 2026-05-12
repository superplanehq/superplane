import { useCallback, useEffect, useMemo, useState } from "react";

type HiddenByNamespace = Record<string, string[]>;

export const CANVAS_MEMORY_HIDDEN_COLUMNS_STORAGE_KEY_PREFIX = "canvasMemoryHiddenColumns";

export function canvasMemoryHiddenColumnsStorageKey(canvasId: string): string {
  return `${CANVAS_MEMORY_HIDDEN_COLUMNS_STORAGE_KEY_PREFIX}:${canvasId}`;
}

function readBlob(canvasId: string): HiddenByNamespace {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.localStorage.getItem(canvasMemoryHiddenColumnsStorageKey(canvasId));
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    const result: HiddenByNamespace = {};
    for (const [namespace, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (Array.isArray(value) && value.every((v) => typeof v === "string")) {
        result[namespace] = value as string[];
      }
    }
    return result;
  } catch {
    return {};
  }
}

function writeBlob(canvasId: string, blob: HiddenByNamespace) {
  if (typeof window === "undefined") return;
  try {
    if (Object.keys(blob).length === 0) {
      window.localStorage.removeItem(canvasMemoryHiddenColumnsStorageKey(canvasId));
      return;
    }
    window.localStorage.setItem(canvasMemoryHiddenColumnsStorageKey(canvasId), JSON.stringify(blob));
  } catch {
    // localStorage may be unavailable or quota-exceeded; preferences are non-critical.
  }
}

export type CanvasMemoryColumnVisibility = {
  hidden: Set<string>;
  visibleColumns: string[];
  toggle: (column: string) => void;
  showAll: () => void;
  hideAll: () => void;
};

export function useCanvasMemoryColumnVisibility(
  canvasId: string,
  namespace: string,
  allColumns: string[],
): CanvasMemoryColumnVisibility {
  const [hidden, setHidden] = useState<string[]>(() => readBlob(canvasId)[namespace] ?? []);

  useEffect(() => {
    setHidden(readBlob(canvasId)[namespace] ?? []);
  }, [canvasId, namespace]);

  const persist = useCallback(
    (next: string[]) => {
      const blob = readBlob(canvasId);
      if (next.length === 0) {
        delete blob[namespace];
      } else {
        blob[namespace] = next;
      }
      writeBlob(canvasId, blob);
      setHidden(next);
    },
    [canvasId, namespace],
  );

  const hiddenSet = useMemo(() => new Set(hidden), [hidden]);
  const visibleColumns = useMemo(() => allColumns.filter((c) => !hiddenSet.has(c)), [allColumns, hiddenSet]);

  const toggle = useCallback(
    (column: string) => {
      const next = hiddenSet.has(column) ? hidden.filter((c) => c !== column) : [...hidden, column];
      persist(next);
    },
    [hidden, hiddenSet, persist],
  );

  const showAll = useCallback(() => persist([]), [persist]);
  const hideAll = useCallback(() => persist([...allColumns]), [allColumns, persist]);

  return { hidden: hiddenSet, visibleColumns, toggle, showAll, hideAll };
}
