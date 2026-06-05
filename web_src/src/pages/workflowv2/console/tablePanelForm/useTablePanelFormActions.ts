import type { SuperplaneComponentsNode } from "@/api-client";
import { draftToPayload, type PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import type { TablePanelContent } from "../panelTypes";
import type {
  WidgetRowAction,
  WidgetRowStyle,
  WidgetSort,
  WidgetTableColumn,
  WidgetTableFilter,
} from "../widget/types";
import { suggestColumnFormat } from "../widget/useMemoryCatalog";
import type { TablePanelPayloadDrafts } from "./useTablePanelPayloadDrafts";

export function useTablePanelFormActions({
  value,
  onChange,
  fields,
  triggerNodes,
  payloadDrafts,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
  fields: Array<{ field: string }>;
  triggerNodes: SuperplaneComponentsNode[];
  payloadDrafts: TablePanelPayloadDrafts;
}) {
  const updateColumn = (idx: number, patch: Partial<WidgetTableColumn>) => {
    const columns = value.render.columns.map((col, i) => (i === idx ? { ...col, ...patch } : col));
    onChange({ ...value, render: { ...value.render, columns } });
  };

  const addColumnFromField = (field: string) => {
    if (!field || value.render.columns.some((c) => c.field === field)) return;
    onChange({
      ...value,
      render: {
        ...value.render,
        columns: [...value.render.columns, { field, label: field, format: suggestColumnFormat(field) }],
      },
    });
  };

  const addAllFields = () => {
    const existing = new Set(value.render.columns.map((c) => c.field));
    const next = [...value.render.columns];
    for (const { field } of fields) {
      if (existing.has(field)) continue;
      next.push({ field, label: field, format: suggestColumnFormat(field) });
    }
    onChange({ ...value, render: { ...value.render, columns: next } });
  };

  const addColumn = () => {
    onChange({
      ...value,
      render: { ...value.render, columns: [...value.render.columns, { field: "", label: "" }] },
    });
  };

  const removeColumn = (idx: number) => {
    onChange({
      ...value,
      render: { ...value.render, columns: value.render.columns.filter((_, i) => i !== idx) },
    });
  };

  const updateFilter = (idx: number, patch: Partial<WidgetTableFilter>) => {
    const where = (value.render.where ?? []).map((f, i) => (i === idx ? { ...f, ...patch } : f));
    onChange({ ...value, render: { ...value.render, where } });
  };

  const addFilter = () => {
    const where = [...(value.render.where ?? []), { field: "", op: "eq" as const, value: "" }];
    onChange({ ...value, render: { ...value.render, where } });
  };

  const removeFilter = (idx: number) => {
    onChange({
      ...value,
      render: { ...value.render, where: (value.render.where ?? []).filter((_, i) => i !== idx) },
    });
  };

  const updateAction = (idx: number, patch: Partial<WidgetRowAction>) => {
    const rowActions = (value.render.rowActions ?? []).map((a, i) =>
      i === idx ? { ...a, ...patch } : a,
    ) as WidgetRowAction[];
    onChange({ ...value, render: { ...value.render, rowActions } });
  };

  const addAction = () => {
    const rowActions: WidgetRowAction[] = [
      ...(value.render.rowActions ?? []),
      { kind: "trigger", label: "Run", node: triggerNodes[0]?.name || triggerNodes[0]?.id || "", hook: "run" },
    ];
    onChange({ ...value, render: { ...value.render, rowActions } });
  };

  const removeAction = (idx: number) => {
    payloadDrafts.dropDraft(idx);
    onChange({
      ...value,
      render: { ...value.render, rowActions: (value.render.rowActions ?? []).filter((_, i) => i !== idx) },
    });
  };

  /**
   * Set or update the table's widget-level sort. Passing a sort whose `field`
   * is blank clears the sort entirely, keeping persisted YAML free of empty
   * `{ field: "" }` stubs.
   */
  const setSort = (nextSort: WidgetSort | undefined) => {
    const trimmedField = nextSort?.field.trim() ?? "";
    if (!nextSort || !trimmedField) {
      const { sort: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    const sort: WidgetSort = { field: nextSort.field };
    if (nextSort.order) sort.order = nextSort.order;
    onChange({ ...value, render: { ...value.render, sort } });
  };

  const rowStyleActions = makeRowStyleActions({ value, onChange });
  const payloadActions = makePayloadActions({ payloadDrafts, updateAction });

  return {
    updateColumn,
    addColumnFromField,
    addAllFields,
    addColumn,
    removeColumn,
    updateFilter,
    addFilter,
    removeFilter,
    updateAction,
    addAction,
    removeAction,
    setSort,
    ...rowStyleActions,
    ...payloadActions,
  };
}

function makeRowStyleActions({
  value,
  onChange,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}) {
  const updateRowStyle = (idx: number, patch: Partial<WidgetRowStyle>) => {
    const rowStyles = (value.render.rowStyles ?? []).map((rule, i) => (i === idx ? { ...rule, ...patch } : rule));
    onChange({ ...value, render: { ...value.render, rowStyles } });
  };

  const addRowStyle = () => {
    const rowStyles: WidgetRowStyle[] = [
      ...(value.render.rowStyles ?? []),
      { field: "", op: "eq", value: "", tone: "red-soft" },
    ];
    onChange({ ...value, render: { ...value.render, rowStyles } });
  };

  const removeRowStyle = (idx: number) => {
    const next = (value.render.rowStyles ?? []).filter((_, i) => i !== idx);
    // Drop the key entirely when emptied so persisted YAML doesn't carry an
    // empty `rowStyles: []` stub.
    if (next.length === 0) {
      const { rowStyles: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    onChange({ ...value, render: { ...value.render, rowStyles: next } });
  };

  return { updateRowStyle, addRowStyle, removeRowStyle };
}

export type TablePanelFormActions = ReturnType<typeof useTablePanelFormActions>;

function makePayloadActions({
  payloadDrafts,
  updateAction,
}: {
  payloadDrafts: TablePanelPayloadDrafts;
  updateAction: (idx: number, patch: Partial<WidgetRowAction>) => void;
}) {
  const commitDraft = (idx: number, entries: PayloadDraftEntry[]) => {
    payloadDrafts.setDraft(idx, entries);
    updateAction(idx, { payload: draftToPayload(entries) });
  };

  const updatePayloadEntry = (actionIdx: number, rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const next = current.map((entry) => (entry.rowId === rowId ? { ...entry, ...patch } : entry));
    commitDraft(actionIdx, next);
  };

  const removePayloadEntry = (actionIdx: number, rowId: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const next = current.filter((entry) => entry.rowId !== rowId);
    commitDraft(actionIdx, next);
  };

  const quickInsertPayloadField = (actionIdx: number, field: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const path = field;
    const template = `{{ ${field} }}`;
    // If the field already exists as a path, leave the existing row alone —
    // the chip click acts as "ensure this field is mapped" and shouldn't
    // overwrite an author-customized value template.
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
