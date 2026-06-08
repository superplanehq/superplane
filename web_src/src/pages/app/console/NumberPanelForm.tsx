import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { DataSourceForm } from "./DataSourceForm";
import { useConsoleContext } from "./ConsoleContext";
import { NumberPanelCompositeSourcesEditor } from "./NumberPanelCompositeSourcesEditor";
import { NumberPanelMetricsEditor } from "./NumberPanelMetricsEditor";
import { NumberPanelSourceModeToggle } from "./NumberPanelSourceModeToggle";
import {
  NumberFormatField,
  NumberLabelField,
  NumberPrefixSuffixFields,
  NumberSparklineField,
} from "./NumberRenderFields";
import { NUMBER_PANEL_AGGREGATIONS } from "./numberPanelFormConstants";
import { isCompositeMemoryDataSource, isMultiNumberContent, type NumberPanelContent } from "./panelTypes";
import type { WidgetNumberAggregation, WidgetNumberRender } from "./widget/types";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

export function NumberPanelForm({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
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
      {isMultiNumberContent(value) ? (
        <NumberPanelMetricsEditor value={value} onChange={onChange} />
      ) : (
        <SingleOrCompositeBody value={value} onChange={onChange} />
      )}
    </div>
  );
}

function SingleOrCompositeBody({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const dataSource = value.dataSource;

  if (dataSource && isCompositeMemoryDataSource(dataSource)) {
    return (
      <>
        <NumberPanelCompositeSourcesEditor value={value} dataSource={dataSource} onChange={onChange} />
        <FormatLabelRow value={value} onChange={onChange} />
        <PrefixSuffixRow value={value} onChange={onChange} />
      </>
    );
  }

  return (
    <>
      {dataSource ? (
        <DataSourceForm value={dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
      ) : null}
      <SimpleAggregationFields value={value} onChange={onChange} />
      <FormatLabelRow value={value} onChange={onChange} />
      <PrefixSuffixRow value={value} onChange={onChange} />
      <SparklineField value={value} onChange={onChange} />
    </>
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
  const ctx = useConsoleContext();
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
  const update = (patch: Partial<WidgetNumberRender>) => onChange({ ...value, render: { ...render, ...patch } });
  return (
    <div className="grid grid-cols-2 gap-3">
      <NumberFormatField render={render} onChange={update} />
      <NumberLabelField render={render} onChange={update} />
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
    <NumberPrefixSuffixFields
      render={render}
      onChange={(patch) => onChange({ ...value, render: { ...render, ...patch } })}
    />
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
    <NumberSparklineField
      render={render}
      onChange={(patch) => onChange({ ...value, render: { ...render, ...patch } })}
    />
  );
}
