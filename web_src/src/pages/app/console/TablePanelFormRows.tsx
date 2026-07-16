import { useId } from "react";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/ui/checkbox";
import { ExpressionEditor } from "@/components/ExpressionEditor";

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
  sampleRow,
  onChange,
}: {
  col: WidgetTableColumn;
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<WidgetTableColumn>) => void;
}) {
  return (
    <>
      <div className="col-span-8">
        <ExpressionEditor
          dialect="cel"
          syntaxProfile="pathOrRaw"
          exampleObj={sampleRow}
          value={col.progressTarget ?? ""}
          onChange={(next) => onChange({ progressTarget: next || undefined })}
          placeholder="target, e.g. 10, payload.goal or {{ items.size() }}"
          inputSize="md"
          showValuePreview
          data-testid="table-column-progress-target"
        />
      </div>
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
  sampleRow,
  onChange,
  onRemove,
}: {
  col: WidgetTableColumn;
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<WidgetTableColumn>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex gap-2 rounded-lg bg-slate-100 p-2 dark:bg-gray-800">
      <div className="grid min-w-0 flex-1 grid-cols-12 items-start gap-2">
        {/* Give the field editor a full row so its preview/quick tip fit. */}
        <div className="col-span-12">
          <ExpressionEditor
            dialect="cel"
            syntaxProfile="pathOrRaw"
            exampleObj={sampleRow}
            value={col.field}
            onChange={(next) => {
              const known = fieldOptions.includes(next);
              onChange({
                field: next,
                ...(known && !col.label ? { label: next } : {}),
                ...(known && col.format == null ? { format: suggestColumnFormat(next) } : {}),
              });
            }}
            placeholder="field (e.g. payload.user_id) or {{ expr }}"
            inputSize="md"
            showValuePreview
            data-testid="table-column-field"
          />
        </div>
        <Input
          className="col-span-6 h-8"
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
          <SelectTrigger className="col-span-6 h-8">
            <SelectValue placeholder="Format" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Default</SelectItem>
            {COLUMN_FORMATS.map((fmt) => (
              <SelectItem key={fmt} value={fmt}>
                {fmt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {col.format === "link" ? (
          <div className="col-span-12">
            <ExpressionEditor
              dialect="cel"
              syntaxProfile="wrapped"
              exampleObj={sampleRow}
              value={col.href ?? ""}
              onChange={(next) => onChange({ href: next || undefined })}
              placeholder="link URL, e.g. {{ prUrl }} or https://github.com/org/repo/pull/{{ prNumber }}"
              inputSize="md"
              showValuePreview
              data-testid="table-column-href"
            />
          </div>
        ) : null}
        {col.format === "progress" ? (
          <ProgressFormatFields col={col} sampleRow={sampleRow} onChange={onChange} />
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
  sampleRow,
  onChange,
  onRemove,
}: {
  filter: WidgetTableFilter;
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<WidgetTableFilter>) => void;
  onRemove: () => void;
}) {
  const needsValue = filter.op !== "exists" && filter.op !== "not_exists";
  return (
    <div className="grid grid-cols-12 items-start gap-2 rounded border border-slate-200 p-2 dark:border-gray-600">
      <div className="col-span-12">
        <ExpressionEditor
          dialect="cel"
          syntaxProfile="pathOrRaw"
          exampleObj={sampleRow}
          value={filter.field}
          onChange={(next) => onChange({ field: next })}
          placeholder="field (e.g. payload.user_id)"
          inputSize="md"
          showValuePreview
        />
      </div>
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
        <div className="col-span-8">
          <ExpressionEditor
            dialect="cel"
            syntaxProfile="wrapped"
            exampleObj={sampleRow}
            value={filter.value ?? ""}
            onChange={(next) => onChange({ value: next })}
            placeholder="value or {{ expr }}"
            inputSize="md"
            showValuePreview
          />
        </div>
      ) : (
        <div className="col-span-8" />
      )}
      <Button type="button" size="icon" variant="ghost" className="col-span-1 h-8 w-8" onClick={onRemove}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

export function RowStyleRow({
  rule,
  sampleRow,
  onChange,
  onRemove,
}: {
  rule: WidgetRowStyle;
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<WidgetRowStyle>) => void;
  onRemove: () => void;
}) {
  const needsValue = rule.op !== "exists" && rule.op !== "not_exists";
  return (
    <div className="grid grid-cols-12 items-start gap-2 rounded border border-slate-200 p-2 dark:border-gray-600">
      <div className="col-span-12">
        <ExpressionEditor
          dialect="cel"
          syntaxProfile="pathOrRaw"
          exampleObj={sampleRow}
          value={rule.field}
          onChange={(next) => onChange({ field: next })}
          placeholder="field (e.g. payload.user_id)"
          inputSize="md"
          showValuePreview
          data-testid="table-row-style-field"
        />
      </div>
      <Select value={rule.op} onValueChange={(v) => onChange({ op: v as WidgetTableFilter["op"] })}>
        <SelectTrigger className="col-span-3 h-8" data-testid="table-row-style-op">
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
        <div className="col-span-5">
          <ExpressionEditor
            dialect="cel"
            syntaxProfile="wrapped"
            exampleObj={sampleRow}
            value={rule.value ?? ""}
            onChange={(next) => onChange({ value: next })}
            placeholder="value or {{ expr }}"
            inputSize="md"
            showValuePreview
            data-testid="table-row-style-value"
          />
        </div>
      ) : (
        <div className="col-span-5" />
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
