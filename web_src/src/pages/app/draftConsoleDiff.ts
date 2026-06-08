import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

type ConsoleSnapshot = {
  panels?: ConsolePanel[];
  layout?: ConsoleLayoutItem[];
} | null | undefined;

export type DraftConsoleDiffCounts = { added: number; updated: number; removed: number };

function comparablePanels(panels: ConsolePanel[] | undefined): unknown[] {
  return (panels ?? [])
    .map((panel) => ({
      id: panel.id ?? "",
      type: panel.type ?? "",
      content: panel.content ?? {},
    }))
    .sort((left, right) => left.id.localeCompare(right.id));
}

function comparableLayout(layout: ConsoleLayoutItem[] | undefined): unknown[] {
  return (layout ?? [])
    .map((item) => ({
      i: item.i ?? "",
      x: item.x ?? 0,
      y: item.y ?? 0,
      w: item.w ?? 0,
      h: item.h ?? 0,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    }))
    .sort((left, right) => left.i.localeCompare(right.i));
}

function comparableConsoleSnapshot(consoleData?: ConsoleSnapshot): string {
  return JSON.stringify({
    panels: comparablePanels(consoleData?.panels),
    layout: comparableLayout(consoleData?.layout),
  });
}

/** True when draft console differs from live (panels and/or layout). */
export function hasDraftVersusLiveConsoleDiff(
  liveConsole?: ConsoleSnapshot,
  draftConsole?: ConsoleSnapshot,
): boolean {
  return comparableConsoleSnapshot(liveConsole) !== comparableConsoleSnapshot(draftConsole);
}

function panelSnapshot(panel: ConsolePanel | undefined): string {
  return JSON.stringify({
    type: panel?.type ?? "",
    content: panel?.content ?? {},
  });
}

function layoutSnapshot(item: ConsoleLayoutItem | undefined): string {
  return JSON.stringify({
    x: item?.x ?? 0,
    y: item?.y ?? 0,
    w: item?.w ?? 0,
    h: item?.h ?? 0,
    ...(item?.minW !== undefined ? { minW: item.minW } : {}),
    ...(item?.minH !== undefined ? { minH: item.minH } : {}),
  });
}

function indexPanels(panels: ConsolePanel[] | undefined): Map<string, ConsolePanel> {
  return new Map((panels ?? []).map((panel) => [panel.id ?? "", panel]));
}

function indexLayout(layout: ConsoleLayoutItem[] | undefined): Map<string, ConsoleLayoutItem> {
  return new Map((layout ?? []).map((item) => [item.i ?? "", item]));
}

/** Counts changed console items by panel/layout id for the edit-mode header badge. */
export function getDraftConsoleDiffCounts(
  liveConsole?: ConsoleSnapshot,
  draftConsole?: ConsoleSnapshot,
): DraftConsoleDiffCounts {
  const livePanels = indexPanels(liveConsole?.panels);
  const draftPanels = indexPanels(draftConsole?.panels);
  const liveLayout = indexLayout(liveConsole?.layout);
  const draftLayout = indexLayout(draftConsole?.layout);
  const ids = new Set([...livePanels.keys(), ...draftPanels.keys(), ...liveLayout.keys(), ...draftLayout.keys()]);
  const counts = { added: 0, updated: 0, removed: 0 };

  ids.forEach((id) => {
    const liveExists = livePanels.has(id) || liveLayout.has(id);
    const draftExists = draftPanels.has(id) || draftLayout.has(id);
    if (!liveExists && draftExists) {
      counts.added += 1;
      return;
    }

    if (liveExists && !draftExists) {
      counts.removed += 1;
      return;
    }

    const panelChanged = panelSnapshot(livePanels.get(id)) !== panelSnapshot(draftPanels.get(id));
    const layoutChanged = layoutSnapshot(liveLayout.get(id)) !== layoutSnapshot(draftLayout.get(id));
    if (panelChanged || layoutChanged) {
      counts.updated += 1;
    }
  });

  return counts;
}
