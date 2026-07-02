import { useCallback, useEffect, useRef, useState } from "react";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import { templateForPanelType, type PanelType } from "./panelTypes";

const SAVE_DEBOUNCE_MS = 500;

export function useConsolePanelState(
  panels: ConsolePanel[],
  layout: ConsoleLayoutItem[],
  onChange: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void,
  onEffectiveChange?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void,
) {
  const [localPanels, setLocalPanels] = useState<ConsolePanel[]>(panels);
  const [localLayout, setLocalLayout] = useState<ConsoleLayoutItem[]>(layout);
  const lastPropsHashRef = useRef<string>("");

  useEffect(() => {
    const next = JSON.stringify({ panels, layout });
    if (next !== lastPropsHashRef.current) {
      lastPropsHashRef.current = next;
      setLocalPanels(panels);
      setLocalLayout(layout);
    }
  }, [panels, layout]);

  useEffect(() => {
    onEffectiveChange?.({ panels: localPanels, layout: localLayout });
  }, [localLayout, localPanels, onEffectiveChange]);

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<{ panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | null>(null);
  const queueSave = useCallback(
    (nextPanels: ConsolePanel[], nextLayout: ConsoleLayoutItem[]) => {
      pendingRef.current = { panels: nextPanels, layout: nextLayout };
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        const pending = pendingRef.current;
        if (!pending) return;
        onChange(pending);
        pendingRef.current = null;
      }, SAVE_DEBOUNCE_MS);
    },
    [onChange],
  );

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      const pending = pendingRef.current;
      if (pending) onChange(pending);
    };
  }, [onChange]);

  const handleAddPanel = useCallback(
    (name: string, type: PanelType = "markdown") => {
      const trimmedName = name.trim();
      const slug = trimmedName
        .toLowerCase()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "")
        .replace(/-+/g, "-")
        .replace(/^-|-$/g, "");
      const baseId = slug || `${type}-${Math.random().toString(36).slice(2, 8)}`;
      const id = uniquePanelId(localPanels, baseId);
      const newPanel: ConsolePanel = { id, type, content: templateForPanelType(type, trimmedName) };
      const maxBottom = localLayout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
      const newLayoutItem: ConsoleLayoutItem = { i: id, x: 0, y: maxBottom, w: 12, h: 6, minW: 2, minH: 2 };
      const nextPanels = [...localPanels, newPanel];
      const nextLayout = [...localLayout, newLayoutItem];
      setLocalPanels(nextPanels);
      setLocalLayout(nextLayout);
      queueSave(nextPanels, nextLayout);
      return id;
    },
    [localLayout, localPanels, queueSave],
  );

  const handleDeletePanel = useCallback(
    (id: string) => {
      const nextPanels = localPanels.filter((p) => p.id !== id);
      const nextLayout = localLayout.filter((l) => l.i !== id);
      setLocalPanels(nextPanels);
      setLocalLayout(nextLayout);
      queueSave(nextPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  const handlePanelContentChange = useCallback(
    (id: string, content: Record<string, unknown>) => {
      const nextPanels = localPanels.map((p) => (p.id === id ? { ...p, content } : p));
      setLocalPanels(nextPanels);
      queueSave(nextPanels, localLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  const handleLayoutChange = useCallback(
    (nextLayout: ConsoleLayoutItem[]) => {
      // Drop layout entries for panels that no longer exist. This guards
      // against react-grid-layout emitting stale entries during transitions.
      const validIds = new Set(localPanels.map((p) => p.id));
      const filtered = nextLayout.filter((item) => validIds.has(item.i));
      const isSame =
        filtered.length === localLayout.length &&
        filtered.every((item, index) => {
          const previous = localLayout[index];
          if (!previous || previous.i !== item.i) return false;
          return (
            previous.x === item.x &&
            previous.y === item.y &&
            previous.w === item.w &&
            previous.h === item.h &&
            previous.minW === item.minW &&
            previous.minH === item.minH
          );
        });
      if (isSame) return;
      setLocalLayout(filtered);
      queueSave(localPanels, filtered);
    },
    [localLayout, localPanels, queueSave],
  );

  return {
    localPanels,
    localLayout,
    handleAddPanel,
    handleDeletePanel,
    handlePanelContentChange,
    handleLayoutChange,
  };
}

/**
 * Append a numeric suffix (`-2`, `-3`, …) to the base id until it doesn't
 * collide with any existing panel. Keeps slug-based ids stable for the
 * common case where the user just adds a uniquely-named panel.
 */
function uniquePanelId(panels: ConsolePanel[], base: string): string {
  const taken = new Set(panels.map((p) => p.id));
  if (!taken.has(base)) return base;
  for (let i = 2; i < 1000; i += 1) {
    const candidate = `${base}-${i}`;
    if (!taken.has(candidate)) return candidate;
  }
  return `${base}-${Math.random().toString(36).slice(2, 8)}`;
}
