import type { ReactNode } from "react";
import { Plus, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/ui/switch";

import {
  SCORECARD_AGGREGATIONS,
  SCORECARD_FORMATS,
  scorecardHasGoalLine,
  type ScorecardPanelContent,
  type ScorecardStatusMode,
} from "./scorecardContent";
import type { ScorecardGoalDirection, ScorecardThreshold } from "./widget/WidgetScorecard";
import type { WidgetDataSource, WidgetDataSourceKind, WidgetNumberAggregation } from "./widget/types";

type BandStatus = ScorecardThreshold["status"];

const BAND_STATUS_OPTIONS: { value: BandStatus; label: string; dot: string }[] = [
  { value: "good", label: "Good", dot: "bg-emerald-500" },
  { value: "warn", label: "At risk", dot: "bg-amber-500" },
  { value: "bad", label: "Bad", dot: "bg-red-500" },
];

const SOURCE_OPTIONS: { value: WidgetDataSourceKind; label: string; hint: string }[] = [
  { value: "runs", label: "Runs", hint: "One row per canvas run." },
  { value: "executions", label: "Executions", hint: "One row per node execution." },
  { value: "memory", label: "Memory", hint: "Entries stored in a canvas memory namespace." },
];

const AGGREGATION_LABELS: Record<WidgetNumberAggregation, string> = {
  last: "Latest value",
  first: "First value",
  count: "Count of rows",
  sum: "Sum",
  avg: "Average",
  min: "Minimum",
  max: "Maximum",
};

function numberOrUndefined(raw: string): number | undefined {
  if (raw.trim() === "") return undefined;
  const n = Number(raw);
  return Number.isFinite(n) ? n : undefined;
}

/** Reset kind-specific fields to sensible defaults, mirroring `DataSourceForm`. */
function dataSourceForKind(kind: WidgetDataSourceKind): WidgetDataSource {
  if (kind === "memory") return { kind: "memory", namespace: "" };
  if (kind === "runs") return { kind: "runs", limit: 100 };
  return { kind: "executions", limit: 50 };
}

export function ScorecardPanelForm({
  value,
  onChange,
}: {
  value: ScorecardPanelContent;
  onChange: (next: ScorecardPanelContent) => void;
}) {
  return (
    <div className="flex flex-col gap-6">
      <DataSourceSection value={value} onChange={onChange} />
      <ValueSection value={value} onChange={onChange} />
      <DisplaySection value={value} onChange={onChange} />
      <StatusSection value={value} onChange={onChange} />
      <TrendSection value={value} onChange={onChange} />
      <OptionsSection value={value} onChange={onChange} />
    </div>
  );
}

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-col gap-0.5">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">{title}</h3>
        {hint ? <p className="text-[11px] text-slate-400 dark:text-gray-500">{hint}</p> : null}
      </div>
      {children}
    </section>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-300">{label}</Label>
      {children}
      {hint ? <p className="text-[11px] text-slate-400 dark:text-gray-500">{hint}</p> : null}
    </div>
  );
}

interface SegmentOption<T extends string> {
  value: T;
  label: string;
}

function SegmentedToggle<T extends string>({
  options,
  value,
  onChange,
}: {
  options: SegmentOption<T>[];
  value: T;
  onChange: (next: T) => void;
}) {
  return (
    <div className="inline-flex rounded-md border border-slate-200 p-0.5 dark:border-gray-700">
      {options.map((option) => (
        <Button
          key={option.value}
          type="button"
          size="sm"
          variant={option.value === value ? "secondary" : "ghost"}
          className="h-7 rounded-[5px] px-3 text-xs"
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </Button>
      ))}
    </div>
  );
}

function SwitchRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex flex-col gap-0.5">
        <span className="text-xs font-medium text-slate-600 dark:text-gray-300">{label}</span>
        {description ? <span className="text-[11px] text-slate-400 dark:text-gray-500">{description}</span> : null}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}

function DataSourceSection({ value, onChange }: SectionProps) {
  const source = value.dataSource;
  const activeHint = SOURCE_OPTIONS.find((option) => option.value === source.kind)?.hint;
  return (
    <Section title="Data source" hint="Where the rows come from — the same sources every data panel uses.">
      <Field label="Source" hint={activeHint}>
        <Select
          value={source.kind}
          onValueChange={(kind) => onChange({ ...value, dataSource: dataSourceForKind(kind as WidgetDataSourceKind) })}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SOURCE_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      <DataSourceFields value={value} onChange={onChange} />
    </Section>
  );
}

function DataSourceFields({ value, onChange }: SectionProps) {
  const source = value.dataSource;
  if (source.kind === "memory") {
    const missingNamespace = !source.namespace.trim();
    return (
      <Field
        label="Namespace"
        hint={missingNamespace ? "Choose the memory namespace to read entries from." : "e.g. deploy_metrics"}
      >
        <Input
          value={source.namespace}
          onChange={(e) => onChange({ ...value, dataSource: { kind: "memory", namespace: e.target.value } })}
          placeholder="deploy_metrics"
          aria-invalid={missingNamespace}
        />
      </Field>
    );
  }
  if (source.kind === "executions") {
    return (
      <div className="grid grid-cols-2 gap-3">
        <Field label="Node" hint="Optional — limit to one node's executions.">
          <Input
            value={source.node ?? ""}
            onChange={(e) => onChange({ ...value, dataSource: { ...source, node: e.target.value || undefined } })}
            placeholder="All nodes"
          />
        </Field>
        <Field label="Limit" hint="Most recent rows to read.">
          <Input
            type="number"
            value={source.limit ?? ""}
            onChange={(e) =>
              onChange({ ...value, dataSource: { ...source, limit: numberOrUndefined(e.target.value) } })
            }
            placeholder="50"
          />
        </Field>
      </div>
    );
  }
  return (
    <Field label="Limit" hint="Most recent runs to read (count uses the server total).">
      <Input
        type="number"
        value={source.limit ?? ""}
        onChange={(e) => onChange({ ...value, dataSource: { kind: "runs", limit: numberOrUndefined(e.target.value) } })}
        placeholder="100"
      />
    </Field>
  );
}

function ValueSection({ value, onChange }: SectionProps) {
  const needsField = value.aggregation !== "count";
  const missingField = needsField && !value.field?.trim();
  return (
    <Section title="Value" hint="How the rows are reduced to the single number shown.">
      <div className="grid grid-cols-2 gap-3">
        <Field label="Calculation">
          <Select
            value={value.aggregation}
            onValueChange={(v) => onChange({ ...value, aggregation: v as WidgetNumberAggregation })}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SCORECARD_AGGREGATIONS.map((aggregation) => (
                <SelectItem key={aggregation} value={aggregation}>
                  {AGGREGATION_LABELS[aggregation]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label="Value format">
          <Select
            value={value.format ?? "number"}
            onValueChange={(v) => onChange({ ...value, format: v as ScorecardPanelContent["format"] })}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SCORECARD_FORMATS.map((format) => (
                <SelectItem key={format} value={format}>
                  {format}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
      </div>
      {needsField ? (
        <Field
          label="Field"
          hint={missingField ? "Choose the field to aggregate." : "Dot path into each row, e.g. success_rate."}
        >
          <Input
            value={value.field ?? ""}
            onChange={(e) => onChange({ ...value, field: e.target.value })}
            placeholder="success_rate"
            aria-invalid={missingField}
          />
        </Field>
      ) : null}
    </Section>
  );
}

function DisplaySection({ value, onChange }: SectionProps) {
  return (
    <Section title="Display">
      <Field label="Label">
        <Input
          value={value.label ?? ""}
          onChange={(e) => onChange({ ...value, label: e.target.value })}
          placeholder="e.g. Success rate"
        />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Prefix" hint="Shown before the value (e.g. $).">
          <Input
            value={value.prefix ?? ""}
            onChange={(e) => onChange({ ...value, prefix: e.target.value })}
            placeholder="$"
          />
        </Field>
        <Field label="Suffix" hint="Shown after the value (e.g. ms).">
          <Input
            value={value.suffix ?? ""}
            onChange={(e) => onChange({ ...value, suffix: e.target.value })}
            placeholder=" ms"
          />
        </Field>
      </div>
    </Section>
  );
}

function StatusSection({ value, onChange }: SectionProps) {
  const missingTarget = value.statusMode === "target" && (value.target == null || !Number.isFinite(value.target));
  return (
    <Section title="Status" hint="When is this metric healthy? This drives the value color.">
      <Field label="Goal direction">
        <SegmentedToggle<ScorecardGoalDirection>
          options={[
            { value: "higher", label: "Higher is better" },
            { value: "lower", label: "Lower is better" },
          ]}
          value={value.goalDirection}
          onChange={(goalDirection) => onChange({ ...value, goalDirection })}
        />
      </Field>
      <Field label="How to evaluate">
        <SegmentedToggle<ScorecardStatusMode>
          options={[
            { value: "target", label: "Single target" },
            { value: "thresholds", label: "Threshold bands" },
          ]}
          value={value.statusMode}
          onChange={(statusMode) => onChange({ ...value, statusMode })}
        />
      </Field>
      {value.statusMode === "target" ? (
        <Field label="Target value" hint={missingTarget ? "Set a target so the value can be scored." : undefined}>
          <Input
            type="number"
            value={value.target ?? ""}
            onChange={(e) => onChange({ ...value, target: numberOrUndefined(e.target.value) })}
            placeholder="e.g. 95"
            aria-invalid={missingTarget}
          />
        </Field>
      ) : (
        <ThresholdBandsEditor value={value} onChange={onChange} />
      )}
    </Section>
  );
}

function ThresholdBandsEditor({ value, onChange }: SectionProps) {
  const bands = value.thresholds ?? [];
  const comparator = value.goalDirection === "higher" ? "≥" : "≤";

  const updateBand = (index: number, patch: Partial<ScorecardThreshold>) => {
    const next = bands.map((band, i) => (i === index ? { ...band, ...patch } : band));
    onChange({ ...value, thresholds: next });
  };
  const removeBand = (index: number) => {
    onChange({ ...value, thresholds: bands.filter((_, i) => i !== index) });
  };
  const addBand = () => {
    onChange({ ...value, thresholds: [...bands, { at: 0, status: "warn" }] });
  };

  return (
    <div className="flex flex-col gap-2">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-300">Threshold bands</Label>
      {bands.length === 0 ? (
        <p className="text-[11px] text-slate-400 dark:text-gray-500">
          Add at least one band. A band applies when the value is {comparator} its value.
        </p>
      ) : null}
      <div className="flex flex-col gap-2">
        {bands.map((band, index) => (
          <BandRow
            key={index}
            band={band}
            comparator={comparator}
            onChange={(patch) => updateBand(index, patch)}
            onRemove={() => removeBand(index)}
          />
        ))}
      </div>
      <div>
        <Button type="button" size="sm" variant="outline" className="h-7 text-xs" onClick={addBand}>
          <Plus className="mr-1 size-3.5" />
          Add band
        </Button>
      </div>
    </div>
  );
}

function BandRow({
  band,
  comparator,
  onChange,
  onRemove,
}: {
  band: ScorecardThreshold;
  comparator: string;
  onChange: (patch: Partial<ScorecardThreshold>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <Select value={band.status} onValueChange={(v) => onChange({ status: v as BandStatus })}>
        <SelectTrigger className="h-8 w-32">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {BAND_STATUS_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              <span className="flex items-center gap-2">
                <span className={cn("size-2 rounded-full", option.dot)} />
                {option.label}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <span className="text-xs text-slate-400 dark:text-gray-500">value {comparator}</span>
      <Input
        type="number"
        className="h-8 flex-1"
        value={Number.isFinite(band.at) ? band.at : ""}
        onChange={(e) => onChange({ at: numberOrUndefined(e.target.value) ?? NaN })}
        placeholder="0"
      />
      <Button
        type="button"
        size="icon"
        variant="ghost"
        className="size-8 shrink-0 text-slate-400 hover:text-red-600 dark:hover:text-red-400"
        onClick={onRemove}
        aria-label="Remove band"
      >
        <Trash2 className="size-3.5" />
      </Button>
    </div>
  );
}

function TrendSection({ value, onChange }: SectionProps) {
  const seriesOn = value.showSparkline || value.showTrend;
  const missingSeries = seriesOn && !value.seriesField?.trim();
  return (
    <Section title="Trend & sparkline" hint="Both read one numeric field plotted across the rows (oldest → newest).">
      {seriesOn ? (
        <Field
          label="Series field"
          hint={
            missingSeries
              ? "Choose a numeric field to plot across the rows."
              : "Dot path plotted per row, e.g. success_rate."
          }
        >
          <Input
            value={value.seriesField ?? ""}
            onChange={(e) => onChange({ ...value, seriesField: e.target.value })}
            placeholder="success_rate"
            aria-invalid={missingSeries}
          />
        </Field>
      ) : null}
      <SwitchRow
        label="Show sparkline"
        description="A small line of the series beneath the value."
        checked={Boolean(value.showSparkline)}
        onCheckedChange={(on) => onChange({ ...value, showSparkline: on })}
      />
      <SwitchRow
        label="Show change vs start of range"
        description="Compares the latest point to the first point of the series."
        checked={Boolean(value.showTrend)}
        onCheckedChange={(on) => onChange({ ...value, showTrend: on })}
      />
      {value.showTrend ? (
        <Field label="Change caption" hint="Text shown next to the change indicator.">
          <Input
            value={value.trendLabel ?? ""}
            onChange={(e) => onChange({ ...value, trendLabel: e.target.value })}
            placeholder="vs start of range"
          />
        </Field>
      ) : null}
    </Section>
  );
}

function OptionsSection({ value, onChange }: SectionProps) {
  const canShowProgress = scorecardHasGoalLine(value);
  return (
    <Section title="Options">
      <SwitchRow
        label="Show progress to target"
        description={canShowProgress ? "A bar showing progress toward the goal line." : "Set a target or bands first."}
        checked={Boolean(value.showProgress) && canShowProgress}
        onCheckedChange={(on) => onChange({ ...value, showProgress: on })}
      />
    </Section>
  );
}

interface SectionProps {
  value: ScorecardPanelContent;
  onChange: (next: ScorecardPanelContent) => void;
}
