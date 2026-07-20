import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TablePanelForm } from "./TablePanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import type { TablePanelContent } from "./panelTypes";
import { normalizeTablePanelContent } from "./panelTypes";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetTable } from "./widget/WidgetTable";

interface TablePanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function TablePanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: TablePanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeTablePanelContent(panel.content);
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
        <TablePanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<TablePanelContent>
        open={editing}
        onOpenChange={setEditingState}
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
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) {
    return <PanelError message="Loading canvas…" />;
  }
  return <TablePanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function TablePanelDataBound({ content, canvasId }: { content: TablePanelContent; canvasId: string }) {
  const { rows, isLoading, error, hasMore, isFetchingMore, loadMore, displayCount } = useWidgetData(
    canvasId,
    content.dataSource,
    renderNeedsRunNodeOutputs(content.render),
    true,
  );
  if (error) return <PanelError message={error} />;
  return (
    <WidgetTable
      render={content.render}
      rows={rows}
      isLoading={isLoading}
      hasMore={hasMore}
      isFetchingMore={isFetchingMore}
      onLoadMore={loadMore}
      displayCount={displayCount}
    />
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
