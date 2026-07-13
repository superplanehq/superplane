import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { SpotlightPanelForm } from "./SpotlightPanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import { DEFAULT_SPOTLIGHT_CONTENT, spotlightPropsFromContent, type SpotlightPanelContent } from "./spotlightContent";
import type { WidgetDataSource } from "./widget/types";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetSpotlight } from "./widget/WidgetSpotlight";

interface SpotlightPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function SpotlightPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: SpotlightPanelCardProps) {
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
        <SpotlightPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<SpotlightPanelContent>
        open={editing}
        onOpenChange={setEditingState}
        panelId={panel.id}
        panelType="spotlight"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <SpotlightPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function SpotlightPanelBody({ content }: { content: SpotlightPanelContent }) {
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  if (!content.dataSource) return <PanelError message="Configure a data source." />;
  return <SpotlightPanelDataBound content={content} canvasId={ctx.canvasId} />;
}

function SpotlightPanelDataBound({ content, canvasId }: { content: SpotlightPanelContent; canvasId: string }) {
  const needsOutputs = renderNeedsRunNodeOutputs({
    titleField: content.titleField,
    hrefField: content.hrefField,
    actorNameField: content.actorNameField,
    checksField: content.checksField,
  });
  const { rows, isLoading, error } = useWidgetData(canvasId, content.dataSource as WidgetDataSource, needsOutputs);
  if (error) return <PanelError message={error} />;
  const props = spotlightPropsFromContent(content, rows[0]);
  return <WidgetSpotlight {...props} isLoading={isLoading} />;
}

function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 p-3 text-xs text-amber-700">
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): SpotlightPanelContent {
  const r = raw ?? {};
  return {
    ...DEFAULT_SPOTLIGHT_CONTENT,
    ...r,
    title: typeof r.title === "string" ? r.title : DEFAULT_SPOTLIGHT_CONTENT.title,
    dataSource: (r.dataSource as SpotlightPanelContent["dataSource"]) ?? DEFAULT_SPOTLIGHT_CONTENT.dataSource,
  };
}
