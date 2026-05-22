import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { CHART_KIND_LABELS, CHART_KINDS, CHART_LEGEND_MODE_LABELS } from "./chartPanelFormConstants";
import type { ChartPanelContent } from "./panelTypes";
import { WIDGET_CHART_LEGEND_MODES, type WidgetChartKind, type WidgetChartLegendMode } from "./widget/types";

export function ChartTopControls({
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
                {CHART_LEGEND_MODE_LABELS[m]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
