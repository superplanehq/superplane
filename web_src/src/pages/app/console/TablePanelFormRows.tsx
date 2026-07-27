import { useId } from "react";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/ui/checkbox";

import { ROW_STYLE_CLASS, ROW_STYLE_LABEL } from "./widget/rowStyles";
import {
  WIDGET_FILTER_OPS,
  WIDGET_ROW_STYLE_TONES,
  WIDGET_TREND_BETTER,
  WIDGET_TREND_DISPLAYS,
  columnSupportsShowTrend,
  type WidgetColumnFormat,
  type WidgetProgressLabel,
  type WidgetRowStyle,
  type WidgetRowStyleTone,
  type WidgetTableColumn,
  type WidgetTableFilter,
  type WidgetTrendBetter,
  type WidgetTrendDisplay,
} from "./widget/types";
import { suggestColumnFormat } from "./widget/useMemoryCatalog";

const COLUMN_FORMATS: WidgetColumnFormat[] = [
  "text",
  "number",
  "percent",
  "date",
  "datetime",
  "relative",
  "duration",
  "status",
  "badge",
  "code",
  "link",
  "avatar",
  "progress",
  "trend",
];

const PROGRESS_LABEL_OPTIONS: { value: WidgetProgressLabel; label: string }[] = [
  { value: "percent", label: "Percent (50%)" },
  { value: "number", label: "Number (5/10)" },
  { value: "none", label: "None" },
];

const TREND_BETTER_LABEL: Record<WidgetTrendBetter, string> = {
  up: "Up is better",
  down: "Down is better",
};

const TREND_DISPLAY_LABEL: Record<WidgetTrendDisplay, string> = {
  percent: "Percent",
  value: "Value",
  none: "None",
};

function columnFormatPatch(format: WidgetColumnFormat | undefined, col: WidgetTableColumn): Partial<WidgetTableColumn> {
  const keepTrendOpts = format === "trend" || columnSupportsShowTrend(format);
  return {
    format,
    ...(format === "link" ? {} : { href: undefined }),
    ...(format === "progress"
      ? { progressLabel: col.progressLabel ?? "percent" }
      : { progressTarget: undefined, progressLabel: undefined }),
    ...(keepTrendOpts
      ? format === "trend"
        ? { showTrend: undefined }
        : {}
      : { showTrend: undefined, trendBetter: undefined, trendDisplay: undefined }),
  };
}

function ProgressFormatFields({
  col,
  fieldOptions,
  onChange,
}: {
  col: WidgetTableColumn;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetTableColumn>) => void;
}) {
  return (
    <>
      <Input
        className="col-span-8 h-8"
        value={col.progressTarget ?? ""}
        onChange={(e) => onChange({ progressTarget: e.target.value || undefined })}
        placeholder="target, e.g. 10, payload.goal or {{ items.size() }}"
        list={fieldOptions.length > 0 ? "table-field-options" : undefined}
        data-testid="table-column-progress-target"
      />
      <Select
        value={col.progressLabel ?? "percent"}
        onValueChange={(v) => onChange({ progressLabel: v as WidgetProgressLabel })}
      >
        <SelectTrigger className="col-span-4 h-8" data-testid="table-column-progress-label">
          <SelectValue placeholder="Label" />
        </SelectTrigger>
        <SelectContent>
          {PROGRESS_LABEL_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </>
  );
}

function TrendFormatFields({
  col,
  onChange,
}: {
  col: WidgetTableColumn;
  onChange: (patch: Partial<WidgetTableColumn>) => void;
}) {
  return (
    <>
      <Select value={col.trendBetter ?? "up"} onValueChange={(v) => onChange({ trendBetter: v as WidgetTrendBetter })}>
        <SelectTrigger className="col-span-6 h-8" data-testid="table-column-trend-better">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_TREND_BETTER.map((direction) => (
            <SelectItem key={direction} value={direction}>
              {TREND_BETTER_LABEL[direction]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={col.trendDisplay ?? "percent"}
        onValueChange={(v) => onChange({ trendDisplay: v as WidgetTrendDisplay })}
      >
        <SelectTrigger className="col-span-6 h-8" data-testid="table-column-trend-display">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_TREND_DISPLAYS.map((mode) => (
            <SelectItem key={mode} value={mode}>
              {TREND_DISPLAY_LABEL[mode]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </>
  );
}

function ShowTrendToggle({
  col,
  onChange,
}: {
  col: WidgetTableColumn;
  onChange: (patch: Partial<WidgetTableColumn>) => void;
}) {
  const showTrendId = useId();
  return (
    <div className="col-span-12 flex items-center gap-2">
      <Checkbox
        id={showTrendId}
        checked={Boolean(col.showTrend)}
        onCheckedChange={(checked) =>
          onChange(
            checked === true
              ? { showTrend: true }
              : { showTrend: undefined, trendBetter: undefined, trendDisplay: undefined },
          )
        }
        className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600 dark:border-gray-600"
        data-testid="table-column-show-trend"
      />
      <Label htmlFor={showTrendId} className="text-xs text-slate-700 dark:text-gray-300">
        Show trend
      </Label>
    </div>
  );
}

export function ColumnRow({
  col,
  fieldOptions,
  formats = COLUMN_FORMATS,
  onChange,
  onRemove,
}: {
  col: WidgetTableColumn;
  fieldOptions: string[];
  formats?: WidgetColumnFormat[];
  onChange: (patch: Partial<WidgetTableColumn>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex gap-2 rounded-lg bg-slate-100 p-2 dark:bg-gray-800">
      <div className="grid min-w-0 flex-1 grid-cols-12 items-center gap-2">
        <Input
          className="col-span-4 h-8"
          value={col.field}
          onChange={(e) => {
            const next = e.target.value;
            const known = fieldOptions.includes(next);
            const suggestedFormat = suggestColumnFormat(next);
            onChange({
              field: next,
              ...(known && !col.label ? { label: next } : {}),
              ...(known && col.format == null
                ? { format: formats.includes(suggestedFormat) ? suggestedFormat : undefined }
                : {}),
            });
          }}
          placeholder="field (e.g. payload.user_id) or {{ expr }}"
          list={fieldOptions.length > 0 ? "table-field-options" : undefined}
          data-testid="table-column-field"
        />
        <Input
          className="col-span-3 h-8"
          value={col.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value })}
          placeholder="Header"
        />
        <Select
          value={col.format ?? "__none__"}
          onValueChange={(v) => {
            const format = v === "__none__" ? undefined : (v as WidgetColumnFormat);
            onChange(columnFormatPatch(format, col));
          }}
        >
          <SelectTrigger className="col-span-4 h-8">
            <SelectValue placeholder="Format" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Default</SelectItem>
            {formats.map((fmt) => (
              <SelectItem key={fmt} value={fmt}>
                {fmt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {col.format === "link" ? (
          <Input
            className="col-span-12 h-8"
            value={col.href ?? ""}
            onChange={(e) => onChange({ href: e.target.value || undefined })}
            placeholder="link URL, e.g. {{ prUrl }} or https://github.com/org/repo/pull/{{ prNumber }}"
            list={fieldOptions.length > 0 ? "table-href-field-options" : undefined}
            data-testid="table-column-href"
          />
        ) : null}
        {col.format === "progress" ? (
          <ProgressFormatFields col={col} fieldOptions={fieldOptions} onChange={onChange} />
        ) : null}
        {columnSupportsShowTrend(col.format) ? (
          <>
            <ShowTrendToggle col={col} onChange={onChange} />
            {col.showTrend ? <TrendFormatFields col={col} onChange={onChange} /> : null}
          </>
        ) : null}
        {col.format === "trend" ? <TrendFormatFields col={col} onChange={onChange} /> : null}
      </div>
      <div className="flex shrink-0 items-start justify-end">
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-red-400"
          onClick={onRemove}
          aria-label="Remove column"
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

export function FilterRow({
  filter,
  fieldOptions,
  onChange,
  onRemove,
}: {
  filter: WidgetTableFilter;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetTableFilter>) => void;
  onRemove: () => void;
}) {
  const needsValue = filter.op !== "exists" && filter.op !== "not_exists";
  return (
    <div className="grid grid-cols-12 gap-2 rounded border border-slate-200 p-2 dark:border-gray-600">
      <Input
        className="col-span-4 h-8"
        value={filter.field}
        onChange={(e) => onChange({ field: e.target.value })}
        placeholder="field (e.g. payload.user_id)"
        list={fieldOptions.length > 0 ? "table-field-options" : undefined}
      />
      <Select value={filter.op} onValueChange={(v) => onChange({ op: v as WidgetTableFilter["op"] })}>
        <SelectTrigger className="col-span-3 h-8">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_FILTER_OPS.map((op) => (
            <SelectItem key={op} value={op}>
              {op}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {needsValue ? (
        <Input
          className="col-span-4 h-8"
          value={filter.value ?? ""}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="value or {{ expr }}"
        />
      ) : (
        <div className="col-span-4" />
      )}
      <Button type="button" size="icon" variant="ghost" className="col-span-1 h-8 w-8" onClick={onRemove}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

export function RowStyleRow({
  rule,
  fieldOptions,
  onChange,
  onRemove,
}: {
  rule: WidgetRowStyle;
  fieldOptions: string[];
  onChange: (patch: Partial<WidgetRowStyle>) => void;
  onRemove: () => void;
}) {
  const needsValue = rule.op !== "exists" && rule.op !== "not_exists";
  return (
    <div className="grid grid-cols-12 gap-2 rounded border border-slate-200 p-2 dark:border-gray-600">
      <Input
        className="col-span-3 h-8"
        value={rule.field}
        onChange={(e) => onChange({ field: e.target.value })}
        placeholder="field (e.g. payload.user_id)"
        list={fieldOptions.length > 0 ? "table-field-options" : undefined}
        data-testid="table-row-style-field"
      />
      <Select value={rule.op} onValueChange={(v) => onChange({ op: v as WidgetTableFilter["op"] })}>
        <SelectTrigger className="col-span-2 h-8" data-testid="table-row-style-op">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_FILTER_OPS.map((op) => (
            <SelectItem key={op} value={op}>
              {op}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {needsValue ? (
        <Input
          className="col-span-3 h-8"
          value={rule.value ?? ""}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="value or {{ expr }}"
          data-testid="table-row-style-value"
        />
      ) : (
        <div className="col-span-3" />
      )}
      <Select value={rule.tone} onValueChange={(v) => onChange({ tone: v as WidgetRowStyleTone })}>
        <SelectTrigger className="col-span-3 h-8" data-testid="table-row-style-tone">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {WIDGET_ROW_STYLE_TONES.map((tone) => (
            <SelectItem key={tone} value={tone}>
              <span className="inline-flex items-center gap-2">
                <span
                  className={`inline-block h-3 w-4 rounded-sm border border-slate-300 dark:border-gray-600 ${ROW_STYLE_CLASS[tone]}`}
                  aria-hidden
                />
                <span>{ROW_STYLE_LABEL[tone]}</span>
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button
        type="button"
        size="icon"
        variant="ghost"
        className="col-span-1 h-8 w-8"
        onClick={onRemove}
        data-testid="table-row-style-remove"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
