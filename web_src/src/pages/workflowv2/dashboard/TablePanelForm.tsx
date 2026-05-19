import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { useDashboardContext } from "./DashboardContext";
import { DataSourceForm } from "./DataSourceForm";
import { MemoryDiscoveryPanel } from "./MemoryDiscoveryPanel";
import type { TablePanelContent } from "./panelTypes";
import {
  WIDGET_FILTER_OPS,
  WIDGET_ROW_ACTION_ICONS,
  WIDGET_ROW_ACTION_VARIANTS,
  type WidgetColumnFormat,
  type WidgetRowAction,
  type WidgetTableColumn,
  type WidgetTableFilter,
} from "./widget/types";
import { suggestColumnFormat, useMemoryCatalog } from "./widget/useMemoryCatalog";

const COLUMN_FORMATS: WidgetColumnFormat[] = [
  "text",
  "number",
  "percent",
  "date",
  "datetime",
  "relative",
  "duration",
  "status",
  "code",
  "link",
];

const CUSTOM_FIELD = "__custom__";

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

  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>

      <DataSourceForm value={value.dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />

      {value.dataSource.kind === "memory" ? (
        <MemoryDiscoveryPanel
          canvasId={canvasId}
          selectedNamespace={value.dataSource.namespace}
          onSelectNamespace={(ns) => {
            if (value.dataSource.kind !== "memory") return;
            onChange({ ...value, dataSource: { ...value.dataSource, namespace: ns } });
          }}
        />
      ) : null}

      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Columns</Label>
          <div className="flex gap-1">
            {value.dataSource.kind === "memory" && fields.length > 0 ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={addAllFields}
                data-testid="table-add-all-columns"
              >
                Add all fields
              </Button>
            ) : null}
            <Button type="button" size="sm" variant="outline" onClick={addColumn} data-testid="table-add-column">
              Add column
            </Button>
          </div>
        </div>
        {value.dataSource.kind === "memory" && fields.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {fields.map((f) => (
              <Button
                key={f.field}
                type="button"
                size="sm"
                variant="secondary"
                className="h-6 text-[10px]"
                onClick={() => addColumnFromField(f.field)}
                title={f.sample ? `e.g. ${f.sample}` : undefined}
              >
                {f.field}
              </Button>
            ))}
          </div>
        ) : null}
        <div className="space-y-2">
          {value.render.columns.map((col, idx) => (
            <ColumnRow
              key={idx}
              col={col}
              fieldOptions={fields.map((f) => f.field)}
              onChange={(patch) => updateColumn(idx, patch)}
              onRemove={() => removeColumn(idx)}
            />
          ))}
          {value.render.columns.length === 0 ? (
            <p className="text-xs text-slate-500">
              Add columns to display memory rows. Use discovered fields or custom paths / CEL.
            </p>
          ) : null}
        </div>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Filters</Label>
          <Button type="button" size="sm" variant="outline" onClick={addFilter}>
            Add filter
          </Button>
        </div>
        <div className="space-y-2">
          {(value.render.where ?? []).map((filter, idx) => (
            <FilterRow
              key={idx}
              filter={filter}
              fieldOptions={fields.map((f) => f.field)}
              onChange={(patch) => updateFilter(idx, patch)}
              onRemove={() => removeFilter(idx)}
            />
          ))}
        </div>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Row actions</Label>
          <Button type="button" size="sm" variant="outline" onClick={addAction} data-testid="table-add-action">
            Add action
          </Button>
        </div>
        <p className="text-[11px] text-slate-500">
          Pick the <strong>trigger</strong> that starts your flow (e.g. Start). HTTP Request and other steps run when
          that trigger fires. Payload values support <code className="text-[10px]">{`{{ field }}`}</code> CEL.
        </p>
        <div className="space-y-3">
          {(value.render.rowActions ?? []).map((action, idx) => (
            <ActionRow
              key={idx}
              action={action}
              triggerNodes={triggerNodes}
              fieldOptions={fields.map((f) => f.field)}
              onChange={(patch) => updateAction(idx, patch)}
              onRemove={() => removeAction(idx)}
              onPayloadChange={(path, template) => updatePayloadEntry(idx, path, template)}
              onAddPayloadEntry={() => addPayloadEntry(idx)}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function ColumnRow({
  col,
  fieldOptions,
  onChange,
  onRemove,
}: {
  col: WidgetTableColumn;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetTableColumn>) => void;
  onRemove: () => void;
}) {
  const selectValue = fieldOptions.includes(col.field) ? col.field : col.field ? CUSTOM_FIELD : "";
  return (
    <div className="grid grid-cols-12 items-center gap-2 rounded border border-slate-200 p-2">
      {selectValue === CUSTOM_FIELD || fieldOptions.length === 0 ? (
        <Input
          className="col-span-4 h-8"
          value={col.field}
          onChange={(e) => onChange({ field: e.target.value })}
          placeholder="field or {{ expr }}"
        />
      ) : (
        <Select
          value={selectValue || CUSTOM_FIELD}
          onValueChange={(v) => {
            if (v === CUSTOM_FIELD) return;
            onChange({ field: v, label: col.label || v, format: col.format ?? suggestColumnFormat(v) });
          }}
        >
          <SelectTrigger className="col-span-4 h-8">
            <SelectValue placeholder="Field" />
          </SelectTrigger>
          <SelectContent>
            {fieldOptions.map((f) => (
              <SelectItem key={f} value={f}>
                {f}
              </SelectItem>
            ))}
            <SelectItem value={CUSTOM_FIELD}>Custom…</SelectItem>
          </SelectContent>
        </Select>
      )}
      <Input
        className="col-span-3 h-8"
        value={col.label ?? ""}
        onChange={(e) => onChange({ label: e.target.value })}
        placeholder="Header"
      />
      <Select
        value={col.format ?? "__none__"}
        onValueChange={(v) => onChange({ format: v === "__none__" ? undefined : (v as WidgetColumnFormat) })}
      >
        <SelectTrigger className="col-span-4 h-8">
          <SelectValue placeholder="Format" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__none__">Default</SelectItem>
          {COLUMN_FORMATS.map((fmt) => (
            <SelectItem key={fmt} value={fmt}>
              {fmt}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button type="button" size="icon" variant="ghost" className="col-span-1 h-8 w-8" onClick={onRemove}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

function FilterRow({
  filter,
  fieldOptions,
  onChange,
  onRemove,
}: {
  filter: WidgetTableFilter;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetTableFilter>) => void;
  onRemove: () => void;
}) {
  const needsValue = filter.op !== "exists" && filter.op !== "not_exists";
  return (
    <div className="grid grid-cols-12 gap-2 rounded border border-slate-200 p-2">
      <Input
        className="col-span-4 h-8"
        value={filter.field}
        onChange={(e) => onChange({ field: e.target.value })}
        placeholder="field"
        list={fieldOptions.length > 0 ? "table-field-options" : undefined}
      />
      <Select value={filter.op} onValueChange={(v) => onChange({ op: v as WidgetTableFilter["op"] })}>
        <SelectTrigger className="col-span-3 h-8">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_FILTER_OPS.map((op) => (
            <SelectItem key={op} value={op}>
              {op}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {needsValue ? (
        <Input
          className="col-span-4 h-8"
          value={filter.value ?? ""}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="value or {{ expr }}"
        />
      ) : (
        <div className="col-span-4" />
      )}
      <Button type="button" size="icon" variant="ghost" className="col-span-1 h-8 w-8" onClick={onRemove}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

function ActionRow({
  action,
  triggerNodes,
  fieldOptions,
  onChange,
  onRemove,
  onPayloadChange,
  onAddPayloadEntry,
}: {
  action: WidgetRowAction;
  triggerNodes: Array<{ id?: string; name?: string }>;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetRowAction>) => void;
  onRemove: () => void;
  onPayloadChange: (path: string, template: string) => void;
  onAddPayloadEntry: () => void;
}) {
  const payloadEntries = Object.entries(action.payload ?? {});

  return (
    <div className="space-y-2 rounded border border-slate-200 p-2">
      <div className="grid grid-cols-12 gap-2">
        <Input
          className="col-span-3 h-8"
          value={action.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value })}
          placeholder="Label"
        />
        <Select value={action.node || "__none__"} onValueChange={(v) => onChange({ node: v === "__none__" ? "" : v })}>
          <SelectTrigger className="col-span-4 h-8">
            <SelectValue placeholder="Trigger node" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Select trigger…</SelectItem>
            {triggerNodes.map((n) => {
              const id = n.name || n.id || "";
              return (
                <SelectItem key={id} value={id}>
                  {n.name || n.id}
                </SelectItem>
              );
            })}
          </SelectContent>
        </Select>
        <Input
          className="col-span-2 h-8"
          value={action.template ?? ""}
          onChange={(e) => onChange({ template: e.target.value || undefined })}
          placeholder="Template"
        />
        <Select
          value={action.variant ?? "default"}
          onValueChange={(v) => onChange({ variant: v as WidgetRowAction["variant"] })}
        >
          <SelectTrigger className="col-span-2 h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_ROW_ACTION_VARIANTS.map((v) => (
              <SelectItem key={v} value={v}>
                {v}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" size="icon" variant="ghost" className="col-span-1 h-8 w-8" onClick={onRemove}>
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Input
          value={action.show ?? ""}
          onChange={(e) => onChange({ show: e.target.value || undefined })}
          placeholder='Show when (status == "running" or {{ expr }})'
          className="h-8 text-xs"
        />
        <Input
          value={action.confirm ?? ""}
          onChange={(e) => onChange({ confirm: e.target.value || undefined })}
          placeholder='Confirm ("Destroy #{{ pr_number }}?")'
          className="h-8 text-xs"
        />
      </div>
      <div className="space-y-1">
        <div className="flex items-center justify-between">
          <span className="text-[11px] font-medium text-slate-600">Payload fields</span>
          <Button type="button" size="sm" variant="ghost" className="h-6 text-[10px]" onClick={onAddPayloadEntry}>
            <Plus className="mr-0.5 h-3 w-3" />
            Add
          </Button>
        </div>
        {payloadEntries.map(([path, template]) => (
          <div key={path} className="grid grid-cols-2 gap-1">
            <Input
              value={path}
              onChange={(e) => {
                const next = e.target.value;
                onPayloadChange(path, "");
                if (next) onPayloadChange(next, template);
              }}
              placeholder="data.issue.number"
              className="h-7 text-xs"
              list={fieldOptions.length > 0 ? "table-field-options" : undefined}
            />
            <Input
              value={template}
              onChange={(e) => onPayloadChange(path, e.target.value)}
              placeholder="{{ pr_number }}"
              className="h-7 text-xs"
            />
          </div>
        ))}
      </div>
      <Select
        value={action.icon ?? "__none__"}
        onValueChange={(v) => onChange({ icon: v === "__none__" ? undefined : (v as WidgetRowAction["icon"]) })}
      >
        <SelectTrigger className="h-8 w-40">
          <SelectValue placeholder="Icon" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__none__">No icon</SelectItem>
          {WIDGET_ROW_ACTION_ICONS.map((icon) => (
            <SelectItem key={icon} value={icon}>
              {icon}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {fieldOptions.length > 0 ? (
        <datalist id="table-field-options">
          {fieldOptions.map((f) => (
            <option key={f} value={f} />
          ))}
        </datalist>
      ) : null}
    </div>
  );
}
