import { useState } from "react";
import { AlertTriangle, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { DashboardPanel } from "@/hooks/useCanvasData";

import { DataSourceForm } from "./DataSourceForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import type { TablePanelContent } from "./panelTypes";
import { useWidgetData } from "./widget/useWidgetData";
import { WidgetTable } from "./widget/WidgetTable";
import type { WidgetColumnFormat, WidgetTableColumn } from "./widget/types";

const COLUMN_FORMATS: WidgetColumnFormat[] = [
  "text",
  "number",
  "percent",
  "date",
  "datetime",
  "duration",
  "status",
  "code",
  "link",
];

interface TablePanelCardProps {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}

export function TablePanelCard({ panel, readOnly, onDelete, onChange }: TablePanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        typeLabel="Table"
        readOnly={readOnly}
        onEdit={() => setEditing(true)}
        onDelete={onDelete}
      >
        <TablePanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<TablePanelContent>
        open={editing}
        onOpenChange={setEditing}
        panelId={panel.id}
        panelType="table"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <TablePanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function TablePanelBody({ content }: { content: TablePanelContent }) {
  const ctx = useDashboardContext();
  if (!ctx?.canvasId) {
    return <PanelError message="Loading canvas…" />;
  }
  return <TablePanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function TablePanelDataBound({ content, canvasId }: { content: TablePanelContent; canvasId: string }) {
  const { rows, isLoading, error } = useWidgetData(canvasId, content.dataSource);
  if (error) return <PanelError message={error} />;
  return <WidgetTable render={content.render} rows={rows} isLoading={isLoading} />;
}

function TablePanelForm({
  value,
  onChange,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}) {
  const updateColumn = (idx: number, patch: Partial<WidgetTableColumn>) => {
    const columns = value.render.columns.map((col, i) => (i === idx ? { ...col, ...patch } : col));
    onChange({ ...value, render: { ...value.render, columns } });
  };
  const addColumn = () => {
    onChange({
      ...value,
      render: { ...value.render, columns: [...value.render.columns, { field: "", label: "" }] },
    });
  };
  const removeColumn = (idx: number) => {
    const columns = value.render.columns.filter((_, i) => i !== idx);
    onChange({ ...value, render: { ...value.render, columns } });
  };

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
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Columns</Label>
          <Button type="button" size="sm" variant="outline" onClick={addColumn} data-testid="table-add-column">
            Add column
          </Button>
        </div>
        <div className="space-y-2">
          {value.render.columns.map((col, idx) => (
            <div key={idx} className="grid grid-cols-12 items-center gap-2 rounded border border-slate-200 p-2">
              <Input
                className="col-span-4 h-8"
                value={col.field}
                onChange={(e) => updateColumn(idx, { field: e.target.value })}
                placeholder="field path (e.g. status)"
                aria-label={`Column ${idx + 1} field`}
              />
              <Input
                className="col-span-4 h-8"
                value={col.label ?? ""}
                onChange={(e) => updateColumn(idx, { label: e.target.value })}
                placeholder="Header (optional)"
                aria-label={`Column ${idx + 1} header`}
              />
              <Select
                value={col.format ?? "__none__"}
                onValueChange={(v) =>
                  updateColumn(idx, { format: v === "__none__" ? undefined : (v as WidgetColumnFormat) })
                }
              >
                <SelectTrigger className="col-span-3 h-8" aria-label={`Column ${idx + 1} format`}>
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
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="col-span-1 h-8 w-8 text-slate-400 hover:text-red-600"
                onClick={() => removeColumn(idx)}
                aria-label={`Remove column ${idx + 1}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))}
          {value.render.columns.length === 0 ? (
            <p className="text-xs text-slate-500">Add at least one column to display.</p>
          ) : null}
        </div>
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

function normalizeContent(raw: Record<string, unknown> | undefined): TablePanelContent {
  const r = raw ?? {};
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as TablePanelContent["dataSource"]) ?? { kind: "executions", limit: 50 },
    render: (r.render as TablePanelContent["render"]) ?? { kind: "table", columns: [] },
  };
}
