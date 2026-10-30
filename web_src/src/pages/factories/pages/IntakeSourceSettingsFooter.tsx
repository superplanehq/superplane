import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { cn } from "@/lib/utils";
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
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";
import type { LineIntakeSourceId } from "./lineIntakeModel";

type IntakeSourceSettingsFooterProps = {
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  canEdit?: boolean;
  getDraft: () => IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  deletePending: boolean;
  deleteError?: string;
  onDelete?: () => Promise<void> | void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
};

export function IntakeSourceSettingsFooter({
  sourceId,
  draft,
  canEdit = true,
  savePending,
  saveError,
  deletePending,
  deleteError,
  onDelete,
  getDraft,
  onSave,
  onClose,
}: IntakeSourceSettingsFooterProps) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  if (!canEdit) {
    return (
      <footer className="flex shrink-0 justify-end border-t border-border px-5 py-3">
        <Button type="button" variant="outline" onClick={onClose}>
          Close
        </Button>
      </footer>
    );
  }

  const deleteControls = intakeSupportsDelete(sourceId);
  const linearProjectsMissing = sourceId === "linear-issues" && draft.linearProjectIds.length === 0;
  const sentryProjectsMissing = sourceId === "sentry-exceptions" && draft.sentryProjectIds.length === 0;
  const projectsMissing = linearProjectsMissing || sentryProjectsMissing;
  const footerError = intakeFooterError(
    saveError,
    deleteError,
    deleteOpen,
    linearProjectsMissing,
    sentryProjectsMissing,
  );

  return (
    <>
      <footer className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-border px-5 py-3">
        <div className={cn("flex min-w-0 flex-1 items-center gap-2", sourceId === "github-issues" && "min-w-[7rem]")}>
          {deleteControls ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="hover:bg-destructive/10 hover:text-destructive dark:hover:bg-destructive/10 dark:hover:text-destructive"
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
        <IntakeSaveButtons
          sourceId={sourceId}
          draft={draft}
          savePending={savePending}
          deletePending={deletePending}
          projectsMissing={projectsMissing}
          getDraft={getDraft}
          onSave={onSave}
          onClose={onClose}
        />
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

function intakeFooterError(
  saveError: string | undefined,
  deleteError: string | undefined,
  deleteOpen: boolean,
  linearProjectsMissing: boolean,
  sentryProjectsMissing: boolean,
): string | undefined {
  return (
    saveError ??
    (deleteOpen ? undefined : deleteError) ??
    (linearProjectsMissing ? LINEAR_INTAKE_SETUP_COPY.projectsRequired : undefined) ??
    (sentryProjectsMissing ? SENTRY_INTAKE_SETUP_COPY.projectsRequired : undefined)
  );
}

function IntakeSaveButtons({
  sourceId,
  draft,
  savePending,
  deletePending,
  projectsMissing,
  getDraft,
  onSave,
  onClose,
}: Pick<
  IntakeSourceSettingsFooterProps,
  "sourceId" | "draft" | "savePending" | "deletePending" | "getDraft" | "onSave" | "onClose"
> & { projectsMissing: boolean }) {
  const layout = useOptionalFactoriesLayout();
  const ownerMissing = layout?.factory?.onboarding?.codexSubscription === true && !draft.defaultOwnerId;
  const labelsMissing = sourceId === "github-issues" && draft.filterByLabel && draft.labels.length === 0;
  return (
    <div className="ml-auto flex shrink-0 items-center gap-3">
      {sourceId === "github-issues" ? (
        <Button type="button" variant="outline" disabled={savePending || deletePending} onClick={onClose}>
          Cancel
        </Button>
      ) : null}
      <Button
        type="button"
        disabled={savePending || projectsMissing || labelsMissing || ownerMissing}
        onClick={async () => {
          try {
            await onSave(normalizeIntakeSourceSettings(getDraft(), sourceId));
            onClose();
          } catch {
            // The parent supplies the actionable error message.
          }
        }}
        data-testid="intake-source-settings-save"
      >
        {savePending
          ? INTAKE_SETTINGS_COPY.saving
          : sourceId === "github-issues"
            ? "Save changes"
            : INTAKE_SETTINGS_COPY.save}
      </Button>
    </div>
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
