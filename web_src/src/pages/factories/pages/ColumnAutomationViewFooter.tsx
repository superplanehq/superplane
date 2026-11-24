import { useState } from "react";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";

export function ColumnAutomationViewFooter({
  confirmDelete,
  deletePending,
  onDelete,
  onConfirmDelete,
  onActionsSlot,
  savePending = false,
}: {
  confirmDelete: boolean;
  deletePending: boolean;
  onDelete: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onActionsSlot: (slot: HTMLDivElement | null) => void;
  savePending?: boolean;
}) {
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string>();
  const pending = deletePending || deleting;
  const remove = async () => {
    if (pending || savePending) return;
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await onDelete();
    } catch (error) {
      setDeleteError(getApiErrorMessage(error, "Could not delete automation. Try again."));
    } finally {
      setDeleting(false);
    }
  };
  return (
    <footer
      className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3"
      data-testid="column-automation-view-footer"
    >
      {confirmDelete ? (
        <>
          <p className="workspace-body-text text-destructive" role="alert">
            {deleteError ?? COLUMN_AUTOMATIONS_COPY.confirmDelete}
          </p>
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" size="sm" disabled={pending} onClick={() => onConfirmDelete(false)}>
              {COLUMN_AUTOMATIONS_COPY.keepLabel}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={pending || savePending}
              onClick={() => void remove()}
              data-testid="column-automation-view-delete-confirm"
            >
              {pending ? COLUMN_AUTOMATIONS_COPY.deletingLabel : COLUMN_AUTOMATIONS_COPY.deleteLabel}
            </Button>
          </div>
        </>
      ) : (
        <>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={pending || savePending}
            onClick={() => onConfirmDelete(true)}
            data-testid="column-automation-view-delete"
          >
            {COLUMN_AUTOMATIONS_COPY.deleteLabel}
          </Button>
          <fieldset disabled={pending} className="flex items-center gap-3">
            <div ref={onActionsSlot} className="flex items-center gap-3" />
          </fieldset>
        </>
      )}
    </footer>
  );
}
