import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { CHART_SERIES_FORMATS } from "./chartPanelFormConstants";
import type { WidgetChartSeries, WidgetColumnFormat } from "./widget/types";

export function ChartSeriesRow({
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
          placeholder="field or {{ expr }} (blank = count)"
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
              {CHART_SERIES_FORMATS.map((f) => (
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
