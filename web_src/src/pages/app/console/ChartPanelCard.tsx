import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { ChartPanelForm } from "./ChartPanelForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import type { ChartPanelContent } from "./panelTypes";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetChart } from "./widget/WidgetChart";

interface ChartPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function ChartPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: ChartPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);
  const setEditingState = (next: boolean) => {
    setEditing(next);
    onEditingChange?.(next);
  };

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        readOnly={readOnly}
        onEdit={() => setEditingState(true)}
        onDelete={onDelete}
      >
        <ChartPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<ChartPanelContent>
        open={editing}
        onOpenChange={setEditingState}
        panelId={panel.id}
        panelType="chart"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <ChartPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function ChartPanelBody({ content }: { content: ChartPanelContent }) {
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  return <ChartPanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function ChartPanelDataBound({ content, canvasId }: { content: ChartPanelContent; canvasId: string }) {
  const { rows, isLoading, error } = useWidgetData(
    canvasId,
    content.dataSource,
    renderNeedsRunNodeOutputs(content.render),
  );
  if (error) return <PanelError message={error} />;
  return <WidgetChart render={content.render} rows={rows} isLoading={isLoading} />;
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): ChartPanelContent {
  const r = raw ?? {};
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as ChartPanelContent["dataSource"]) ?? { kind: "executions", limit: 100 },
    render: (r.render as ChartPanelContent["render"]) ?? {
      kind: "chart",
      type: "bar",
      xField: "status",
      series: [{ label: "Count" }],
    },
  };
}
