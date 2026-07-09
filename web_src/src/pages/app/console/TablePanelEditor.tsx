import { PanelEditorDialog } from "./PanelEditorDialog";
import { TablePanelForm } from "./TablePanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import type { TablePanelContent } from "./panelTypes";
import type { WidgetTableRender } from "./widget/types";
import { WidgetTable } from "./widget/WidgetTable";

export interface TablePanelEditorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialContent: TablePanelContent;
  onSave: (next: TablePanelContent) => void;
  /** Sample rows the live preview renders (mockup stand-in for `useWidgetData` rows). */
  sampleRows: Record<string, unknown>[];
}

/**
 * Edit experience for the table panel, reusing the real `PanelEditorDialog`
 * (Form/YAML tabs, validation, YAML diff) and `TablePanelForm` while injecting
 * an always-on live preview above the form. Staged as a prototype: it is a
 * story-only convenience wrapper and is not wired into the panel router.
 */
export function TablePanelEditor({ open, onOpenChange, initialContent, onSave, sampleRows }: TablePanelEditorProps) {
  return (
    <PanelEditorDialog<TablePanelContent>
      open={open}
      onOpenChange={onOpenChange}
      panelId="table-editor-preview"
      panelType="table"
      initialContent={initialContent}
      onSave={onSave}
      renderForm={({ value, onChange }) => (
        <div className="flex min-w-0 flex-col gap-6">
          <TableLivePreview title={value.title} render={value.render} rows={sampleRows} />
          <TablePanelForm value={value} onChange={onChange} />
        </div>
      )}
    />
  );
}

function TableLivePreview({
  title,
  render,
  rows,
}: {
  title?: string;
  render: WidgetTableRender;
  rows: Record<string, unknown>[];
}) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <span className="text-[11px] font-medium uppercase tracking-wide text-slate-400 dark:text-gray-500">Preview</span>
      <div className="min-w-0 rounded-lg bg-slate-100 p-4 dark:bg-gray-800/50">
        {/*
         * The editor dialog uses a content-sized grid, so a wide table would
         * force the modal wider and clip. Pin the preview to the modal's own
         * inner width (dialog is min(100vw - 2rem, 48rem); subtract the dialog
         * + frame padding) so WidgetTable scrolls horizontally instead.
         */}
        <div className="h-[240px]" style={{ width: "calc(min(100vw - 2rem, 48rem) - 6rem)" }}>
          <TypedPanelShell title={title} fallbackTitle="Table" readOnly onEdit={() => {}} onDelete={() => {}}>
            <WidgetTable render={render} rows={rows} isLoading={false} />
          </TypedPanelShell>
        </div>
      </div>
    </div>
  );
}
