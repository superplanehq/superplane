import { useState, type ReactNode } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Reusable card shell for all typed panel kinds (node / table / chart /
 * number). Provides the header (title + edit/delete actions), the drag handle
 * wiring, and the delete confirmation flow so each card body component only
 * has to focus on rendering its specific content.
 *
 * Edit is triggered by the pencil icon or a double-click on the header;
 * delete prompts for confirmation. Both actions are hidden in read-only mode.
 */
export function TypedPanelShell({
  title,
  fallbackTitle,
  typeLabel,
  readOnly,
  onEdit,
  onDelete,
  children,
  bodyClassName,
}: {
  title?: string;
  fallbackTitle: string;
  typeLabel: string;
  readOnly: boolean;
  onEdit: () => void;
  onDelete: () => void;
  children: ReactNode;
  bodyClassName?: string;
}) {
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const displayTitle = title?.trim() || fallbackTitle;

  return (
    <>
      <div className="group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
        <div
          className={
            "flex items-center justify-between border-b border-slate-100 bg-slate-50/80 px-3 py-1.5 " +
            (readOnly ? "" : "dashboard-grid-drag-handle cursor-grab active:cursor-grabbing")
          }
          onDoubleClick={readOnly ? undefined : onEdit}
        >
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-xs font-medium text-slate-700" title={displayTitle}>
              {displayTitle}
            </span>
            <span className="hidden shrink-0 rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-500 sm:inline">
              {typeLabel}
            </span>
          </div>
          {!readOnly ? (
            // `dashboard-grid-no-drag` exempts these controls from RGL's
            // draggable area (see DashboardView.draggableCancel).
            <div className="dashboard-grid-no-drag flex items-center gap-0.5 opacity-0 transition-opacity group-hover/panel:opacity-100">
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit();
                }}
                onMouseDown={(e) => e.stopPropagation()}
                onPointerDown={(e) => e.stopPropagation()}
                aria-label="Edit panel"
                className="h-6 w-6 cursor-pointer text-slate-500 hover:text-slate-700"
                data-testid="dashboard-edit-panel"
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={(e) => {
                  e.stopPropagation();
                  setConfirmingDelete(true);
                }}
                onMouseDown={(e) => e.stopPropagation()}
                onPointerDown={(e) => e.stopPropagation()}
                aria-label="Delete panel"
                className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600"
                data-testid="dashboard-delete-panel"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ) : null}
        </div>
        <div
          className={bodyClassName ?? "min-h-0 flex-1 overflow-auto"}
          onDoubleClick={readOnly ? undefined : onEdit}
          data-testid="typed-panel-body"
        >
          {children}
        </div>
      </div>
      <Dialog open={confirmingDelete} onOpenChange={(next) => (next ? null : setConfirmingDelete(false))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete this panel?</DialogTitle>
            <DialogDescription>
              This panel and its contents will be removed from the dashboard. The content is not recoverable.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setConfirmingDelete(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                setConfirmingDelete(false);
                onDelete();
              }}
              data-testid="dashboard-delete-confirm"
            >
              Delete panel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
