import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import {
  CHART_KIND_LABELS,
  CHART_KINDS,
  CHART_LEGEND_MODE_LABELS,
  CHART_X_AXIS_FORMATS,
  CHART_Y_AXIS_FORMATS,
} from "./chartPanelFormConstants";
import type { ChartPanelContent } from "./panelTypes";
import {
  WIDGET_CHART_LEGEND_MODES,
  WIDGET_SORT_ORDERS,
  type WidgetChartKind,
  type WidgetChartLegendMode,
  type WidgetColumnFormat,
  type WidgetSort,
  type WidgetSortOrder,
} from "./widget/types";

const NONE_VALUE = "__none__";

export function ChartTopControls({
  value,
  onChange,
  fieldListId,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
  fieldListId: string | undefined;
}) {
  const seriesField = value.render.seriesField ?? "";
  const updateSeriesField = (next: string) => {
    const trimmed = next.trim();
    if (!trimmed) {
      const { seriesField: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    onChange({ ...value, render: { ...value.render, seriesField: next } });
  };

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Chart type</Label>
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
          <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">X-axis field</Label>
          <Input
            list={fieldListId}
            value={value.render.xField}
            onChange={(e) => onChange({ ...value, render: { ...value.render, xField: e.target.value } })}
            placeholder='e.g. status or {{ formatDate(createdAt, "MM/dd") }}'
            data-testid="chart-x-field"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Legend</Label>
          <Select
            value={value.render.legend ?? "auto"}
            onValueChange={(v) =>
              onChange({ ...value, render: { ...value.render, legend: v as WidgetChartLegendMode } })
            }
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
      <ChartAxisFormatRow value={value} onChange={onChange} />
      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-1.5 col-span-2">
          <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Stack by field (optional)</Label>
          <Input
            list={fieldListId}
            value={seriesField}
            onChange={(e) => updateSeriesField(e.target.value)}
            placeholder="e.g. service (pivots rows into one series per value)"
            data-testid="chart-series-field"
          />
          <p className="text-[11px] text-slate-500 dark:text-gray-400">
            When set, the value comes from the first series&apos; field, summed per (X, Stack) bucket.
          </p>
        </div>
      </div>
      <ChartSortRow value={value} onChange={onChange} fieldListId={fieldListId} />
    </div>
  );
}

function ChartAxisFormatRow({
  value,
  onChange,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
}) {
  const updateAxisFormat = (key: "xFormat" | "yFormat", next: string) => {
    if (next === NONE_VALUE) {
      const { [key]: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    onChange({ ...value, render: { ...value.render, [key]: next as WidgetColumnFormat } });
  };

  const updateYLabel = (next: string) => {
    if (next.trim() === "") {
      const { yLabel: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    onChange({ ...value, render: { ...value.render, yLabel: next } });
  };

  return (
    <div className="grid grid-cols-3 gap-3">
      <AxisFormatSelect
        label="X-axis format"
        value={value.render.xFormat}
        formats={CHART_X_AXIS_FORMATS}
        testId="chart-x-format"
        onValueChange={(v) => updateAxisFormat("xFormat", v)}
      />
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Y-axis label</Label>
        <Input
          value={value.render.yLabel ?? ""}
          onChange={(e) => updateYLabel(e.target.value)}
          placeholder="e.g. USD or Errors / day"
          data-testid="chart-y-label"
        />
      </div>
      <AxisFormatSelect
        label="Y-axis format"
        value={value.render.yFormat}
        formats={CHART_Y_AXIS_FORMATS}
        testId="chart-y-format"
        onValueChange={(v) => updateAxisFormat("yFormat", v)}
      />
    </div>
  );
}

function AxisFormatSelect({
  label,
  value,
  formats,
  testId,
  onValueChange,
}: {
  label: string;
  value: WidgetColumnFormat | undefined;
  formats: readonly WidgetColumnFormat[];
  testId: string;
  onValueChange: (next: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">{label}</Label>
      <Select value={value ?? NONE_VALUE} onValueChange={onValueChange}>
        <SelectTrigger className="w-full" data-testid={testId}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NONE_VALUE}>Default</SelectItem>
          {formats.map((f) => (
            <SelectItem key={f} value={f}>
              {f}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function ChartSortRow({
  value,
  onChange,
  fieldListId,
}: {
  value: ChartPanelContent;
  onChange: (next: ChartPanelContent) => void;
  fieldListId: string | undefined;
}) {
  const sort = value.render.sort;
  const sortField = sort?.field ?? "";
  const sortOrder: WidgetSortOrder = sort?.order ?? "asc";
  const hasSortField = sortField.trim() !== "";

  const updateField = (nextField: string) => {
    const trimmed = nextField.trim();
    if (!trimmed) {
      const { sort: _omit, ...rest } = value.render;
      void _omit;
      onChange({ ...value, render: rest });
      return;
    }
    const nextSort: WidgetSort = { field: nextField };
    if (sort?.order) nextSort.order = sort.order;
    onChange({ ...value, render: { ...value.render, sort: nextSort } });
  };

  const updateOrder = (nextOrder: WidgetSortOrder) => {
    if (!hasSortField) return;
    onChange({
      ...value,
      render: { ...value.render, sort: { field: sortField, order: nextOrder } },
    });
  };

  return (
    <div className="grid grid-cols-3 gap-3">
      <div className="space-y-1.5 col-span-2">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Sort by (optional)</Label>
        <Input
          list={fieldListId}
          value={sortField}
          onChange={(e) => updateField(e.target.value)}
          placeholder="e.g. createdAt or {{ expr }} (blank = unsorted)"
          data-testid="chart-sort-field"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Order</Label>
        <Select value={sortOrder} onValueChange={(v) => updateOrder(v as WidgetSortOrder)} disabled={!hasSortField}>
          <SelectTrigger className="w-full" data-testid="chart-sort-order">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_SORT_ORDERS.map((o) => (
              <SelectItem key={o} value={o}>
                {o === "asc" ? "Ascending" : "Descending"}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
