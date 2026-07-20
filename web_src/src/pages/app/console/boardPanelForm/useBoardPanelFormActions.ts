import type { SuperplaneComponentsNode } from "@/api-client";
import { draftToPayload, type PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import type { BoardPanelContent } from "../boardPanelContent";
import type { RowActionPayloadDrafts } from "../tablePanelForm/useRowActionPayloadDrafts";
import type {
  WidgetBoardLane,
  WidgetRowAction,
  WidgetSort,
  WidgetTableColumn,
  WidgetTableFilter,
} from "../widget/types";
import { suggestColumnFormat } from "../widget/useMemoryCatalog";

/**
 * Actions surface for {@link BoardPanelForm}. Mirrors
 * `useTablePanelFormActions` in shape but scoped to `BoardPanelContent`
 * (lanes, card, row actions, filters, sort). Keeping this a hook means
 * the row-action payload drafts stay stable across renders even as the
 * user edits list items.
 */
export function useBoardPanelFormActions({
  value,
  onChange,
  triggerNodes,
  payloadDrafts,
}: {
  value: BoardPanelContent;
  onChange: (next: BoardPanelContent) => void;
  triggerNodes: SuperplaneComponentsNode[];
  payloadDrafts: RowActionPayloadDrafts;
}) {
  const laneActions = makeLaneActions(value, onChange);
  const cardActions = makeCardActions(value, onChange);
  const filterActions = makeFilterActions(value, onChange);
  const rowActionActions = makeRowActionActions(value, onChange, triggerNodes, payloadDrafts);
  const setSort = makeSetSort(value, onChange);

  return {
    ...laneActions,
    ...cardActions,
    ...filterActions,
    setSort,
    ...rowActionActions,
  };
}

export type BoardPanelFormActions = ReturnType<typeof useBoardPanelFormActions>;

function makeLaneActions(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  const setLanes = (lanes: WidgetBoardLane[]) => onChange({ ...value, render: { ...value.render, lanes } });
  return {
    addLane: () => setLanes([...value.render.lanes, { value: "" }]),
    updateLane: (idx: number, patch: Partial<WidgetBoardLane>) =>
      setLanes(value.render.lanes.map((lane, i) => (i === idx ? { ...lane, ...patch } : lane))),
    removeLane: (idx: number) => setLanes(value.render.lanes.filter((_, i) => i !== idx)),
  };
}

function makeCardActions(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  const setCardFields = (fields: WidgetTableColumn[] | undefined) => {
    const card = { ...value.render.card };
    if (fields && fields.length > 0) {
      card.fields = fields;
    } else {
      delete card.fields;
    }
    onChange({ ...value, render: { ...value.render, card } });
  };
  const currentFields = () => value.render.card.fields ?? [];
  return {
    addCardField: () => setCardFields([...currentFields(), { field: "", label: "" }]),
    updateCardField: (idx: number, patch: Partial<WidgetTableColumn>) => {
      const next = currentFields().map((col, i) => (i === idx ? applyCardFieldPatch(col, patch) : col));
      setCardFields(next);
    },
    removeCardField: (idx: number) => setCardFields(currentFields().filter((_, i) => i !== idx)),
  };
}

function applyCardFieldPatch(col: WidgetTableColumn, patch: Partial<WidgetTableColumn>): WidgetTableColumn {
  const next = { ...col, ...patch };
  // Suggest a sensible format when the user picks a new field and hasn't
  // customized the format yet. Mirrors the ColumnRow onChange behavior.
  if (patch.field && !next.format) next.format = suggestColumnFormat(patch.field);
  return next;
}

function makeFilterActions(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  const setWhere = (where: WidgetTableFilter[] | undefined) => {
    const nextRender = { ...value.render };
    if (where && where.length > 0) {
      nextRender.where = where;
    } else {
      delete nextRender.where;
    }
    onChange({ ...value, render: nextRender });
  };
  const current = () => value.render.where ?? [];
  return {
    addFilter: () => setWhere([...current(), { field: "", op: "eq" as const, value: "" }]),
    updateFilter: (idx: number, patch: Partial<WidgetTableFilter>) =>
      setWhere(current().map((f, i) => (i === idx ? { ...f, ...patch } : f))),
    removeFilter: (idx: number) => setWhere(current().filter((_, i) => i !== idx)),
  };
}

function makeSetSort(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  return (nextSort: WidgetSort | undefined) => {
    const trimmedField = nextSort?.field.trim() ?? "";
    if (!nextSort || !trimmedField) {
      const nextRender = { ...value.render };
      delete nextRender.sort;
      onChange({ ...value, render: nextRender });
      return;
    }
    const sort: WidgetSort = { field: nextSort.field };
    if (nextSort.order) sort.order = nextSort.order;
    onChange({ ...value, render: { ...value.render, sort } });
  };
}

function makeRowActionActions(
  value: BoardPanelContent,
  onChange: (next: BoardPanelContent) => void,
  triggerNodes: SuperplaneComponentsNode[],
  payloadDrafts: RowActionPayloadDrafts,
) {
  const setActions = (rowActions: WidgetRowAction[] | undefined) => {
    const nextRender = { ...value.render };
    if (rowActions && rowActions.length > 0) {
      nextRender.rowActions = rowActions;
    } else {
      delete nextRender.rowActions;
    }
    onChange({ ...value, render: nextRender });
  };
  const current = () => value.render.rowActions ?? [];

  const updateAction = (idx: number, patch: Partial<WidgetRowAction>) => {
    const nextActions = current().map((a, i) => (i === idx ? { ...a, ...patch } : a)) as WidgetRowAction[];
    setActions(nextActions);
  };

  const addAction = () => {
    const first = triggerNodes[0];
    const node = first?.name || first?.id || "";
    setActions([...current(), { kind: "trigger", label: "Run", node, hook: "run" }]);
  };

  const removeAction = (idx: number) => {
    payloadDrafts.dropDraft(idx);
    setActions(current().filter((_, i) => i !== idx));
  };

  return {
    updateAction,
    addAction,
    removeAction,
    ...makePayloadActions({ payloadDrafts, updateAction }),
  };
}

function makePayloadActions({
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
    if (!current) return;
    const path = field;
    const template = `{{ ${field} }}`;
    if (current.some((entry) => entry.path === path)) return;
    const trailing = current[current.length - 1];
    const next = [...current];
    if (trailing && !trailing.path && !trailing.template) {
      next[next.length - 1] = { ...trailing, path, template };
    } else {
      next.push({ rowId: payloadDrafts.newRowId(), path, template });
    }
    commitDraft(actionIdx, next);
  };

  return { updatePayloadEntry, removePayloadEntry, quickInsertPayloadField };
}
