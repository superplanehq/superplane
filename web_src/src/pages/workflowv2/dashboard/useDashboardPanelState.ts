import { useCallback, useEffect, useRef, useState } from "react";

import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";

const SAVE_DEBOUNCE_MS = 500;

export function useDashboardPanelState(
  panels: DashboardPanel[],
  layout: DashboardLayoutItem[],
  onChange: (next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => void,
) {
  const [localPanels, setLocalPanels] = useState<DashboardPanel[]>(panels);
  const [localLayout, setLocalLayout] = useState<DashboardLayoutItem[]>(layout);
  const lastPropsHashRef = useRef<string>("");

  useEffect(() => {
    const next = JSON.stringify({ panels, layout });
    if (next !== lastPropsHashRef.current) {
      lastPropsHashRef.current = next;
      setLocalPanels(panels);
      setLocalLayout(layout);
    }
  }, [panels, layout]);

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<{ panels: DashboardPanel[]; layout: DashboardLayoutItem[] } | null>(null);
  const queueSave = useCallback(
    (nextPanels: DashboardPanel[], nextLayout: DashboardLayoutItem[]) => {
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
    (name: string) => {
      const slug = name
        .toLowerCase()
        .trim()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "")
        .replace(/-+/g, "-")
        .replace(/^-|-$/g, "");
      const id = slug || `panel-${Math.random().toString(36).slice(2, 10)}`;
      const newPanel: DashboardPanel = { id, type: "markdown", content: { body: "" } };
      const maxBottom = localLayout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
      const newLayoutItem: DashboardLayoutItem = { i: id, x: 0, y: maxBottom, w: 12, h: 6, minW: 2, minH: 2 };
      const nextPanels = [...localPanels, newPanel];
      const nextLayout = [...localLayout, newLayoutItem];
      setLocalPanels(nextPanels);
      setLocalLayout(nextLayout);
      queueSave(nextPanels, nextLayout);
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

  return {
    localPanels,
    handleAddPanel,
    handleDeletePanel,
    handlePanelContentChange,
  };
}
