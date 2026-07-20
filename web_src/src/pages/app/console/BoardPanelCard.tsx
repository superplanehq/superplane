import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { BoardPanelForm } from "./BoardPanelForm";
import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import type { BoardPanelContent } from "./boardPanelContent";
import { normalizeBoardPanelContent } from "./boardPanelContent";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetBoard } from "./widget/WidgetBoard";

interface BoardPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

/**
 * Board panel: renders the same row shape as the table widget as a
 * kanban board grouped by a scalar field. Read-only for now — lane
 * membership is owned by the workflow's data, not by drag-and-drop.
 */
export function BoardPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: BoardPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeBoardPanelContent(panel.content);
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
        <BoardPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<BoardPanelContent>
        open={editing}
        onOpenChange={setEditingState}
        panelId={panel.id}
        panelType="board"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <BoardPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function BoardPanelBody({ content }: { content: BoardPanelContent }) {
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) {
    return <PanelError message="Loading canvas…" />;
  }
  return <BoardPanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function BoardPanelDataBound({ content, canvasId }: { content: BoardPanelContent; canvasId: string }) {
  const { rows, isLoading, error } = useWidgetData(
    canvasId,
    content.dataSource,
    renderNeedsRunNodeOutputs(content.render),
    true,
  );
  if (error) return <PanelError message={error} />;
  return <WidgetBoard render={content.render} rows={rows} isLoading={isLoading} />;
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}
