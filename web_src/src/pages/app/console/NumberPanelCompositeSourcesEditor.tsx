import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { useConsoleContext } from "./ConsoleContext";
import { NUMBER_PANEL_AGGREGATIONS } from "./numberPanelFormConstants";
import {
  WIDGET_NUMBER_COMBINE_OPS,
  type CompositeMemoryNumberDataSource,
  type MemoryNumberSource,
  type NumberPanelContent,
  type WidgetNumberCombine,
} from "./panelTypes";
import type { WidgetNumberAggregation } from "./widget/types";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

export function NumberPanelCompositeSourcesEditor({
  value,
  dataSource,
  onChange,
}: {
  value: NumberPanelContent;
  dataSource: CompositeMemoryNumberDataSource;
  onChange: (next: NumberPanelContent) => void;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const { namespaces } = useMemoryCatalog(canvasId);
  const namespaceOptions = namespaces.map((n) => n.namespace);
  const namespaceListId = namespaceOptions.length > 0 ? `number-source-namespaces-${canvasId ?? ""}` : undefined;

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
    <div className="space-y-3 rounded-md border border-slate-200 bg-slate-50/40 p-3 dark:border-gray-800/70 dark:bg-gray-900">
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
              sourceRowId={idx}
              namespaceListId={namespaceListId}
              onChange={(patch) => updateSource(idx, patch)}
              onRemove={dataSource.sources.length > 1 ? () => removeSource(idx) : undefined}
            />
          ))}
        </div>
        {namespaceListId ? (
          <datalist id={namespaceListId}>
            {namespaceOptions.map((ns) => (
              <option key={ns} value={ns} />
            ))}
          </datalist>
        ) : null}
      </div>
    </div>
  );
}

function MemorySourceRow({
  source,
  canvasId,
  sourceRowId,
  namespaceListId,
  onChange,
  onRemove,
}: {
  source: MemoryNumberSource;
  canvasId: string | undefined;
  sourceRowId: number;
  namespaceListId: string | undefined;
  onChange: (patch: Partial<MemoryNumberSource>) => void;
  onRemove?: () => void;
}) {
  const { fields } = useMemoryCatalog(canvasId, source.namespace);
  const needsField = source.aggregation !== "count";
  const fieldListId = fields.length > 0 ? `number-source-fields-${canvasId ?? ""}-${sourceRowId}` : undefined;

  return (
    <div className="space-y-2 rounded-md border border-slate-200 bg-white p-2 dark:border-gray-800/70 dark:bg-gray-900">
      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Namespace</Label>
          <Input
            list={namespaceListId}
            value={source.namespace}
            onChange={(e) => onChange({ namespace: e.target.value })}
            placeholder="e.g. african-countries"
            data-testid="number-source-namespace"
          />
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
              {NUMBER_PANEL_AGGREGATIONS.map((a) => (
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
            list={fieldListId}
            value={source.field ?? ""}
            onChange={(e) => onChange({ field: e.target.value || undefined })}
            placeholder="e.g. cost"
            data-testid="number-source-field"
          />
          {fieldListId ? (
            <datalist id={fieldListId}>
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
