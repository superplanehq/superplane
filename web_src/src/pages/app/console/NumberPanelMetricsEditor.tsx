import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { DataSourceForm } from "./DataSourceForm";
import { useConsoleContext } from "./ConsoleContext";
import { NumberFormatField, NumberPrefixSuffixFields, NumberSparklineField } from "./NumberRenderFields";
import { NUMBER_PANEL_AGGREGATIONS } from "./numberPanelFormConstants";
import type { NumberMetric, NumberPanelContent, TablePanelDataSource } from "./panelTypes";
import type { WidgetNumberAggregation, WidgetNumberRender } from "./widget/types";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

const DEFAULT_METRIC: NumberMetric = {
  dataSource: { kind: "runs", limit: 100 },
  render: { kind: "number", aggregation: "count", label: "" },
};

/**
 * A metric drafted in the YAML tab can be missing `dataSource` and/or `render`
 * before schema validation passes. The editor reads those fields directly, so
 * fill in safe defaults here to keep the form rendering (the dialog still
 * surfaces the validation error and blocks Save) instead of throwing and
 * tearing down the whole panel editor.
 */
function normalizeMetric(metric: NumberMetric | undefined): NumberMetric {
  return {
    dataSource: metric?.dataSource ?? { kind: "runs", limit: 100 },
    render: metric?.render ?? { kind: "number", aggregation: "count" },
  };
}

export function NumberPanelMetricsEditor({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const metrics = value.metrics ?? [];

  const updateMetric = (idx: number, patch: Partial<NumberMetric>) => {
    const next = metrics.map((metric, i) => (i === idx ? { ...metric, ...patch } : metric));
    onChange({ ...value, metrics: next });
  };

  const addMetric = () => {
    onChange({ ...value, metrics: [...metrics, DEFAULT_METRIC] });
  };

  const removeMetric = (idx: number) => {
    if (metrics.length <= 1) return;
    onChange({ ...value, metrics: metrics.filter((_, i) => i !== idx) });
  };

  return (
    <div className="space-y-3 rounded-md border border-slate-200 bg-slate-50/40 p-3">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Numbers</Label>
        <Button type="button" size="sm" variant="outline" onClick={addMetric} data-testid="number-add-metric">
          Add number
        </Button>
      </div>
      <p className="text-[11px] text-slate-500">
        Each number is configured independently. They render side-by-side and wrap to new lines when the panel is
        narrow.
      </p>
      <div className="space-y-2">
        {metrics.map((metric, idx) => (
          <MetricRow
            key={idx}
            metric={metric}
            index={idx}
            onChange={(patch) => updateMetric(idx, patch)}
            onRemove={metrics.length > 1 ? () => removeMetric(idx) : undefined}
          />
        ))}
      </div>
    </div>
  );
}

function MetricRow({
  metric: rawMetric,
  index,
  onChange,
  onRemove,
}: {
  metric: NumberMetric;
  index: number;
  onChange: (patch: Partial<NumberMetric>) => void;
  onRemove?: () => void;
}) {
  const metric = normalizeMetric(rawMetric);
  const updateRender = (patch: Partial<WidgetNumberRender>) => onChange({ render: { ...metric.render, ...patch } });
  return (
    <div className="space-y-2 rounded-md border border-slate-200 bg-white p-3" data-testid={`number-metric-${index}`}>
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Number {index + 1}</span>
        {onRemove ? (
          <Button type="button" size="sm" variant="ghost" onClick={onRemove} data-testid="number-metric-remove">
            <Trash2 className="h-3.5 w-3.5" />
            <span className="sr-only">Remove number</span>
          </Button>
        ) : null}
      </div>
      <MetricNameField metric={metric} onChange={onChange} />
      <DataSourceForm
        value={metric.dataSource}
        onChange={(ds) => onChange({ dataSource: ds as TablePanelDataSource })}
      />
      <MetricAggregationFields metric={metric} index={index} onChange={onChange} />
      <NumberFormatField render={metric.render} variant="compact" onChange={updateRender} />
      <NumberPrefixSuffixFields render={metric.render} variant="compact" onChange={updateRender} />
      <NumberSparklineField render={metric.render} variant="compact" onChange={updateRender} />
    </div>
  );
}

function MetricNameField({
  metric,
  onChange,
}: {
  metric: NumberMetric;
  onChange: (patch: Partial<NumberMetric>) => void;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Name</Label>
      <Input
        value={metric.render.label ?? ""}
        onChange={(e) => onChange({ render: { ...metric.render, label: e.target.value || undefined } })}
        placeholder="e.g. Total runs"
        data-testid="number-metric-name"
      />
    </div>
  );
}

function MetricAggregationFields({
  metric,
  index,
  onChange,
}: {
  metric: NumberMetric;
  index: number;
  onChange: (patch: Partial<NumberMetric>) => void;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const memoryNamespace = metric.dataSource.kind === "memory" ? metric.dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const aggregation = metric.render.aggregation ?? "count";
  const needsField = aggregation !== "count";
  const fieldListId =
    memoryNamespace && fields.length > 0 ? `number-metric-${index}-fields-${memoryNamespace}` : undefined;

  return (
    <div className="grid grid-cols-2 gap-2">
      <div className="space-y-1">
        <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Aggregation</Label>
        <Select
          value={aggregation}
          onValueChange={(v) =>
            onChange({
              render: {
                ...metric.render,
                aggregation: v as WidgetNumberAggregation,
                field: v === "count" ? undefined : metric.render.field,
              } satisfies WidgetNumberRender,
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {NUMBER_PANEL_AGGREGATIONS.map((a) => (
              <SelectItem key={a} value={a}>
                {a}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {needsField ? (
        <div className="space-y-1">
          <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Field</Label>
          <Input
            list={fieldListId}
            value={metric.render.field ?? ""}
            onChange={(e) => onChange({ render: { ...metric.render, field: e.target.value || undefined } })}
            placeholder="e.g. durationMs"
            data-testid="number-metric-field"
          />
          {fieldListId ? (
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
