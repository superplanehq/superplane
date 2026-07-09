import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { ROW_STYLE_CLASS, ROW_STYLE_LABEL } from "./widget/rowStyles";
import {
  WIDGET_FILTER_OPS,
  WIDGET_ROW_STYLE_TONES,
  type WidgetColumnFormat,
  type WidgetRowStyle,
  type WidgetRowStyleTone,
  type WidgetTableColumn,
  type WidgetTableFilter,
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
];

export function ColumnRow({
  col,
  fieldOptions,
  onChange,
  onRemove,
}: {
  col: WidgetTableColumn;
  fieldOptions: string[];
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
            onChange({
              field: next,
              ...(known && !col.label ? { label: next } : {}),
              ...(known && col.format == null ? { format: suggestColumnFormat(next) } : {}),
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
            onChange({ format, ...(format === "link" ? {} : { href: undefined }) });
          }}
        >
          <SelectTrigger className="col-span-4 h-8">
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
          <Input
            className="col-span-12 h-8"
            value={col.href ?? ""}
            onChange={(e) => onChange({ href: e.target.value || undefined })}
            placeholder="link URL, e.g. {{ prUrl }} or https://github.com/org/repo/pull/{{ prNumber }}"
            list={fieldOptions.length > 0 ? "table-href-field-options" : undefined}
            data-testid="table-column-href"
          />
        ) : null}
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
