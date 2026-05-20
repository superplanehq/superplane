import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import {
  WIDGET_FILTER_OPS,
  WIDGET_ROW_ACTION_ICONS,
  WIDGET_ROW_ACTION_VARIANTS,
  type WidgetColumnFormat,
  type WidgetRowAction,
  type WidgetTableColumn,
  type WidgetTableFilter,
} from "./widget/types";
import { suggestColumnFormat } from "./widget/useMemoryCatalog";

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

export function ColumnRow({
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

export function FilterRow({
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

export function ActionRow({
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
  return (
    <div className="space-y-2 rounded border border-slate-200 p-2">
      <ActionMainFields action={action} triggerNodes={triggerNodes} onChange={onChange} onRemove={onRemove} />
      <ActionConditions action={action} onChange={onChange} />
      <PayloadEditor
        payload={action.payload}
        fieldOptions={fieldOptions}
        onPayloadChange={onPayloadChange}
        onAddPayloadEntry={onAddPayloadEntry}
      />
      <ActionIconSelect action={action} onChange={onChange} />
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

function ActionMainFields({
  action,
  triggerNodes,
  onChange,
  onRemove,
}: {
  action: WidgetRowAction;
  triggerNodes: Array<{ id?: string; name?: string }>;
  onChange: (patch: Partial<WidgetRowAction>) => void;
  onRemove: () => void;
}) {
  return (
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
  );
}

function ActionConditions({
  action,
  onChange,
}: {
  action: WidgetRowAction;
  onChange: (patch: Partial<WidgetRowAction>) => void;
}) {
  return (
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
  );
}

function PayloadEditor({
  payload,
  fieldOptions,
  onPayloadChange,
  onAddPayloadEntry,
}: {
  payload: WidgetRowAction["payload"];
  fieldOptions: string[];
  onPayloadChange: (path: string, template: string) => void;
  onAddPayloadEntry: () => void;
}) {
  const payloadEntries = Object.entries(payload ?? {});

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-medium text-slate-600">Payload fields</span>
        <Button type="button" size="sm" variant="ghost" className="h-6 text-[10px]" onClick={onAddPayloadEntry}>
          <Plus className="mr-0.5 h-3 w-3" />
          Add
        </Button>
      </div>
      {payloadEntries.map(([path, template]) => (
        <PayloadEntry
          key={path}
          path={path}
          template={template}
          fieldOptions={fieldOptions}
          onPayloadChange={onPayloadChange}
        />
      ))}
    </div>
  );
}

function PayloadEntry({
  path,
  template,
  fieldOptions,
  onPayloadChange,
}: {
  path: string;
  template: string;
  fieldOptions: string[];
  onPayloadChange: (path: string, template: string) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-1">
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
  );
}

function ActionIconSelect({
  action,
  onChange,
}: {
  action: WidgetRowAction;
  onChange: (patch: Partial<WidgetRowAction>) => void;
}) {
  return (
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
  );
}
