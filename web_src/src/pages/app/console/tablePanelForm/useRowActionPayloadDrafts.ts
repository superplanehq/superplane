import { useRef } from "react";

import {
  draftToPayload,
  draftToPayloadShape,
  payloadShallowEqual,
  type PayloadDraftEntry,
} from "@/lib/tablePanelPayloadDraft";
import type { SuperplaneComponentsNode } from "@/api-client";

import type { WidgetRowAction, WidgetSort, WidgetTableFilter } from "../widget/types";

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

export function makeRowActionPayloadActions({
  payloadDrafts,
  updateAction,
}: {
  payloadDrafts: RowActionPayloadDrafts;
  updateAction: (idx: number, patch: Partial<WidgetRowAction>) => void;
}) {
  const commitDraft = (idx: number, entries: PayloadDraftEntry[]) => {
    payloadDrafts.setDraft(idx, entries);
    updateAction(idx, { payload: draftToPayload(entries) });
  };

  const updatePayloadEntry = (actionIdx: number, rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    commitDraft(
      actionIdx,
      current.map((entry) => (entry.rowId === rowId ? { ...entry, ...patch } : entry)),
    );
  };

  const removePayloadEntry = (actionIdx: number, rowId: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    commitDraft(
      actionIdx,
      current.filter((entry) => entry.rowId !== rowId),
    );
  };

  const quickInsertPayloadField = (actionIdx: number, field: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current || current.some((entry) => entry.path === field)) return;
    const trailing = current[current.length - 1];
    const next = [...current];
    const entry = { path: field, template: `{{ ${field} }}` };
    if (trailing && !trailing.path && !trailing.template) {
      next[next.length - 1] = { ...trailing, ...entry };
    } else {
      next.push({ rowId: payloadDrafts.newRowId(), ...entry });
    }
    commitDraft(actionIdx, next);
  };

  return { updatePayloadEntry, removePayloadEntry, quickInsertPayloadField };
}

interface RowPanelRender {
  where?: WidgetTableFilter[];
  sort?: WidgetSort;
  rowActions?: WidgetRowAction[];
}

export function makeRowPanelFormActions<R extends RowPanelRender>({
  render,
  onRenderChange,
  triggerNodes,
  payloadDrafts,
}: {
  render: R;
  onRenderChange: (next: R) => void;
  triggerNodes: SuperplaneComponentsNode[];
  payloadDrafts: RowActionPayloadDrafts;
}) {
  const setWhere = (where: WidgetTableFilter[]) => {
    const next = { ...render };
    if (where.length > 0) next.where = where;
    else delete next.where;
    onRenderChange(next);
  };
  const updateFilter = (idx: number, patch: Partial<WidgetTableFilter>) =>
    setWhere((render.where ?? []).map((filter, i) => (i === idx ? { ...filter, ...patch } : filter)));
  const addFilter = () => setWhere([...(render.where ?? []), { field: "", op: "eq", value: "" }]);
  const removeFilter = (idx: number) => setWhere((render.where ?? []).filter((_, i) => i !== idx));

  const setSort = (sort: WidgetSort | undefined) => {
    const next = { ...render };
    if (!sort?.field.trim()) delete next.sort;
    else next.sort = sort.order ? { field: sort.field, order: sort.order } : { field: sort.field };
    onRenderChange(next);
  };

  const setActions = (rowActions: WidgetRowAction[]) => {
    const next = { ...render };
    if (rowActions.length > 0) next.rowActions = rowActions;
    else delete next.rowActions;
    onRenderChange(next);
  };
  const updateAction = (idx: number, patch: Partial<WidgetRowAction>) =>
    setActions((render.rowActions ?? []).map((action, i) => (i === idx ? { ...action, ...patch } : action)));
  const addAction = () => {
    const trigger = triggerNodes[0];
    setActions([
      ...(render.rowActions ?? []),
      { kind: "trigger", label: "Run", node: trigger?.name || trigger?.id || "", hook: "run" },
    ]);
  };
  const removeAction = (idx: number) => {
    payloadDrafts.dropDraft(idx);
    setActions((render.rowActions ?? []).filter((_, i) => i !== idx));
  };

  return {
    updateFilter,
    addFilter,
    removeFilter,
    setSort,
    updateAction,
    addAction,
    removeAction,
    ...makeRowActionPayloadActions({ payloadDrafts, updateAction }),
  };
}
