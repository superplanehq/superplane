import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { useDashboardContext } from "./DashboardContext";
import { DataSourceForm } from "./DataSourceForm";
import { MemoryDiscoveryPanel } from "./MemoryDiscoveryPanel";
import type { TablePanelContent } from "./panelTypes";
import { ActionRow, ColumnRow, FilterRow } from "./TablePanelFormRows";
import type { WidgetRowAction, WidgetTableColumn, WidgetTableFilter } from "./widget/types";
import { suggestColumnFormat, useMemoryCatalog } from "./widget/useMemoryCatalog";

interface TablePanelFormProps {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}

export function TablePanelForm({ value, onChange }: TablePanelFormProps) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const triggerNodes = (ctx?.nodes ?? []).filter((n) => n.type === "TYPE_TRIGGER");
  const namespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : "";
  const { fields } = useMemoryCatalog(canvasId, namespace);
  const fieldOptions = fields.map((f) => f.field);
  const actions = useTablePanelFormActions({ value, onChange, fields, triggerNodes });

  return (
    <div className="space-y-4">
      <TitleField value={value} onChange={onChange} />
      <DataSourceForm value={value.dataSource} onChange={(dataSource) => onChange({ ...value, dataSource })} />
      <MemorySourcePicker value={value} canvasId={canvasId} onChange={onChange} />
      <ColumnsSection value={value} fields={fields} fieldOptions={fieldOptions} actions={actions} />
      <FiltersSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <RowActionsSection value={value} triggerNodes={triggerNodes} fieldOptions={fieldOptions} actions={actions} />
    </div>
  );
}

function useTablePanelFormActions({
  value,
  onChange,
  fields,
  triggerNodes,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
  fields: Array<{ field: string }>;
  triggerNodes: Array<{ id?: string; name?: string }>;
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
    onChange({
      ...value,
      render: { ...value.render, rowActions: (value.render.rowActions ?? []).filter((_, i) => i !== idx) },
    });
  };

  const updatePayloadEntry = (actionIdx: number, path: string, template: string) => {
    const action = value.render.rowActions?.[actionIdx];
    if (!action) return;
    const payload = { ...(action.payload ?? {}) };
    if (!template.trim()) {
      delete payload[path];
    } else {
      payload[path] = template;
    }
    updateAction(actionIdx, { payload: Object.keys(payload).length > 0 ? payload : undefined });
  };

  const addPayloadEntry = (actionIdx: number) => {
    const action = value.render.rowActions?.[actionIdx];
    if (!action) return;
    const payload = { ...(action.payload ?? {}), "": "" };
    updateAction(actionIdx, { payload });
  };

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
    updatePayloadEntry,
    addPayloadEntry,
  };
}

type TablePanelFormActions = ReturnType<typeof useTablePanelFormActions>;

function TitleField({ value, onChange }: { value: TablePanelContent; onChange: (next: TablePanelContent) => void }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
      <Input
        value={value.title ?? ""}
        onChange={(e) => onChange({ ...value, title: e.target.value })}
        placeholder="Defaults to panel id"
      />
    </div>
  );
}

function MemorySourcePicker({
  value,
  canvasId,
  onChange,
}: {
  value: TablePanelContent;
  canvasId: string | undefined;
  onChange: (next: TablePanelContent) => void;
}) {
  if (value.dataSource.kind !== "memory") return null;
  const dataSource = value.dataSource;

  return (
    <MemoryDiscoveryPanel
      canvasId={canvasId}
      selectedNamespace={dataSource.namespace}
      onSelectNamespace={(namespace) => onChange({ ...value, dataSource: { ...dataSource, namespace } })}
    />
  );
}

function ColumnsSection({
  value,
  fields,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fields: Array<{ field: string; sample?: string }>;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Columns</Label>
        <div className="flex gap-1">
          {value.dataSource.kind === "memory" && fields.length > 0 ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={actions.addAllFields}
              data-testid="table-add-all-columns"
            >
              Add all fields
            </Button>
          ) : null}
          <Button type="button" size="sm" variant="outline" onClick={actions.addColumn} data-testid="table-add-column">
            Add column
          </Button>
        </div>
      </div>
      <MemoryFieldButtons value={value} fields={fields} onSelect={actions.addColumnFromField} />
      <div className="space-y-2">
        {value.render.columns.map((col, idx) => (
          <ColumnRow
            key={idx}
            col={col}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateColumn(idx, patch)}
            onRemove={() => actions.removeColumn(idx)}
          />
        ))}
        {value.render.columns.length === 0 ? (
          <p className="text-xs text-slate-500">
            Add columns to display memory rows. Use discovered fields or custom paths / CEL.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function MemoryFieldButtons({
  value,
  fields,
  onSelect,
}: {
  value: TablePanelContent;
  fields: Array<{ field: string; sample?: string }>;
  onSelect: (field: string) => void;
}) {
  if (value.dataSource.kind !== "memory" || fields.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1">
      {fields.map((f) => (
        <Button
          key={f.field}
          type="button"
          size="sm"
          variant="secondary"
          className="h-6 text-[10px]"
          onClick={() => onSelect(f.field)}
          title={f.sample ? `e.g. ${f.sample}` : undefined}
        >
          {f.field}
        </Button>
      ))}
    </div>
  );
}

function FiltersSection({
  value,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Filters</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addFilter}>
          Add filter
        </Button>
      </div>
      <div className="space-y-2">
        {(value.render.where ?? []).map((filter, idx) => (
          <FilterRow
            key={idx}
            filter={filter}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateFilter(idx, patch)}
            onRemove={() => actions.removeFilter(idx)}
          />
        ))}
      </div>
    </div>
  );
}

function RowActionsSection({
  value,
  triggerNodes,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  triggerNodes: Array<{ id?: string; name?: string }>;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Row actions</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addAction} data-testid="table-add-action">
          Add action
        </Button>
      </div>
      <p className="text-[11px] text-slate-500">
        Pick the <strong>trigger</strong> that starts your flow (e.g. Start). HTTP Request and other steps run when that
        trigger fires. Payload values support <code className="text-[10px]">{`{{ field }}`}</code> CEL.
      </p>
      <div className="space-y-3">
        {(value.render.rowActions ?? []).map((action, idx) => (
          <ActionRow
            key={idx}
            action={action}
            triggerNodes={triggerNodes}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateAction(idx, patch)}
            onRemove={() => actions.removeAction(idx)}
            onPayloadChange={(path, template) => actions.updatePayloadEntry(idx, path, template)}
            onAddPayloadEntry={() => actions.addPayloadEntry(idx)}
          />
        ))}
      </div>
    </div>
  );
}
