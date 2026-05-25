import { useMemo, useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { DashboardPanel } from "@/hooks/useCanvasData";

import { useCanvasMemoryEntries } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext } from "./DashboardContext";
import { NumberPanelForm } from "./NumberPanelForm";
import {
  isCompositeMemoryDataSource,
  type CompositeMemoryNumberDataSource,
  type NumberPanelContent,
} from "./panelTypes";
import { useWidgetData } from "./widget/useWidgetData";
import { WidgetNumber } from "./widget/WidgetNumber";

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
  const composite = useMemo(
    () => ({
      entries: memoryQuery.data ?? [],
      sources: dataSource.sources,
      combine: dataSource.combine,
    }),
    [memoryQuery.data, dataSource.sources, dataSource.combine],
  );
  if (memoryQuery.error) return <PanelError message={String(memoryQuery.error)} />;
  return <WidgetNumber render={content.render} rows={[]} isLoading={memoryQuery.isLoading} composite={composite} />;
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
