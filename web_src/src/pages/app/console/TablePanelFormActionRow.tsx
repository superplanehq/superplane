import { useMemo } from "react";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import { getTriggerTemplates } from "./consoleTriggerParameters";
import { ConsoleExpressionEditor } from "./ConsoleExpressionEditor";
import { PayloadEditor } from "./TablePanelPayloadEditor";
import { WIDGET_ROW_ACTION_ICONS, WIDGET_ROW_ACTION_VARIANTS, type WidgetRowAction } from "./widget/types";

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
    <div className="space-y-3 rounded-lg bg-slate-100 p-3">
      <ActionMainFields
        action={action}
        triggerNodes={triggerNodes}
        templates={templates}
        onChange={onChange}
        onRemove={onRemove}
      />
      <ActionConditions action={action} sampleRow={sampleRow} onChange={onChange} />
      <PayloadEditor
        entries={payloadEntries}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        onEntryChange={onPayloadEntryChange}
        onEntryRemove={onPayloadEntryRemove}
        onQuickInsert={onPayloadEntryQuickInsert}
      />
      <ActionIconSelect action={action} onChange={onChange} />
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
      <div className="flex gap-2">
        <div className="grid min-w-0 flex-1 grid-cols-11 items-center gap-2">
          <Input
            className="col-span-3 h-8"
            value={action.label ?? ""}
            onChange={(e) => onChange({ label: e.target.value })}
            placeholder="Label"
          />
          <Select
            value={action.node || "__none__"}
            onValueChange={(v) => onChange({ node: v === "__none__" ? "" : v })}
          >
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
        </div>
        <div className="flex shrink-0 items-start justify-end">
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600"
            onClick={onRemove}
            aria-label="Remove row action"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
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
  // Hide when the trigger only exposes one template — `buildConsoleTriggerParameters`
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
  sampleRow,
  onChange,
}: {
  action: WidgetRowAction;
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<WidgetRowAction>) => void;
}) {
  return (
    <div className="grid grid-cols-2 items-start gap-2">
      <ConsoleExpressionEditor
        // Runtime for the row-action "show" field routes bare input through
        // the legacy `evaluateShow` parser (not CEL), so keep preview scoped
        // to `{{ … }}` expressions where semantics match.
        syntaxProfile="singleWrapped"
        exampleObj={sampleRow}
        value={action.show ?? ""}
        onChange={(next) => onChange({ show: next || undefined })}
        placeholder='Show when (status == "running" or {{ expr }})'
        quickTip="Tip: bare conditions run at render time; use one full `{{ … }}` CEL expression for preview."
        inputSize="md"
        showValuePreview
      />
      <ConsoleExpressionEditor
        syntaxProfile="wrapped"
        exampleObj={sampleRow}
        value={action.confirm ?? ""}
        onChange={(next) => onChange({ confirm: next || undefined })}
        placeholder='Confirm ("Destroy #{{ pr_number }}?")'
        inputSize="md"
        showValuePreview
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
