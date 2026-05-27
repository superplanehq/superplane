import { useMemo } from "react";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { SuperplaneComponentsNode } from "@/api-client";

import { getTriggerTemplates } from "./dashboardTriggerParameters";
import type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import { PayloadEditor } from "./TablePanelPayloadEditor";
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

export type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

const COLUMN_FORMATS: WidgetColumnFormat[] = [
  "text",
  "number",
  "percent",
  "date",
  "datetime",
  "relative",
  "duration",
  "status",
  "badge",
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
  sampleRow,
  payloadEntries,
  onChange,
  onRemove,
  onPayloadEntryChange,
  onPayloadEntryRemove,
  onPayloadEntryQuickInsert,
}: {
  action: WidgetRowAction;
  triggerNodes: SuperplaneComponentsNode[];
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  payloadEntries: PayloadDraftEntry[];
  onChange: (patch: Partial<WidgetRowAction>) => void;
  onRemove: () => void;
  onPayloadEntryChange: (rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => void;
  onPayloadEntryRemove: (rowId: string) => void;
  onPayloadEntryQuickInsert: (field: string) => void;
}) {
  const selectedNode = useMemo(() => {
    if (!action.node) return undefined;
    return triggerNodes.find((n) => n.name === action.node || n.id === action.node);
  }, [triggerNodes, action.node]);
  const templates = useMemo(() => getTriggerTemplates(selectedNode), [selectedNode]);

  return (
    <div className="space-y-3 rounded border border-slate-200 p-3">
      <ActionMainFields
        action={action}
        triggerNodes={triggerNodes}
        templates={templates}
        onChange={onChange}
        onRemove={onRemove}
      />
      <ActionConditions action={action} onChange={onChange} />
      <PayloadEditor
        entries={payloadEntries}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        onEntryChange={onPayloadEntryChange}
        onEntryRemove={onPayloadEntryRemove}
        onQuickInsert={onPayloadEntryQuickInsert}
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

const TEMPLATE_CUSTOM = "__custom__";

function templateFieldSelectValue(currentValue: string, matchesKnown: boolean): string {
  if (matchesKnown) return currentValue;
  if (currentValue) return TEMPLATE_CUSTOM;
  return "__default__";
}

function ActionMainFields({
  action,
  triggerNodes,
  templates,
  onChange,
  onRemove,
}: {
  action: WidgetRowAction;
  triggerNodes: SuperplaneComponentsNode[];
  templates: ReturnType<typeof getTriggerTemplates>;
  onChange: (patch: Partial<WidgetRowAction>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-12 gap-2">
        <Input
          className="col-span-3 h-8"
          value={action.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value })}
          placeholder="Label"
        />
        <Select value={action.node || "__none__"} onValueChange={(v) => onChange({ node: v === "__none__" ? "" : v })}>
          <SelectTrigger className="col-span-6 h-8">
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
      <ActionTemplateField action={action} templates={templates} onChange={onChange} />
    </div>
  );
}

function ActionTemplateField({
  action,
  templates,
  onChange,
}: {
  action: WidgetRowAction;
  templates: ReturnType<typeof getTriggerTemplates>;
  onChange: (patch: Partial<WidgetRowAction>) => void;
}) {
  // Hide when the trigger only exposes one template — `buildDashboardTriggerParameters`
  // picks it automatically, so making the author choose adds no value.
  if (templates.length === 1) return null;

  const knownNames = templates.map((t) => t.name);
  const currentValue = action.template ?? "";
  const matchesKnown = currentValue ? knownNames.includes(currentValue) : false;
  const hasTemplates = templates.length > 0;

  if (!hasTemplates) {
    return (
      <div className="space-y-1">
        <Label className="text-[11px] font-medium text-slate-600">
          Start template <span className="font-normal text-slate-400">(optional)</span>
        </Label>
        <Input
          className="h-8 text-xs"
          value={currentValue}
          onChange={(e) => onChange({ template: e.target.value || undefined })}
          placeholder="Template name (when this trigger has multiple templates)"
        />
      </div>
    );
  }

  const selectValue = templateFieldSelectValue(currentValue, matchesKnown);

  return (
    <div className="space-y-1">
      <Label className="text-[11px] font-medium text-slate-600">Start template</Label>
      <div className="grid grid-cols-2 gap-2">
        <Select
          value={selectValue}
          onValueChange={(v) => {
            if (v === "__default__") {
              onChange({ template: undefined });
              return;
            }
            if (v === TEMPLATE_CUSTOM) return;
            onChange({ template: v });
          }}
        >
          <SelectTrigger className="h-8">
            <SelectValue placeholder="Use first template" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__default__">First template (default)</SelectItem>
            {templates.map((t) => (
              <SelectItem key={t.name} value={t.name}>
                {t.name}
              </SelectItem>
            ))}
            <SelectItem value={TEMPLATE_CUSTOM}>Custom…</SelectItem>
          </SelectContent>
        </Select>
        {selectValue === TEMPLATE_CUSTOM ? (
          <Input
            className="h-8 text-xs"
            value={currentValue}
            onChange={(e) => onChange({ template: e.target.value || undefined })}
            placeholder="Custom template name"
          />
        ) : (
          <p className="self-center text-[11px] text-slate-500">
            {templates.length} templates available. Leave default to use the first.
          </p>
        )}
      </div>
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
