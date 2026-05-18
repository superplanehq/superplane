import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { DashboardPanel } from "@/hooks/useCanvasData";

import { DataSourceForm } from "./DataSourceForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import type { NumberPanelContent } from "./panelTypes";
import { useWidgetData } from "./widget/useWidgetData";
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
  return <NumberPanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function NumberPanelDataBound({ content, canvasId }: { content: NumberPanelContent; canvasId: string }) {
  const { rows, isLoading, error, totalCount } = useWidgetData(canvasId, content.dataSource);
  if (error) return <PanelError message={error} />;
  return <WidgetNumber render={content.render} rows={rows} isLoading={isLoading} totalCount={totalCount} />;
}

function NumberPanelForm({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const aggregationNeedsField = value.render.aggregation !== "count";

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
      <DataSourceForm value={value.dataSource} onChange={(ds) => onChange({ ...value, dataSource: ds })} />
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">Aggregation</Label>
          <Select
            value={value.render.aggregation}
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
              value={value.render.field ?? ""}
              onChange={(e) => onChange({ ...value, render: { ...value.render, field: e.target.value } })}
              placeholder="e.g. durationMs"
            />
          </div>
        ) : null}
      </div>
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
