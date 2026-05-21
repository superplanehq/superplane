import { useState } from "react";
import { AlertTriangle, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { DashboardPanel } from "@/hooks/useCanvasData";

import { useCanvasMemoryEntries } from "@/hooks/useCanvasData";

import { DataSourceForm } from "./DataSourceForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import {
  isCompositeMemoryDataSource,
  WIDGET_NUMBER_COMBINE_OPS,
  type CompositeMemoryNumberDataSource,
  type MemoryNumberSource,
  type NumberPanelContent,
  type WidgetNumberCombine,
} from "./panelTypes";
import { useWidgetData } from "./widget/useWidgetData";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";
import { WidgetNumber } from "./widget/WidgetNumber";
import type { WidgetColumnFormat, WidgetNumberAggregation } from "./widget/types";

const AGGREGATIONS: WidgetNumberAggregation[] = ["count", "sum", "avg", "min", "max", "first", "last"];
const NUMBER_FORMATS: WidgetColumnFormat[] = ["text", "number", "percent", "duration"];

interface NumberPanelCardProps {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}

export function NumberPanelCard({ panel, readOnly, onDelete, onChange }: NumberPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        typeLabel="Number"
        readOnly={readOnly}
        onEdit={() => setEditing(true)}
        onDelete={onDelete}
      >
        <NumberPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<NumberPanelContent>
        open={editing}
        onOpenChange={setEditing}
        panelId={panel.id}
        panelType="number"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <NumberPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function NumberPanelBody({ content }: { content: NumberPanelContent }) {
  const ctx = useDashboardContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  const dataSource = content.dataSource;
  if (isCompositeMemoryDataSource(dataSource)) {
    return <CompositeNumberPanelDataBound content={content} dataSource={dataSource} canvasId={ctx.canvasId} />;
  }
  return <NumberPanelDataBound content={content} dataSource={dataSource} canvasId={ctx.canvasId} />;
}

function NumberPanelDataBound({
  content,
  dataSource,
  canvasId,
}: {
  content: NumberPanelContent;
  dataSource: Exclude<NumberPanelContent["dataSource"], CompositeMemoryNumberDataSource>;
  canvasId: string;
}) {
  const { rows, isLoading, error, totalCount } = useWidgetData(canvasId, dataSource);
  if (error) return <PanelError message={error} />;
  return <WidgetNumber render={content.render} rows={rows} isLoading={isLoading} totalCount={totalCount} />;
}

function CompositeNumberPanelDataBound({
  content,
  dataSource,
  canvasId,
}: {
  content: NumberPanelContent;
  dataSource: CompositeMemoryNumberDataSource;
  canvasId: string;
}) {
  const memoryQuery = useCanvasMemoryEntries(canvasId, true);
  if (memoryQuery.error) return <PanelError message={String(memoryQuery.error)} />;
  return (
    <WidgetNumber
      render={content.render}
      rows={[]}
      isLoading={memoryQuery.isLoading}
      composite={{
        entries: memoryQuery.data ?? [],
        sources: dataSource.sources,
        combine: dataSource.combine,
      }}
    />
  );
}

function NumberPanelForm({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
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
      <NumberSourceModeToggle value={value} onChange={onChange} />
      {isCompositeMemoryDataSource(dataSource) ? (
        <CompositeMemorySourcesEditor value={value} dataSource={dataSource} onChange={onChange} />
      ) : (
        <>
          <DataSourceForm value={dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
          <SimpleAggregationFields value={value} onChange={onChange} />
        </>
      )}
      <FormatLabelRow value={value} onChange={onChange} />
      <PrefixSuffixRow value={value} onChange={onChange} />
      {isCompositeMemoryDataSource(dataSource) ? null : <SparklineField value={value} onChange={onChange} />}
    </div>
  );
}

/**
 * Switch between the single-namespace shape and the composite shape that lets
 * each namespace declare its own aggregation. Switching to composite seeds
 * one source from the current render config so the user doesn't lose context.
 */
function NumberSourceModeToggle({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const dataSource = value.dataSource;
  const composite = isCompositeMemoryDataSource(dataSource);

  const switchToComposite = () => {
    if (isCompositeMemoryDataSource(dataSource)) return;
    const seed: MemoryNumberSource =
      dataSource.kind === "memory"
        ? {
            namespace: dataSource.namespace || "",
            aggregation: value.render.aggregation ?? "count",
            field: value.render.field,
            fieldPath: dataSource.fieldPath,
          }
        : { namespace: "", aggregation: value.render.aggregation ?? "count", field: value.render.field };
    onChange({
      ...value,
      dataSource: { kind: "memory", sources: [seed], combine: "sum" },
      render: { ...value.render, aggregation: undefined, field: undefined },
    });
  };

  const switchToSimple = () => {
    if (!isCompositeMemoryDataSource(dataSource)) return;
    const first = dataSource.sources[0];
    onChange({
      ...value,
      dataSource: { kind: "memory", namespace: first?.namespace ?? "", fieldPath: first?.fieldPath },
      render: { ...value.render, aggregation: first?.aggregation ?? "count", field: first?.field },
    });
  };

  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Source mode</Label>
      <div className="flex gap-1">
        <Button
          type="button"
          size="sm"
          variant={composite ? "outline" : "secondary"}
          onClick={switchToSimple}
          data-testid="number-mode-simple"
        >
          Single source
        </Button>
        <Button
          type="button"
          size="sm"
          variant={composite ? "secondary" : "outline"}
          onClick={switchToComposite}
          data-testid="number-mode-composite"
        >
          Multiple memory sources
        </Button>
      </div>
    </div>
  );
}

function SimpleAggregationFields({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const memoryNamespace =
    value.dataSource.kind === "memory" && "namespace" in value.dataSource ? value.dataSource.namespace : undefined;
  const { fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const aggregation = value.render.aggregation ?? "count";
  const aggregationNeedsField = aggregation !== "count";
  const fieldListId = memoryNamespace ? `number-simple-fields-${memoryNamespace}` : undefined;

  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Aggregation</Label>
        <Select
          value={aggregation}
          onValueChange={(v) =>
            onChange({ ...value, render: { ...value.render, aggregation: v as WidgetNumberAggregation } })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {AGGREGATIONS.map((a) => (
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
            value={value.render.field ?? ""}
            onChange={(e) => onChange({ ...value, render: { ...value.render, field: e.target.value } })}
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
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Format</Label>
        <Select
          value={value.render.format ?? "__none__"}
          onValueChange={(v) =>
            onChange({
              ...value,
              render: { ...value.render, format: v === "__none__" ? undefined : (v as WidgetColumnFormat) },
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Default" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Default</SelectItem>
            {NUMBER_FORMATS.map((f) => (
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
          value={value.render.label ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...value.render, label: e.target.value || undefined } })}
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
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Prefix (optional)</Label>
        <Input
          value={value.render.prefix ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...value.render, prefix: e.target.value || undefined } })}
          placeholder="e.g. R$"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Suffix (optional)</Label>
        <Input
          value={value.render.suffix ?? ""}
          onChange={(e) => onChange({ ...value, render: { ...value.render, suffix: e.target.value || undefined } })}
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
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Sparkline field (optional)</Label>
      <Input
        value={value.render.sparklineField ?? ""}
        onChange={(e) =>
          onChange({
            ...value,
            render: { ...value.render, sparklineField: e.target.value || undefined },
          })
        }
        placeholder="e.g. createdAt"
      />
    </div>
  );
}

/**
 * Composite memory editor: a Combine selector plus one row per source, where
 * each row carries its own namespace, aggregation, and field. Namespace
 * suggestions come from the canvas memory catalog so users can compose values
 * across heterogeneous schemas (e.g. sum of cost + count of tests).
 */
function CompositeMemorySourcesEditor({
  value,
  dataSource,
  onChange,
}: {
  value: NumberPanelContent;
  dataSource: CompositeMemoryNumberDataSource;
  onChange: (next: NumberPanelContent) => void;
}) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const { namespaces } = useMemoryCatalog(canvasId);
  const namespaceOptions = namespaces.map((n) => n.namespace);

  const updateDataSource = (next: CompositeMemoryNumberDataSource) => {
    onChange({ ...value, dataSource: next });
  };

  const updateSource = (idx: number, patch: Partial<MemoryNumberSource>) => {
    const sources = dataSource.sources.map((source, i) => (i === idx ? { ...source, ...patch } : source));
    updateDataSource({ ...dataSource, sources });
  };

  const addSource = () => {
    updateDataSource({
      ...dataSource,
      sources: [...dataSource.sources, { namespace: "", aggregation: "count" }],
    });
  };

  const removeSource = (idx: number) => {
    if (dataSource.sources.length <= 1) return;
    updateDataSource({ ...dataSource, sources: dataSource.sources.filter((_, i) => i !== idx) });
  };

  return (
    <div className="space-y-3 rounded-md border border-slate-200 bg-slate-50/40 p-3">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">Combine</Label>
          <Select
            value={dataSource.combine}
            onValueChange={(v) => updateDataSource({ ...dataSource, combine: v as WidgetNumberCombine })}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {WIDGET_NUMBER_COMBINE_OPS.map((op) => (
                <SelectItem key={op} value={op}>
                  {op}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Memory sources</Label>
          <Button type="button" size="sm" variant="outline" onClick={addSource} data-testid="number-add-source">
            Add source
          </Button>
        </div>
        <p className="text-[11px] text-slate-500">
          Each source aggregates its own namespace independently; the partials are combined with the operator above.
        </p>
        <div className="space-y-2">
          {dataSource.sources.map((source, idx) => (
            <MemorySourceRow
              key={idx}
              source={source}
              canvasId={canvasId}
              namespaceOptions={namespaceOptions}
              onChange={(patch) => updateSource(idx, patch)}
              onRemove={dataSource.sources.length > 1 ? () => removeSource(idx) : undefined}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function MemorySourceRow({
  source,
  canvasId,
  namespaceOptions,
  onChange,
  onRemove,
}: {
  source: MemoryNumberSource;
  canvasId: string | undefined;
  namespaceOptions: string[];
  onChange: (patch: Partial<MemoryNumberSource>) => void;
  onRemove?: () => void;
}) {
  const { fields } = useMemoryCatalog(canvasId, source.namespace);
  const needsField = source.aggregation !== "count";

  return (
    <div className="space-y-2 rounded-md border border-slate-200 bg-white p-2">
      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Namespace</Label>
          <Input
            list={namespaceOptions.length > 0 ? `number-source-namespaces-${canvasId ?? ""}` : undefined}
            value={source.namespace}
            onChange={(e) => onChange({ namespace: e.target.value })}
            placeholder="e.g. african-countries"
            data-testid="number-source-namespace"
          />
          {namespaceOptions.length > 0 ? (
            <datalist id={`number-source-namespaces-${canvasId ?? ""}`}>
              {namespaceOptions.map((ns) => (
                <option key={ns} value={ns} />
              ))}
            </datalist>
          ) : null}
        </div>
        <div className="space-y-1">
          <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Aggregation</Label>
          <Select
            value={source.aggregation}
            onValueChange={(v) =>
              onChange({
                aggregation: v as WidgetNumberAggregation,
                field: v === "count" ? undefined : source.field,
              })
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {AGGREGATIONS.map((a) => (
                <SelectItem key={a} value={a}>
                  {a}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      {needsField ? (
        <div className="space-y-1">
          <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Field</Label>
          <Input
            list={fields.length > 0 ? `number-source-fields-${source.namespace}` : undefined}
            value={source.field ?? ""}
            onChange={(e) => onChange({ field: e.target.value || undefined })}
            placeholder="e.g. cost"
            data-testid="number-source-field"
          />
          {fields.length > 0 ? (
            <datalist id={`number-source-fields-${source.namespace}`}>
              {fields.map((f) => (
                <option key={f.field} value={f.field} />
              ))}
            </datalist>
          ) : null}
        </div>
      ) : null}
      {onRemove ? (
        <div className="flex justify-end">
          <Button type="button" size="sm" variant="ghost" onClick={onRemove} data-testid="number-source-remove">
            <Trash2 className="h-3.5 w-3.5" />
            <span className="sr-only">Remove source</span>
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): NumberPanelContent {
  const r = raw ?? {};
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as NumberPanelContent["dataSource"]) ?? { kind: "runs", limit: 100 },
    render: (r.render as NumberPanelContent["render"]) ?? {
      kind: "number",
      aggregation: "count",
    },
  };
}
