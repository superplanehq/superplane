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
    <div className="flex gap-2 rounded-lg bg-slate-100 p-2">
      <div className="min-w-0 flex-1 space-y-2">
        <div className="grid grid-cols-2 gap-2">
          <Input
            list={fieldListId}
            className="h-8"
            value={series.field ?? ""}
            onChange={(e) => onChange({ field: e.target.value || undefined })}
            placeholder="field or {{ expr }} (blank = count)"
            aria-label={`Series ${index + 1} field`}
          />
          <Input
            className="h-8"
            value={series.label ?? ""}
            onChange={(e) => onChange({ label: e.target.value || undefined })}
            placeholder="label"
            aria-label={`Series ${index + 1} label`}
          />
        </div>
        <div className="grid grid-cols-3 gap-2">
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
          <Input
            className="h-8"
            value={series.prefix ?? ""}
            onChange={(e) => onChange({ prefix: e.target.value || undefined })}
            placeholder="prefix (e.g. $)"
            aria-label={`Series ${index + 1} prefix`}
          />
          <Input
            className="h-8"
            value={series.suffix ?? ""}
            onChange={(e) => onChange({ suffix: e.target.value || undefined })}
            placeholder="suffix (e.g. MWh)"
            aria-label={`Series ${index + 1} suffix`}
          />
        </div>
      </div>
      <div className="flex shrink-0 items-start justify-end">
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600"
          onClick={onRemove}
          aria-label={`Remove series ${index + 1}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}
