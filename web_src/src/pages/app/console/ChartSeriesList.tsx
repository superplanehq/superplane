import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

import { ChartSeriesRow } from "./ChartSeriesRow";
import type { ChartPanelContent } from "./panelTypes";
import type { WidgetChartSeries } from "./widget/types";

export function ChartSeriesList({
  value,
  onChange,
  sampleRow,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
  sampleRow: Record<string, unknown>;
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
            sampleRow={sampleRow}
            onChange={(patch) => updateSeries(idx, patch)}
            onRemove={() => removeSeries(idx)}
          />
        ))}
      </div>
    </div>
  );
}
