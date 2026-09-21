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
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function IntakeSourceSettingsFooter({
  sourceId,
  draft,
  savePending,
  saveError,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
  onSave,
  onClose,
}: {
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  const sentryControls = sourceId === "sentry-exceptions";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const footerError = saveError ?? pauseError ?? deleteError;
  const busy = pausePending || deletePending;

  return (
    <>
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {sentryControls ? (
            <>
              {paused ? (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  title={INTAKE_SETTINGS_COPY.pauseHelper}
                  onClick={() => void onResume?.()}
                  data-testid="intake-source-settings-resume"
                >
                  {pausePending ? INTAKE_SETTINGS_COPY.resuming : INTAKE_SETTINGS_COPY.resume}
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  title={INTAKE_SETTINGS_COPY.pauseHelper}
                  onClick={() => void onPause?.()}
                  data-testid="intake-source-settings-pause"
                >
                  {pausePending ? INTAKE_SETTINGS_COPY.pausing : INTAKE_SETTINGS_COPY.pause}
                </Button>
              )}
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={busy}
                onClick={() => setDeleteOpen(true)}
                data-testid="intake-source-settings-delete"
              >
                {INTAKE_SETTINGS_COPY.delete}
              </Button>
            </>
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
      {sentryControls ? (
        <IntakeDeleteConfirmDialog
          open={deleteOpen}
          pending={deletePending}
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
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
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
