import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { useState } from "react";

import {
  INTAKE_SETTINGS_COPY,
  intakeSupportsDelete,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function IntakeSourceSettingsFooter({
  sourceId,
  draft,
  savePending,
  saveError,
  deletePending,
  deleteError,
  onDelete,
  onSave,
  onClose,
}: {
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  deletePending: boolean;
  deleteError?: string;
  onDelete?: () => Promise<void> | void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  const deleteControls = intakeSupportsDelete(sourceId);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const footerError = saveError ?? (deleteOpen ? undefined : deleteError);

  return (
    <>
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {deleteControls ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={deletePending}
              onClick={() => setDeleteOpen(true)}
              data-testid="intake-source-settings-delete"
            >
              {INTAKE_SETTINGS_COPY.delete}
            </Button>
          ) : null}
          {footerError ? (
            <p className="workspace-body-text text-destructive" role="alert">
              {footerError}
            </p>
          ) : null}
        </div>
        <Button
          type="button"
          disabled={savePending}
          onClick={async () => {
            try {
              await onSave(normalizeIntakeSourceSettings(draft));
              onClose();
            } catch {
              // The parent supplies the actionable error message.
            }
          }}
          data-testid="intake-source-settings-save"
        >
          {savePending ? INTAKE_SETTINGS_COPY.saving : INTAKE_SETTINGS_COPY.save}
        </Button>
      </footer>
      {deleteControls ? (
        <IntakeDeleteConfirmDialog
          open={deleteOpen}
          pending={deletePending}
          error={deleteError}
          onOpenChange={setDeleteOpen}
          onConfirm={onDelete}
        />
      ) : null}
    </>
  );
}

function IntakeDeleteConfirmDialog({
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onConfirm?: () => Promise<void> | void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent data-testid="intake-delete-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{INTAKE_SETTINGS_COPY.deleteTitle}</AlertDialogTitle>
          <AlertDialogDescription>{INTAKE_SETTINGS_COPY.deleteDescription}</AlertDialogDescription>
        </AlertDialogHeader>
        {error ? (
          <p className="workspace-body-text text-destructive" role="alert" data-testid="intake-delete-error">
            {error}
          </p>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel data-testid="intake-delete-cancel">{INTAKE_SETTINGS_COPY.deleteCancel}</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            onClick={(event) => {
              event.preventDefault();
              void (async () => {
                try {
                  await onConfirm?.();
                  onOpenChange(false);
                } catch {
                  // The parent supplies the actionable error message.
                }
              })();
            }}
            data-testid="intake-delete-confirm"
          >
            {INTAKE_SETTINGS_COPY.deleteConfirm}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
