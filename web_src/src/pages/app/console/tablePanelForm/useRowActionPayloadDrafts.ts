import { useRef } from "react";

import { draftToPayloadShape, payloadShallowEqual, type PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import type { WidgetRowAction } from "../widget/types";

/**
 * Track stable per-action payload row ids so React reconciles inputs by id
 * rather than by `path`. Shared by any panel form that renders the
 * `ActionRow` payload editor over a `WidgetRowAction[]` — currently the
 * table and board panels.
 *
 * Re-keying by path caused focus loss + duplicate rows whenever the user
 * edited a path (tab/blur). Drafts are the authoritative ordered row list
 * during editing; blank rows and rows with an empty path are kept in the
 * UI but stripped before save. We only re-seed drafts from the persisted
 * payload when the value changes externally (e.g. YAML tab edit or undo),
 * detected by comparing what the draft would persist against what is
 * actually persisted.
 */
export function useRowActionPayloadDrafts(rowActions: WidgetRowAction[] | undefined) {
  const drafts = useRef<Map<number, PayloadDraftEntry[]>>(new Map());
  const counter = useRef(0);
  const newRowId = () => `row-${counter.current++}`;

  const seedFromPayload = (payload: Record<string, string> | undefined): PayloadDraftEntry[] => {
    const seeded: PayloadDraftEntry[] = [];
    for (const [path, template] of Object.entries(payload ?? {})) {
      seeded.push({ rowId: newRowId(), path, template });
    }
    return seeded;
  };

  const ensureTrailingBlank = (entries: PayloadDraftEntry[]): PayloadDraftEntry[] => {
    const last = entries[entries.length - 1];
    if (last && !last.path && !last.template) return entries;
    return [...entries, { rowId: newRowId(), path: "", template: "" }];
  };

  const snapshots: PayloadDraftEntry[][] = (rowActions ?? []).map((action, idx) => {
    let entries = drafts.current.get(idx);
    if (!entries) {
      entries = seedFromPayload(action.payload);
    } else {
      const wouldPersist = draftToPayloadShape(entries);
      const persisted = action.payload ?? {};
      if (!payloadShallowEqual(wouldPersist, persisted)) {
        entries = seedFromPayload(action.payload);
      }
    }
    entries = ensureTrailingBlank(entries);
    drafts.current.set(idx, entries);
    return entries;
  });

  return {
    snapshots,
    newRowId,
    setDraft: (idx: number, entries: PayloadDraftEntry[]) => {
      drafts.current.set(idx, entries);
    },
    dropDraft: (idx: number) => {
      const shifted = new Map<number, PayloadDraftEntry[]>();
      for (const [key, entries] of drafts.current) {
        if (key === idx) continue;
        shifted.set(key > idx ? key - 1 : key, entries);
      }
      drafts.current = shifted;
    },
  };
}

export type RowActionPayloadDrafts = ReturnType<typeof useRowActionPayloadDrafts>;
