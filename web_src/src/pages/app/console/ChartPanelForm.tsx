import { Info } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { ChartSeriesList } from "./ChartSeriesList";
import { ChartTopControls } from "./ChartTopControls";
import { DataSourceForm } from "./DataSourceForm";
import { useConsoleContext } from "./ConsoleContext";
import type { ChartPanelContent } from "./panelTypes";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

export function ChartPanelForm({
  value,
  onChange,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const memoryNamespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const fieldListId = memoryNamespace ? `chart-fields-${memoryNamespace}` : undefined;
  const hasFieldSuggestions = fields.length > 0 && Boolean(fieldListId);
  const hasSeriesFieldPivot = Boolean(value.render.seriesField?.trim());
  const stackedBarNeedsMoreSeries =
    value.render.type === "stacked-bar" && !hasSeriesFieldPivot && value.render.series.length < 2;

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
