import type { ConsoleLayoutItem, ConsolePage, ConsolePanel } from "@/hooks/useCanvasData";
import * as yaml from "js-yaml";
import { DEFAULT_CONSOLE_PAGE_ID, DEFAULT_CONSOLE_PAGE_NAME } from "./console/consoleYaml";
import type { DraftDiffLine, DraftDiffStatus } from "./draftNodeDiff";

/**
 * Snapshot shape the diff engine operates on. The console is multi-page
 * and panel ids are only unique within a page, so every internal index
 * uses a composite `pageId::panelId` key. Two pages that happen to share
 * a panel id (e.g. two markdown panels both called `overview`) are
 * treated as distinct panels, and moving a panel across pages surfaces
 * as a removal + add, not a silent update.
 */
type ConsoleSnapshot =
  | {
      pages?: ConsolePage[];
    }
  | null
  | undefined;

/**
 * Composite key that combines the page id with the panel id (or layout
 * item id). We use a delimiter that cannot appear in slugified ids so
 * the composite key can always be split back into `pageId` + `panelId`.
 */
const COMPOSITE_DELIMITER = "\u0000";

type ScopedPanel = { pageId: string; panel: ConsolePanel };
type ScopedLayout = { pageId: string; item: ConsoleLayoutItem };

/**
 * Normalize the page list for equality comparisons only. A single
 * default page (`main`/`Main`) with no panels or layout is
 * semantically equivalent to zero pages — that is the on-disk shape
 * committed YAML produces for a fresh or fully-cleared console. Without
 * this collapse, deleting the last panel on the default page would
 * false-positive as an uncommitted change and the console header would
 * show "UNCOMMITTED CHANGES" even though re-exporting produces
 * byte-identical YAML.
 */
function normalizePagesForComparison(pages: ConsolePage[]): ConsolePage[] {
  if (pages.length !== 1) return pages;
  const only = pages[0]!;
  if ((only.panels?.length ?? 0) > 0 || (only.layout?.length ?? 0) > 0) return pages;
  if (only.id !== DEFAULT_CONSOLE_PAGE_ID) return pages;
  if (only.name && only.name !== DEFAULT_CONSOLE_PAGE_NAME) return pages;
  return [];
}

function normalizedPages(consoleData: ConsoleSnapshot): ConsolePage[] {
  return normalizePagesForComparison(consoleData?.pages ?? []);
}

function scopedPanels(consoleData: ConsoleSnapshot): ScopedPanel[] {
  const out: ScopedPanel[] = [];
  for (const page of normalizedPages(consoleData)) {
    for (const panel of page.panels) out.push({ pageId: page.id, panel });
  }
  return out;
}

function scopedLayout(consoleData: ConsoleSnapshot): ScopedLayout[] {
  const out: ScopedLayout[] = [];
  for (const page of normalizedPages(consoleData)) {
    for (const item of page.layout) out.push({ pageId: page.id, item });
  }
  return out;
}

function compositeKey(pageId: string, id: string): string {
  return `${pageId}${COMPOSITE_DELIMITER}${id}`;
}

function splitCompositeKey(key: string): { pageId: string; id: string } {
  const idx = key.indexOf(COMPOSITE_DELIMITER);
  if (idx < 0) return { pageId: "", id: key };
  return { pageId: key.slice(0, idx), id: key.slice(idx + COMPOSITE_DELIMITER.length) };
}

export type DraftConsoleDiffCounts = { added: number; updated: number; removed: number };

export type DraftConsoleDiffItem = {
  /** Page id this item belongs to. Panel ids are only unique per page,
   * so downstream consumers that key by panel id (e.g. ConsoleGrid's
   * `itemsById` map) must also scope by page id to avoid collisions. */
  pageId: string;
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

function comparableScopedPanels(scoped: ScopedPanel[]): unknown[] {
  return scoped
    .map(({ pageId, panel }) => ({
      pageId,
      id: panel.id ?? "",
      type: panel.type ?? "",
      content: panel.content ?? {},
    }))
    .sort((left, right) => {
      const byPage = left.pageId.localeCompare(right.pageId);
      return byPage !== 0 ? byPage : left.id.localeCompare(right.id);
    });
}

function comparableScopedLayout(scoped: ScopedLayout[]): unknown[] {
  return scoped
    .map(({ pageId, item }) => ({
      pageId,
      i: item.i ?? "",
      x: item.x ?? 0,
      y: item.y ?? 0,
      w: item.w ?? 0,
      h: item.h ?? 0,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    }))
    .sort((left, right) => {
      const byPage = left.pageId.localeCompare(right.pageId);
      return byPage !== 0 ? byPage : left.i.localeCompare(right.i);
    });
}

function comparablePagesMetadata(consoleData?: ConsoleSnapshot): unknown[] {
  // Include page-level metadata (id, name, order) so page-only edits
  // — adding an empty page, renaming a tab, or reordering pages —
  // still register as a diff. Without this the flattened
  // panels/layout arrays are unchanged and the console incorrectly
  // reports "no changes", which caused staged `console.yaml` writes
  // to be silently discarded on save. The normalization step collapses
  // a sole empty default page down to zero pages so a fresh or
  // fully-cleared console does not false-positive against a committed
  // legacy console.
  return normalizedPages(consoleData).map((page, index) => ({
    order: index,
    id: page.id ?? "",
    name: page.name ?? "",
  }));
}

function comparableConsoleSnapshot(consoleData?: ConsoleSnapshot): string {
  return stableStringify({
    pages: comparablePagesMetadata(consoleData),
    panels: comparableScopedPanels(scopedPanels(consoleData)),
    layout: comparableScopedLayout(scopedLayout(consoleData)),
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

function indexScopedPanels(scoped: ScopedPanel[]): Map<string, ConsolePanel> {
  return new Map(scoped.map(({ pageId, panel }) => [compositeKey(pageId, panel.id ?? ""), panel]));
}

function indexScopedLayout(scoped: ScopedLayout[]): Map<string, ConsoleLayoutItem> {
  return new Map(scoped.map(({ pageId, item }) => [compositeKey(pageId, item.i ?? ""), item]));
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

function panelDiffPath(pageId: string, id: string): string {
  const safeId = id || "unknown";
  return pageId ? `console/pages/${pageId}/panels/${safeId}.yaml` : `console/panels/${safeId}.yaml`;
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
  pageId: string,
  id: string,
  panel: ConsolePanel | undefined,
  layout: ConsoleLayoutItem | undefined,
): DraftDiffLine[] {
  const path = panelDiffPath(pageId, id);
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

type UpdatedPanelInput = {
  pageId: string;
  id: string;
  livePanel: ConsolePanel | undefined;
  draftPanel: ConsolePanel | undefined;
  liveLayout: ConsoleLayoutItem | undefined;
  draftLayout: ConsoleLayoutItem | undefined;
};

function buildUpdatedPanelLines(input: UpdatedPanelInput): DraftDiffLine[] {
  const { pageId, id, livePanel, draftPanel, liveLayout, draftLayout } = input;
  const path = panelDiffPath(pageId, id);
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
  const livePanels = indexScopedPanels(scopedPanels(liveConsole));
  const draftPanels = indexScopedPanels(scopedPanels(draftConsole));
  const liveLayout = indexScopedLayout(scopedLayout(liveConsole));
  const draftLayout = indexScopedLayout(scopedLayout(draftConsole));
  const keys = Array.from(
    new Set([...livePanels.keys(), ...draftPanels.keys(), ...liveLayout.keys(), ...draftLayout.keys()]),
  )
    .filter(Boolean)
    .sort((left, right) => left.localeCompare(right));
  const items: DraftConsoleDiffItem[] = [];
  let addedCount = 0;
  let updatedCount = 0;
  let removedCount = 0;

  keys.forEach((key) => {
    const { pageId, id } = splitCompositeKey(key);
    const livePanel = livePanels.get(key);
    const draftPanel = draftPanels.get(key);
    const liveLayoutItem = liveLayout.get(key);
    const draftLayoutItem = draftLayout.get(key);
    const liveExists = !!livePanel || !!liveLayoutItem;
    const draftExists = !!draftPanel || !!draftLayoutItem;

    if (!liveExists && draftExists) {
      items.push({
        pageId,
        id,
        title: panelTitle(draftPanel, id),
        changeType: "added",
        panel: draftPanel,
        layout: draftLayoutItem,
        lines: buildPanelLines("+", pageId, id, draftPanel, draftLayoutItem),
      });
      addedCount += 1;
      return;
    }

    if (liveExists && !draftExists) {
      items.push({
        pageId,
        id,
        title: panelTitle(livePanel, id),
        changeType: "removed",
        panel: livePanel,
        layout: liveLayoutItem,
        lines: buildPanelLines("-", pageId, id, livePanel, liveLayoutItem),
      });
      removedCount += 1;
      return;
    }

    const panelChanged = panelSnapshot(livePanel) !== panelSnapshot(draftPanel);
    const layoutChanged = layoutSnapshot(liveLayoutItem) !== layoutSnapshot(draftLayoutItem);
    if (panelChanged || layoutChanged) {
      items.push({
        pageId,
        id,
        title: panelTitle(draftPanel ?? livePanel, id),
        changeType: "updated",
        panel: draftPanel,
        layout: draftLayoutItem,
        lines: buildUpdatedPanelLines({
          pageId,
          id,
          livePanel,
          draftPanel,
          liveLayout: liveLayoutItem,
          draftLayout: draftLayoutItem,
        }),
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
