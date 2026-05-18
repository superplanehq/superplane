import { useState } from "react";
import { AlertTriangle, Trash2 } from "lucide-react";

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
import { WidgetChart } from "./widget/WidgetChart";
import type { WidgetChartKind, WidgetChartSeries } from "./widget/types";

const CHART_KINDS: WidgetChartKind[] = ["bar", "stacked-bar", "line", "area", "donut"];

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
      <div className="grid grid-cols-2 gap-3">
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
                  {k}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">X-axis field</Label>
          <Input
            value={value.render.xField}
            onChange={(e) => onChange({ ...value, render: { ...value.render, xField: e.target.value } })}
            placeholder="e.g. status"
          />
        </div>
      </div>
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Series</Label>
          <Button type="button" size="sm" variant="outline" onClick={addSeries} data-testid="chart-add-series">
            Add series
          </Button>
        </div>
        <div className="space-y-2">
          {value.render.series.map((s, idx) => (
            <div key={idx} className="grid grid-cols-12 items-center gap-2 rounded border border-slate-200 p-2">
              <Input
                className="col-span-5 h-8"
                value={s.field ?? ""}
                onChange={(e) => updateSeries(idx, { field: e.target.value || undefined })}
                placeholder="field (blank = count)"
                aria-label={`Series ${idx + 1} field`}
              />
              <Input
                className="col-span-5 h-8"
                value={s.label ?? ""}
                onChange={(e) => updateSeries(idx, { label: e.target.value })}
                placeholder="label"
                aria-label={`Series ${idx + 1} label`}
              />
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="col-span-2 h-8 w-8 text-slate-400 hover:text-red-600"
                onClick={() => removeSeries(idx)}
                aria-label={`Remove series ${idx + 1}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))}
        </div>
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
      series: [{ field: "count", label: "Count" }],
    },
  };
}
