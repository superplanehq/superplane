import { useMemo, useState } from "react";
import { AlertTriangle } from "lucide-react";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { useCanvasMemoryEntries } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useConsoleContext } from "./ConsoleContext";
import { NumberPanelForm } from "./NumberPanelForm";
import {
  isCompositeMemoryDataSource,
  isMultiNumberContent,
  type CompositeMemoryNumberDataSource,
  type NumberMetric,
  type NumberPanelContent,
  type TablePanelDataSource,
} from "./panelTypes";
import { renderNeedsRunNodeOutputs, useWidgetData } from "./widget/useWidgetData";
import { WidgetNumber } from "./widget/WidgetNumber";

interface NumberPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function NumberPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: NumberPanelCardProps) {
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
        <NumberPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<NumberPanelContent>
        open={editing}
        onOpenChange={setEditingState}
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
  const ctx = useConsoleContext();
  if (!ctx?.canvasId) return <PanelError message="Loading canvas…" />;
  if (isMultiNumberContent(content) && content.metrics) {
    return <MultiNumberPanelBody metrics={content.metrics} canvasId={ctx.canvasId} />;
  }
  const dataSource = content.dataSource;
  if (!dataSource) return <PanelError message="Configure a data source." />;
  if (isCompositeMemoryDataSource(dataSource)) {
    return <CompositeNumberPanelDataBound content={content} dataSource={dataSource} canvasId={ctx.canvasId} />;
  }
  if (!content.render) return <PanelError message="Configure aggregation." />;
  return (
    <NumberPanelDataBound
      render={content.render}
      dataSource={dataSource as Exclude<NumberPanelContent["dataSource"], CompositeMemoryNumberDataSource | undefined>}
      canvasId={ctx.canvasId}
    />
  );
}

function NumberPanelDataBound({
  render,
  dataSource,
  canvasId,
}: {
  render: NonNullable<NumberPanelContent["render"]>;
  dataSource: Exclude<NumberPanelContent["dataSource"], CompositeMemoryNumberDataSource | undefined>;
  canvasId: string;
}) {
  const { rows, isLoading, error, totalCount } = useWidgetData(canvasId, dataSource, renderNeedsRunNodeOutputs(render));
  if (error) return <PanelError message={error} />;
  return <WidgetNumber render={render} rows={rows} isLoading={isLoading} totalCount={totalCount} />;
}

function MultiNumberPanelBody({ metrics, canvasId }: { metrics: NumberMetric[]; canvasId: string }) {
  if (metrics.length === 0) {
    return <PanelError message="Add at least one number to display." />;
  }
  return (
    <div
      className="flex h-full flex-row flex-wrap content-center items-center justify-center gap-x-8 gap-y-4 p-4"
      data-testid="multi-number-panel"
    >
      {metrics.map((metric, idx) => (
        <NumberMetricItem key={idx} metric={metric} canvasId={canvasId} />
      ))}
    </div>
  );
}

function NumberMetricItem({ metric, canvasId }: { metric: NumberMetric; canvasId: string }) {
  const { rows, isLoading, error, totalCount } = useWidgetData(
    canvasId,
    metric.dataSource as TablePanelDataSource,
    renderNeedsRunNodeOutputs(metric.render),
  );
  if (error) {
    return (
      <div className="flex items-center justify-center gap-2 text-xs text-amber-700">
        <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
        <span>{error}</span>
      </div>
    );
  }
  return (
    <WidgetNumber render={metric.render} rows={rows} isLoading={isLoading} totalCount={totalCount} variant="inline" />
  );
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
  const render = content.render ?? { kind: "number" };
  return <WidgetNumber render={render} rows={[]} isLoading={memoryQuery.isLoading} composite={composite} />;
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
  if (Array.isArray(r.metrics)) {
    return {
      title: typeof r.title === "string" ? r.title : "",
      metrics: r.metrics as NumberMetric[],
    };
  }
  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: (r.dataSource as NumberPanelContent["dataSource"]) ?? { kind: "runs", limit: 100 },
    render: (r.render as NumberPanelContent["render"]) ?? {
      kind: "number",
      aggregation: "count",
    },
  };
}
