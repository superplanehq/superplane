import { useState } from "react";
import { AlertTriangle, Info, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { DashboardPanel } from "@/hooks/useCanvasData";

import { DataSourceForm } from "./DataSourceForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import type { ChartPanelContent } from "./panelTypes";
import { useWidgetData } from "./widget/useWidgetData";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";
import { WidgetChart } from "./widget/WidgetChart";
import {
  WIDGET_CHART_LEGEND_MODES,
  type WidgetChartKind,
  type WidgetChartLegendMode,
  type WidgetChartSeries,
  type WidgetColumnFormat,
} from "./widget/types";

const CHART_KINDS: WidgetChartKind[] = ["bar", "stacked-bar", "line", "area", "donut"];
const CHART_KIND_LABELS: Record<WidgetChartKind, string> = {
  bar: "Bar",
  "stacked-bar": "Stacked bar",
  line: "Line",
  area: "Area",
  donut: "Donut",
};
const SERIES_FORMATS: WidgetColumnFormat[] = ["text", "number", "percent", "duration"];
const LEGEND_MODE_LABELS: Record<WidgetChartLegendMode, string> = {
  auto: "Auto",
  show: "Always show",
  hide: "Hide",
};

interface ChartPanelCardProps {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}

export function ChartPanelCard({ panel, readOnly, onDelete, onChange }: ChartPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        typeLabel="Chart"
        readOnly={readOnly}
        onEdit={() => setEditing(true)}
        onDelete={onDelete}
      >
        <ChartPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<ChartPanelContent>
        open={editing}
        onOpenChange={setEditing}
        panelId={panel.id}
        panelType="chart"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <ChartPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function ChartPanelBody({ content }: { content: ChartPanelContent }) {
  const ctx = useDashboardContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  return <ChartPanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function ChartPanelDataBound({ content, canvasId }: { content: ChartPanelContent; canvasId: string }) {
  const { rows, isLoading, error } = useWidgetData(canvasId, content.dataSource);
  if (error) return <PanelError message={error} />;
  return <WidgetChart render={content.render} rows={rows} isLoading={isLoading} />;
}

function ChartPanelForm({
  value,
  onChange,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
}) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const memoryNamespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const fieldListId = memoryNamespace ? `chart-fields-${memoryNamespace}` : undefined;
  const hasFieldSuggestions = fields.length > 0 && Boolean(fieldListId);
  const stackedBarNeedsMoreSeries = value.render.type === "stacked-bar" && value.render.series.length < 2;

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>
      <DataSourceForm value={value.dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
      <ChartTopControls value={value} onChange={onChange} fieldListId={hasFieldSuggestions ? fieldListId : undefined} />
      {stackedBarNeedsMoreSeries ? <StackedBarHint /> : null}
      {hasFieldSuggestions ? (
        <datalist id={fieldListId}>
          {fields.map((f) => (
            <option key={f.field} value={f.field} />
          ))}
        </datalist>
      ) : null}
      <ChartSeriesList value={value} onChange={onChange} fieldListId={hasFieldSuggestions ? fieldListId : undefined} />
    </div>
  );
}

function ChartTopControls({
  value,
  onChange,
  fieldListId,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
  fieldListId: string | undefined;
}) {
  return (
    <div className="grid grid-cols-3 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Chart type</Label>
        <Select
          value={value.render.type}
          onValueChange={(v) => onChange({ ...value, render: { ...value.render, type: v as WidgetChartKind } })}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {CHART_KINDS.map((k) => (
              <SelectItem key={k} value={k}>
                {CHART_KIND_LABELS[k]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">X-axis field</Label>
        <Input
          list={fieldListId}
          value={value.render.xField}
          onChange={(e) => onChange({ ...value, render: { ...value.render, xField: e.target.value } })}
          placeholder="e.g. status"
          data-testid="chart-x-field"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Legend</Label>
        <Select
          value={value.render.legend ?? "auto"}
          onValueChange={(v) => onChange({ ...value, render: { ...value.render, legend: v as WidgetChartLegendMode } })}
        >
          <SelectTrigger className="w-full" data-testid="chart-legend-mode">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_CHART_LEGEND_MODES.map((m) => (
              <SelectItem key={m} value={m}>
                {LEGEND_MODE_LABELS[m]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function StackedBarHint() {
  return (
    <div
      className="flex items-start gap-2 rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800"
      data-testid="chart-stacked-bar-hint"
    >
      <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>Stacked bar combines multiple series per category. Add another series, or pick Bar.</span>
    </div>
  );
}

function ChartSeriesList({
  value,
  onChange,
  fieldListId,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
  fieldListId: string | undefined;
}) {
  const updateSeries = (idx: number, patch: Partial<WidgetChartSeries>) => {
    const series = value.render.series.map((s, i) => (i === idx ? { ...s, ...patch } : s));
    onChange({ ...value, render: { ...value.render, series } });
  };
  const addSeries = () => {
    onChange({
      ...value,
      render: { ...value.render, series: [...value.render.series, { field: "", label: "" }] },
    });
  };
  const removeSeries = (idx: number) => {
    onChange({ ...value, render: { ...value.render, series: value.render.series.filter((_, i) => i !== idx) } });
  };
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Series</Label>
        <Button type="button" size="sm" variant="outline" onClick={addSeries} data-testid="chart-add-series">
          Add series
        </Button>
      </div>
      <div className="space-y-2">
        {value.render.series.map((s, idx) => (
          <ChartSeriesRow
            key={idx}
            index={idx}
            series={s}
            fieldListId={fieldListId}
            onChange={(patch) => updateSeries(idx, patch)}
            onRemove={() => removeSeries(idx)}
          />
        ))}
      </div>
    </div>
  );
}

function ChartSeriesRow({
  index,
  series,
  fieldListId,
  onChange,
  onRemove,
}: {
  index: number;
  series: WidgetChartSeries;
  fieldListId: string | undefined;
  onChange: (patch: Partial<WidgetChartSeries>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="space-y-2 rounded border border-slate-200 p-2">
      <div className="grid grid-cols-12 items-center gap-2">
        <Input
          list={fieldListId}
          className="col-span-5 h-8"
          value={series.field ?? ""}
          onChange={(e) => onChange({ field: e.target.value || undefined })}
          placeholder="field (blank = count)"
          aria-label={`Series ${index + 1} field`}
        />
        <Input
          className="col-span-5 h-8"
          value={series.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value || undefined })}
          placeholder="label"
          aria-label={`Series ${index + 1} label`}
        />
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="col-span-2 h-8 w-8 text-slate-400 hover:text-red-600"
          onClick={onRemove}
          aria-label={`Remove series ${index + 1}`}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      <div className="grid grid-cols-12 items-center gap-2">
        <div className="col-span-4">
          <Select
            value={series.format ?? "__none__"}
            onValueChange={(v) => onChange({ format: v === "__none__" ? undefined : (v as WidgetColumnFormat) })}
          >
            <SelectTrigger className="h-8 w-full" aria-label={`Series ${index + 1} format`}>
              <SelectValue placeholder="Format" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">Default</SelectItem>
              {SERIES_FORMATS.map((f) => (
                <SelectItem key={f} value={f}>
                  {f}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Input
          className="col-span-4 h-8"
          value={series.prefix ?? ""}
          onChange={(e) => onChange({ prefix: e.target.value || undefined })}
          placeholder="prefix (e.g. $)"
          aria-label={`Series ${index + 1} prefix`}
        />
        <Input
          className="col-span-4 h-8"
          value={series.suffix ?? ""}
          onChange={(e) => onChange({ suffix: e.target.value || undefined })}
          placeholder="suffix (e.g. MWh)"
          aria-label={`Series ${index + 1} suffix`}
        />
      </div>
    </div>
  );
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): ChartPanelContent {
  const r = raw ?? {};
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as ChartPanelContent["dataSource"]) ?? { kind: "executions", limit: 100 },
    render: (r.render as ChartPanelContent["render"]) ?? {
      kind: "chart",
      type: "bar",
      xField: "status",
      series: [{ label: "Count" }],
    },
  };
}
