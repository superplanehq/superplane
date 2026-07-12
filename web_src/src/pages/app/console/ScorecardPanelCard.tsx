import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { ScorecardPanelForm } from "./ScorecardPanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import type { ScorecardPanelContent, TablePanelDataSource } from "./panelTypes";
import { renderNeedsRunNodeOutputs, runsRenderIsTotalCountOnly, useWidgetData } from "./widget/useWidgetData";
import type { WidgetScorecardRender } from "./widget/types";
import { WidgetScorecard } from "./widget/WidgetScorecard";

interface ScorecardPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function ScorecardPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: ScorecardPanelCardProps) {
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
        <ScorecardPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<ScorecardPanelContent>
        open={editing}
        onOpenChange={setEditingState}
        panelId={panel.id}
        panelType="scorecard"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <ScorecardPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function ScorecardPanelBody({ content }: { content: ScorecardPanelContent }) {
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  if (!content.dataSource) return <PanelError message="Configure a data source." />;
  if (!content.render) return <PanelError message="Configure the scorecard render." />;
  return <ScorecardPanelDataBound render={content.render} dataSource={content.dataSource} canvasId={ctx.canvasId} />;
}

function ScorecardPanelDataBound({
  render,
  dataSource,
  canvasId,
}: {
  render: WidgetScorecardRender;
  dataSource: TablePanelDataSource;
  canvasId: string;
}) {
  const { rows, isLoading, error, totalCount } = useWidgetData(
    canvasId,
    dataSource,
    renderNeedsRunNodeOutputs(render),
    false,
    runsRenderIsTotalCountOnly(dataSource, render),
  );
  if (error) return <PanelError message={error} />;
  return <WidgetScorecard render={render} rows={rows} isLoading={isLoading} totalCount={totalCount} />;
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): ScorecardPanelContent {
  const r = raw ?? {};
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as ScorecardPanelContent["dataSource"]) ?? { kind: "memory", namespace: "" },
    render: (r.render as ScorecardPanelContent["render"]) ?? {
      kind: "scorecard",
      aggregation: "last",
    },
  };
}
