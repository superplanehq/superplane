import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { DataSourceForm } from "./DataSourceForm";
import { useConsoleContext } from "./ConsoleContext";
import { NUMBER_PANEL_AGGREGATIONS, NUMBER_PANEL_FORMATS } from "./numberPanelFormConstants";
import type { ScorecardPanelContent, TablePanelDataSource } from "./panelTypes";
import type {
  WidgetColumnFormat,
  WidgetNumberAggregation,
  WidgetScorecardRender,
  WidgetScorecardShowChange,
  WidgetTrendBetter,
} from "./widget/types";
import { WIDGET_SCORECARD_SHOW_CHANGES } from "./widget/types";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

const SHOW_CHANGE_LABELS: Record<WidgetScorecardShowChange, string> = {
  percent: "Percent",
  number: "Number",
  both: "Both",
  none: "None",
};

const BETTER_LABELS: Record<WidgetTrendBetter, string> = {
  up: "Higher is better",
  down: "Lower is better",
};

/**
 * Runs and executions come from the API newest-first, so the persisted
 * `first` aggregation actually picks the *latest* record, and `last`
 * picks the earliest. Memory rows land in insertion order (oldest-first),
 * so the mapping flips. Persisted YAML always stores `first`/`last`;
 * this map only affects the labels rendered in the form.
 */
function aggregationLabel(
  aggregation: WidgetNumberAggregation,
  sourceKind: TablePanelDataSource["kind"] | undefined,
): string {
  const newestFirst = sourceKind === "runs" || sourceKind === "executions";
  if (aggregation === "first") return newestFirst ? "Latest" : "Earliest";
  if (aggregation === "last") return newestFirst ? "Earliest" : "Latest";
  return aggregation;
}

interface ScorecardPanelFormProps {
  value: ScorecardPanelContent;
  onChange: (next: ScorecardPanelContent) => void;
}

export function ScorecardPanelForm({ value, onChange }: ScorecardPanelFormProps) {
  const render = value.render ?? {
    kind: "scorecard" as const,
    aggregation: "last" as WidgetNumberAggregation,
  };
  const updateRender = (patch: Partial<WidgetScorecardRender>) =>
    onChange({ ...value, render: { ...render, ...patch } });

  return (
    <div className="space-y-3">
      <TitleField value={value} onChange={onChange} />
      <DataSourceForm value={value.dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
      <AggregationFields value={value} render={render} onChange={updateRender} />
      <FormatLabelRow render={render} onChange={updateRender} />
      <PrefixSuffixRow render={render} onChange={updateRender} />
      <SeriesFields render={render} onChange={updateRender} />
      <StatusFields render={render} onChange={updateRender} />
      <ChangeFields render={render} onChange={updateRender} />
      <TargetFields render={render} onChange={updateRender} />
    </div>
  );
}

function TitleField({
  value,
  onChange,
}: {
  value: ScorecardPanelContent;
  onChange: (next: ScorecardPanelContent) => void;
}) {
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

function AggregationFields({
  value,
  render,
  onChange,
}: {
  value: ScorecardPanelContent;
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const memoryNamespace =
    value.dataSource && value.dataSource.kind === "memory" ? value.dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const fieldListId = memoryNamespace ? `scorecard-fields-${memoryNamespace}` : undefined;
  const aggregation = render.aggregation ?? "last";
  const aggregationNeedsField = aggregation !== "count";
  const sourceKind = value.dataSource?.kind;

  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Aggregation</Label>
        <Select value={aggregation} onValueChange={(v) => onChange({ aggregation: v as WidgetNumberAggregation })}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {NUMBER_PANEL_AGGREGATIONS.map((a) => (
              <SelectItem key={a} value={a}>
                {aggregationLabel(a, sourceKind)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {aggregationNeedsField ? (
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">Field</Label>
          <Input
            list={fields.length > 0 && fieldListId ? fieldListId : undefined}
            value={render.field ?? ""}
            onChange={(e) => onChange({ field: e.target.value || undefined })}
            placeholder="e.g. openCount"
            data-testid="scorecard-field"
          />
          {fields.length > 0 && fieldListId ? (
            <datalist id={fieldListId}>
              {fields.map((f) => (
                <option key={f.field} value={f.field} />
              ))}
            </datalist>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function FormatLabelRow({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Format</Label>
        <Select
          value={render.format ?? "__none__"}
          onValueChange={(v) => onChange({ format: v === "__none__" ? undefined : (v as WidgetColumnFormat) })}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Default" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Default</SelectItem>
            {NUMBER_PANEL_FORMATS.map((f) => (
              <SelectItem key={f} value={f}>
                {f}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Label (optional)</Label>
        <Input
          value={render.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value || undefined })}
          placeholder="e.g. Open UX papercuts"
        />
      </div>
    </div>
  );
}

function PrefixSuffixRow({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Prefix (optional)</Label>
        <Input
          value={render.prefix ?? ""}
          onChange={(e) => onChange({ prefix: e.target.value || undefined })}
          placeholder="e.g. R$"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Suffix (optional)</Label>
        <Input
          value={render.suffix ?? ""}
          onChange={(e) => onChange({ suffix: e.target.value || undefined })}
          placeholder="e.g. MWh"
        />
      </div>
    </div>
  );
}

function SeriesFields({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Series field (optional)</Label>
      <Input
        value={render.sparklineField ?? ""}
        onChange={(e) => onChange({ sparklineField: e.target.value || undefined })}
        placeholder="e.g. openCount"
        data-testid="scorecard-sparkline-field"
      />
      <p className="text-[11px] text-slate-500 dark:text-gray-400">
        Draws the sparkline. When empty, the change chip still renders using the primary field.
      </p>
    </div>
  );
}

function StatusFields({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  const better = render.better ?? "up";
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Status direction</Label>
      <Select value={better} onValueChange={(v) => onChange({ better: v as WidgetTrendBetter })}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="up">{BETTER_LABELS.up}</SelectItem>
          <SelectItem value="down">{BETTER_LABELS.down}</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
}

function ChangeFields({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  const showChange = render.showChange ?? "both";
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Show change as</Label>
        <Select value={showChange} onValueChange={(v) => onChange({ showChange: v as WidgetScorecardShowChange })}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_SCORECARD_SHOW_CHANGES.map((option) => (
              <SelectItem key={option} value={option}>
                {SHOW_CHANGE_LABELS[option]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Change caption (optional)</Label>
        <Input
          value={render.changeCaption ?? ""}
          onChange={(e) => onChange({ changeCaption: e.target.value || undefined })}
          placeholder="e.g. vs previous"
        />
      </div>
    </div>
  );
}

function TargetFields({
  render,
  onChange,
}: {
  render: WidgetScorecardRender;
  onChange: (patch: Partial<WidgetScorecardRender>) => void;
}) {
  const showProgress = render.showProgress ?? false;
  return (
    <div className="space-y-2 rounded-lg bg-slate-100 p-3 dark:bg-gray-800">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-300">Target (optional)</Label>
        <Input
          value={render.target ?? ""}
          onChange={(e) => onChange({ target: e.target.value || undefined })}
          placeholder="e.g. 80 or {{ goal }}"
          data-testid="scorecard-target"
        />
        <p className="text-[11px] text-slate-500 dark:text-gray-400">
          Literal number or a full{" "}
          <code className="rounded bg-slate-200 px-1 py-0.5 text-[10px] dark:bg-gray-700">{"{{ CEL }}"}</code>{" "}
          expression.
        </p>
      </div>
      <label className="flex items-center gap-2 text-xs text-slate-700 dark:text-gray-300">
        <Checkbox
          checked={showProgress}
          onChange={(e) => onChange({ showProgress: e.target.checked ? true : undefined })}
          data-testid="scorecard-show-progress"
        />
        Show progress bar toward target
      </label>
    </div>
  );
}
