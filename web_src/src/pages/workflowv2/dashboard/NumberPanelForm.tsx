import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { DataSourceForm } from "./DataSourceForm";
import { useDashboardContext } from "./DashboardContext";
import { NumberPanelCompositeSourcesEditor } from "./NumberPanelCompositeSourcesEditor";
import { NumberPanelMetricsEditor } from "./NumberPanelMetricsEditor";
import { NumberPanelSourceModeToggle } from "./NumberPanelSourceModeToggle";
import { NUMBER_PANEL_AGGREGATIONS, NUMBER_PANEL_FORMATS } from "./numberPanelFormConstants";
import { isCompositeMemoryDataSource, isMultiNumberContent, type NumberPanelContent } from "./panelTypes";
import type { WidgetColumnFormat, WidgetNumberAggregation, WidgetNumberRender } from "./widget/types";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

export function NumberPanelForm({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const multi = isMultiNumberContent(value);
  const dataSource = value.dataSource;

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
      <NumberPanelSourceModeToggle value={value} onChange={onChange} />
      {multi ? (
        <NumberPanelMetricsEditor value={value} onChange={onChange} />
      ) : (
        <>
          {dataSource && isCompositeMemoryDataSource(dataSource) ? (
            <NumberPanelCompositeSourcesEditor value={value} dataSource={dataSource} onChange={onChange} />
          ) : (
            <>
              {dataSource && !isCompositeMemoryDataSource(dataSource) ? (
                <DataSourceForm value={dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
              ) : null}
              <SimpleAggregationFields value={value} onChange={onChange} />
            </>
          )}
          <FormatLabelRow value={value} onChange={onChange} />
          <PrefixSuffixRow value={value} onChange={onChange} />
          {dataSource && isCompositeMemoryDataSource(dataSource) ? null : (
            <SparklineField value={value} onChange={onChange} />
          )}
        </>
      )}
    </div>
  );
}

const EMPTY_RENDER: WidgetNumberRender = { kind: "number" };

function SimpleAggregationFields({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const dataSource = value.dataSource;
  const memoryNamespace =
    dataSource && dataSource.kind === "memory" && "namespace" in dataSource ? dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const render = value.render ?? EMPTY_RENDER;
  const aggregation = render.aggregation ?? "count";
  const aggregationNeedsField = aggregation !== "count";
  const fieldListId = memoryNamespace ? `number-simple-fields-${memoryNamespace}` : undefined;

  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Aggregation</Label>
        <Select
          value={aggregation}
          onValueChange={(v) =>
            onChange({ ...value, render: { ...render, aggregation: v as WidgetNumberAggregation } })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {NUMBER_PANEL_AGGREGATIONS.map((a) => (
              <SelectItem key={a} value={a}>
                {a}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {aggregationNeedsField ? (
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">Field</Label>
          <Input
            list={fields.length > 0 && fieldListId ? fieldListId : undefined}
            value={render.field ?? ""}
            onChange={(e) => onChange({ ...value, render: { ...render, field: e.target.value } })}
            placeholder="e.g. durationMs"
            data-testid="number-simple-field"
          />
          {fields.length > 0 && fieldListId ? (
            <datalist id={fieldListId}>
              {fields.map((f) => (
                <option key={f.field} value={f.field} />
              ))}
            </datalist>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function FormatLabelRow({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const render = value.render ?? EMPTY_RENDER;
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Format</Label>
        <Select
          value={render.format ?? "__none__"}
          onValueChange={(v) =>
            onChange({
              ...value,
              render: { ...render, format: v === "__none__" ? undefined : (v as WidgetColumnFormat) },
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Default" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Default</SelectItem>
            {NUMBER_PANEL_FORMATS.map((f) => (
              <SelectItem key={f} value={f}>
                {f}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Label (optional)</Label>
        <Input
          value={render.label ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...render, label: e.target.value || undefined } })}
          placeholder="e.g. Total duration"
        />
      </div>
    </div>
  );
}

function PrefixSuffixRow({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const render = value.render ?? EMPTY_RENDER;
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Prefix (optional)</Label>
        <Input
          value={render.prefix ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...render, prefix: e.target.value || undefined } })}
          placeholder="e.g. R$"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Suffix (optional)</Label>
        <Input
          value={render.suffix ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...render, suffix: e.target.value || undefined } })}
          placeholder="e.g. MWh"
        />
      </div>
    </div>
  );
}

function SparklineField({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const render = value.render ?? EMPTY_RENDER;
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Sparkline field (optional)</Label>
      <Input
        value={render.sparklineField ?? ""}
        onChange={(e) =>
          onChange({
            ...value,
            render: { ...render, sparklineField: e.target.value || undefined },
          })
        }
        placeholder="e.g. createdAt"
      />
    </div>
  );
}
