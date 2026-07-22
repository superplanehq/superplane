import type { SuperplaneComponentsNode } from "@/api-client";

import type { TablePanelContent } from "../panelTypes";
import type { WidgetRowStyle, WidgetTableColumn } from "../widget/types";
import { suggestColumnFormat } from "../widget/useMemoryCatalog";
import { makeRowPanelFormActions, type RowActionPayloadDrafts } from "./useRowActionPayloadDrafts";

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
  payloadDrafts: RowActionPayloadDrafts;
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

  const rowStyleActions = makeRowStyleActions({ value, onChange });
  const rowPanelActions = makeRowPanelFormActions({
    render: value.render,
    onRenderChange: (render) => onChange({ ...value, render }),
    triggerNodes,
    payloadDrafts,
  });

  return {
    updateColumn,
    addColumnFromField,
    addAllFields,
    addColumn,
    removeColumn,
    ...rowStyleActions,
    ...rowPanelActions,
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
