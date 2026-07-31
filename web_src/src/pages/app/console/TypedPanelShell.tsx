import { useState, type ReactNode } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { CONSOLE_PANEL_BODY_SURFACE, CONSOLE_PANEL_SHELL_SURFACE } from "./consolePanelStyles";

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
  readOnly,
  onEdit,
  onDelete,
  children,
  bodyClassName,
}: {
  title?: string;
  fallbackTitle: string;
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
      <div
        className={cn(
          "group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-edge-default",
          CONSOLE_PANEL_SHELL_SURFACE,
        )}
      >
        <div
          className={cn(
            "flex items-center justify-between rounded-t-lg py-1.5 pl-3 pr-1.5",
            !readOnly && "console-grid-drag-handle cursor-grab active:cursor-grabbing",
          )}
          onDoubleClick={readOnly ? undefined : onEdit}
        >
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-[13px] font-medium text-content-primary" title={displayTitle}>
              {displayTitle}
            </span>
          </div>
          {!readOnly ? (
            // `console-grid-no-drag` exempts these controls from RGL's
            // draggable area (see ConsoleView.draggableCancel).
            <div className="console-grid-no-drag -mr-0.5 flex shrink-0 items-center opacity-0 transition-opacity group-hover/panel:opacity-100">
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
                className="h-6 w-6 cursor-pointer text-content-secondary hover:text-content-primary"
                data-testid="console-edit-panel"
              >
                <Pencil className="size-3.5" />
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
                className="h-6 w-6 cursor-pointer text-content-secondary hover:bg-status-danger-subtle hover:text-status-danger"
                data-testid="console-delete-panel"
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ) : null}
        </div>
        <div
          className={cn("min-h-0 flex-1 overflow-auto rounded-b-lg", CONSOLE_PANEL_BODY_SURFACE, bodyClassName)}
          onDoubleClick={readOnly ? undefined : onEdit}
          data-testid="typed-panel-body"
        >
          {children}
        </div>
      </div>
      <Dialog open={confirmingDelete} onOpenChange={(next) => (next ? null : setConfirmingDelete(false))}>
        <DialogContent className="border-edge-default bg-surface-overlay">
          <DialogHeader>
            <DialogTitle>Delete this panel?</DialogTitle>
            <DialogDescription>
              This panel and its contents will be removed from the console. The content is not recoverable.
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
              data-testid="console-delete-confirm"
            >
              Delete panel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
