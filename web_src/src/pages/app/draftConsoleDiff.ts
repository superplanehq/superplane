import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import * as yaml from "js-yaml";
import type { DraftDiffLine, DraftDiffStatus } from "./draftNodeDiff";

type ConsoleSnapshot =
  | {
      panels?: ConsolePanel[];
      layout?: ConsoleLayoutItem[];
    }
  | null
  | undefined;

export type DraftConsoleDiffCounts = { added: number; updated: number; removed: number };

export type DraftConsoleDiffItem = {
  id: string;
  title: string;
  changeType: DraftDiffStatus;
  panel?: ConsolePanel;
  layout?: ConsoleLayoutItem;
  lines: DraftDiffLine[];
};

export type DraftConsoleDiffSummary = {
  items: DraftConsoleDiffItem[];
  addedCount: number;
  updatedCount: number;
  removedCount: number;
};

/**
 * Recursively sort object keys so structurally-identical values produce
 * identical JSON regardless of key insertion order. The committed console is
 * serialized by the backend (Go `json.Marshal` emits map keys alphabetically)
 * while the staged/effective console keeps the editor's insertion order. A
 * plain `JSON.stringify` would treat those two as different and leave the
 * "UNCOMMITTED CHANGES" badge stuck after a commit, so every comparison below
 * canonicalizes through this helper first.
 */
function canonicalize(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(canonicalize);
  }
  if (value && typeof value === "object") {
    const source = value as Record<string, unknown>;
    return Object.keys(source)
      .sort()
      .reduce<Record<string, unknown>>((acc, key) => {
        acc[key] = canonicalize(source[key]);
        return acc;
      }, {});
  }
  return value;
}

function stableStringify(value: unknown): string {
  return JSON.stringify(canonicalize(value));
}

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
  return stableStringify({
    panels: comparablePanels(consoleData?.panels),
    layout: comparableLayout(consoleData?.layout),
  });
}

/** True when draft console differs from live (panels and/or layout). */
export function hasDraftVersusLiveConsoleDiff(liveConsole?: ConsoleSnapshot, draftConsole?: ConsoleSnapshot): boolean {
  return comparableConsoleSnapshot(liveConsole) !== comparableConsoleSnapshot(draftConsole);
}

function panelSnapshot(panel: ConsolePanel | undefined): string {
  return stableStringify({
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

function panelTitle(panel: ConsolePanel | undefined, id: string): string {
  const content = panel?.content;
  if (content && typeof content === "object" && !Array.isArray(content)) {
    const title = (content as Record<string, unknown>).title;
    if (typeof title === "string" && title.trim()) {
      return title.trim();
    }
  }

  return id || "Untitled panel";
}

function panelDiffPath(id: string): string {
  return `console/panels/${id || "unknown"}.yaml`;
}

function formatDiffValueLines(value: unknown): string[] {
  return yaml
    .dump(value === undefined ? null : value, {
      lineWidth: -1,
      noRefs: true,
      sortKeys: true,
    })
    .trimEnd()
    .split("\n");
}

function buildYamlFieldLines(prefix: "+" | "-", key: string, value: unknown): DraftDiffLine[] {
  const valueLines = formatDiffValueLines(value);
  if (valueLines.length === 1) {
    return [{ prefix, text: `${key}: ${valueLines[0]}` }];
  }

  return [{ prefix, text: `${key}:` }, ...valueLines.map((line) => ({ prefix, text: `  ${line}` }))];
}

function comparablePanelFields(panel: ConsolePanel | undefined, layout: ConsoleLayoutItem | undefined) {
  return {
    type: panel?.type ?? "",
    content: panel?.content ?? {},
    layout: layout
      ? {
          x: layout.x ?? 0,
          y: layout.y ?? 0,
          w: layout.w ?? 0,
          h: layout.h ?? 0,
          ...(layout.minW !== undefined ? { minW: layout.minW } : {}),
          ...(layout.minH !== undefined ? { minH: layout.minH } : {}),
        }
      : null,
  };
}

function buildPanelLines(
  prefix: "+" | "-",
  id: string,
  panel: ConsolePanel | undefined,
  layout: ConsoleLayoutItem | undefined,
): DraftDiffLine[] {
  const path = panelDiffPath(id);
  const header: DraftDiffLine[] = [
    { prefix: "meta", text: `diff --git a/${path} b/${path}` },
    { prefix: "meta", text: `--- ${prefix === "-" ? `a/${path}` : "/dev/null"}` },
    { prefix: "meta", text: `+++ ${prefix === "+" ? `b/${path}` : "/dev/null"}` },
    { prefix: "context", text: "@@ -1,0 +1,0 @@" },
  ];
  const fields = comparablePanelFields(panel, layout);

  return [
    ...header,
    ...buildYamlFieldLines(prefix, "id", id),
    ...buildYamlFieldLines(prefix, "type", fields.type),
    ...buildYamlFieldLines(prefix, "content", fields.content),
    ...buildYamlFieldLines(prefix, "layout", fields.layout),
  ];
}

function buildUpdatedPanelLines(
  id: string,
  livePanel: ConsolePanel | undefined,
  draftPanel: ConsolePanel | undefined,
  liveLayout: ConsoleLayoutItem | undefined,
  draftLayout: ConsoleLayoutItem | undefined,
): DraftDiffLine[] {
  const path = panelDiffPath(id);
  const previousFields = comparablePanelFields(livePanel, liveLayout);
  const currentFields = comparablePanelFields(draftPanel, draftLayout);
  const lines: DraftDiffLine[] = [
    { prefix: "meta", text: `diff --git a/${path} b/${path}` },
    { prefix: "meta", text: `--- a/${path}` },
    { prefix: "meta", text: `+++ b/${path}` },
    { prefix: "context", text: "@@ -1,0 +1,0 @@" },
  ];

  (["type", "content", "layout"] as const).forEach((key) => {
    if (stableStringify(previousFields[key]) === stableStringify(currentFields[key])) {
      return;
    }

    lines.push(...buildYamlFieldLines("-", key, previousFields[key]));
    lines.push(...buildYamlFieldLines("+", key, currentFields[key]));
  });

  return lines;
}

export function buildDraftConsoleDiffSummary(
  liveConsole?: ConsoleSnapshot,
  draftConsole?: ConsoleSnapshot,
): DraftConsoleDiffSummary {
  const livePanels = indexPanels(liveConsole?.panels);
  const draftPanels = indexPanels(draftConsole?.panels);
  const liveLayout = indexLayout(liveConsole?.layout);
  const draftLayout = indexLayout(draftConsole?.layout);
  const ids = Array.from(
    new Set([...livePanels.keys(), ...draftPanels.keys(), ...liveLayout.keys(), ...draftLayout.keys()]),
  )
    .filter(Boolean)
    .sort((left, right) => left.localeCompare(right));
  const items: DraftConsoleDiffItem[] = [];
  let addedCount = 0;
  let updatedCount = 0;
  let removedCount = 0;

  ids.forEach((id) => {
    const livePanel = livePanels.get(id);
    const draftPanel = draftPanels.get(id);
    const liveLayoutItem = liveLayout.get(id);
    const draftLayoutItem = draftLayout.get(id);
    const liveExists = !!livePanel || !!liveLayoutItem;
    const draftExists = !!draftPanel || !!draftLayoutItem;

    if (!liveExists && draftExists) {
      items.push({
        id,
        title: panelTitle(draftPanel, id),
        changeType: "added",
        panel: draftPanel,
        layout: draftLayoutItem,
        lines: buildPanelLines("+", id, draftPanel, draftLayoutItem),
      });
      addedCount += 1;
      return;
    }

    if (liveExists && !draftExists) {
      items.push({
        id,
        title: panelTitle(livePanel, id),
        changeType: "removed",
        panel: livePanel,
        layout: liveLayoutItem,
        lines: buildPanelLines("-", id, livePanel, liveLayoutItem),
      });
      removedCount += 1;
      return;
    }

    const panelChanged = panelSnapshot(livePanel) !== panelSnapshot(draftPanel);
    const layoutChanged = layoutSnapshot(liveLayoutItem) !== layoutSnapshot(draftLayoutItem);
    if (panelChanged || layoutChanged) {
      items.push({
        id,
        title: panelTitle(draftPanel ?? livePanel, id),
        changeType: "updated",
        panel: draftPanel,
        layout: draftLayoutItem,
        lines: buildUpdatedPanelLines(id, livePanel, draftPanel, liveLayoutItem, draftLayoutItem),
      });
      updatedCount += 1;
    }
  });

  return { items, addedCount, updatedCount, removedCount };
}

/** Counts changed console items by panel/layout id for the edit-mode header badge. */
export function getDraftConsoleDiffCounts(
  liveConsole?: ConsoleSnapshot,
  draftConsole?: ConsoleSnapshot,
): DraftConsoleDiffCounts {
  const summary = buildDraftConsoleDiffSummary(liveConsole, draftConsole);
  return { added: summary.addedCount, updated: summary.updatedCount, removed: summary.removedCount };
}
