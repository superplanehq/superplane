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
  intakeDangerZoneHelper,
  intakeDeleteHelper,
  intakePauseHelper,
  intakeSettingsSectionDomId,
  intakeSupportsPause,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

/**
 * Compact danger zone for pause and delete. Sits under the main intake settings,
 * not as its own empty page.
 */
export function IntakeSettingsLifecycle({
  sourceId,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
}: {
  sourceId: LineIntakeSourceId;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
}) {
  const pauseControls = intakeSupportsPause(sourceId);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const busy = pausePending || deletePending;

  if (!pauseControls) {
    return null;
  }

  return (
    <>
      <section
        id={intakeSettingsSectionDomId("danger")}
        className="scroll-mt-6"
        aria-labelledby="intake-danger-zone-title"
        data-testid="intake-danger-zone"
      >
        <h3 id="intake-danger-zone-title" className="workspace-section-title text-destructive">
          {INTAKE_SETTINGS_COPY.dangerZone}
        </h3>
        <p className="mt-1 text-[13px] leading-5 text-muted-foreground">{intakeDangerZoneHelper(sourceId)}</p>
        <div className="mt-4 overflow-hidden rounded-lg border border-destructive/30">
          <div className="flex flex-col gap-3 border-b border-destructive/20 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
            <div className="min-w-0">
              <p className="text-[13px] font-medium text-foreground">
                {paused ? INTAKE_SETTINGS_COPY.resume : INTAKE_SETTINGS_COPY.pause}
              </p>
              <p className="mt-0.5 text-[13px] leading-5 text-muted-foreground">{intakePauseHelper(sourceId)}</p>
            </div>
            {paused ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="shrink-0"
                disabled={busy}
                onClick={() => void ignoreFailedIntakeAction(onResume)}
                data-testid="intake-source-settings-resume"
              >
                {pausePending ? INTAKE_SETTINGS_COPY.resuming : INTAKE_SETTINGS_COPY.resume}
              </Button>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="shrink-0"
                disabled={busy}
                onClick={() => void ignoreFailedIntakeAction(onPause)}
                data-testid="intake-source-settings-pause"
              >
                {pausePending ? INTAKE_SETTINGS_COPY.pausing : INTAKE_SETTINGS_COPY.pause}
              </Button>
            )}
          </div>
          {pauseError ? (
            <p className="border-b border-destructive/20 px-4 py-2 workspace-body-text text-destructive" role="alert">
              {pauseError}
            </p>
          ) : null}
          <div className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
            <div className="min-w-0">
              <p className="text-[13px] font-medium text-foreground">{INTAKE_SETTINGS_COPY.delete}</p>
              <p className="mt-0.5 text-[13px] leading-5 text-muted-foreground">{intakeDeleteHelper(sourceId)}</p>
            </div>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              className="shrink-0"
              disabled={busy}
              onClick={() => setDeleteOpen(true)}
              data-testid="intake-source-settings-delete"
            >
              {INTAKE_SETTINGS_COPY.delete}
            </Button>
          </div>
        </div>
      </section>
      <IntakeDeleteConfirmDialog
        sourceId={sourceId}
        open={deleteOpen}
        pending={deletePending}
        error={deleteError}
        onOpenChange={setDeleteOpen}
        onConfirm={onDelete}
      />
    </>
  );
}

export type IntakeSettingsSaveActionProps = {
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
  saveDisabled?: boolean;
};

/** Save action for the intake settings top bar. */
export function IntakeSettingsSaveAction({
  draft,
  savePending,
  saveError,
  onSave,
  onClose,
  saveDisabled = false,
}: IntakeSettingsSaveActionProps) {
  return (
    <>
      {saveError ? (
        <p className="max-w-xs truncate workspace-body-text text-destructive" role="alert">
          {saveError}
        </p>
      ) : null}
      <Button
        type="button"
        disabled={savePending || saveDisabled}
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
    </>
  );
}

async function ignoreFailedIntakeAction(action?: () => Promise<void> | void) {
  try {
    await action?.();
  } catch {
    // The parent supplies the actionable error message.
  }
}

function IntakeDeleteConfirmDialog({
  sourceId,
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  sourceId: LineIntakeSourceId;
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
          <AlertDialogDescription>{intakeDeleteHelper(sourceId)}</AlertDialogDescription>
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
