import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { DashboardPanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TablePanelForm } from "./TablePanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import type { TablePanelContent } from "./panelTypes";
import { normalizeTablePanelContent } from "./panelTypes";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetTable } from "./widget/WidgetTable";

interface TablePanelCardProps {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}

export function TablePanelCard({ panel, readOnly, onDelete, onChange }: TablePanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeTablePanelContent(panel.content);

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
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
  const { rows, isLoading, error } = useWidgetData(
    canvasId,
    content.dataSource,
    renderNeedsRunNodeOutputs(content.render),
  );
  if (error) return <PanelError message={error} />;
  return <WidgetTable render={content.render} rows={rows} isLoading={isLoading} />;
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}
