import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";

import { SettingsAutomationCanvas } from "./SettingsAutomationCanvas";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  appendUniqueTrimmedString,
  normalizePRFeedbackDraft,
  prFeedbackDraftIsValid,
  type PRFeedbackDraftSettings,
} from "./prFeedbackSettingsModel";

export function PRFeedbackSettingsFooter({
  draft,
  pendingCheckName,
  confirmDelete,
  savePending,
  deletePending,
  saveError,
  onSave,
  onDelete,
  onConfirmDelete,
  onClose,
}: {
  draft: PRFeedbackDraftSettings;
  pendingCheckName: string;
  confirmDelete: boolean;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onClose: () => void;
}) {
  if (confirmDelete) {
    return (
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <p className="workspace-body-text text-destructive" role="alert">
          {PR_FEEDBACK_SETTINGS_COPY.confirmDelete}
        </p>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => onConfirmDelete(false)}>
            {PR_FEEDBACK_SETTINGS_COPY.keep}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={deletePending}
            onClick={() => void onDelete?.()}
            data-testid="pr-feedback-settings-delete-confirm"
          >
            {deletePending ? PR_FEEDBACK_SETTINGS_COPY.deleting : PR_FEEDBACK_SETTINGS_COPY.delete}
          </Button>
        </div>
      </footer>
    );
  }

  return (
    <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
      {saveError ? (
        <p className="workspace-body-text text-destructive" role="alert">
          {saveError}
        </p>
      ) : onDelete ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onConfirmDelete(true)}
          data-testid="pr-feedback-settings-delete"
        >
          {PR_FEEDBACK_SETTINGS_COPY.delete}
        </Button>
      ) : (
        <span />
      )}
      <Button
        type="button"
        disabled={savePending || !prFeedbackDraftIsValid(draft)}
        onClick={async () => {
          try {
            await onSave(
              normalizePRFeedbackDraft({
                ...draft,
                checkNames: appendUniqueTrimmedString(draft.checkNames, pendingCheckName),
              }),
            );
            onClose();
          } catch {
            // The parent supplies the actionable error message.
          }
        }}
        data-testid="pr-feedback-settings-save"
      >
        {savePending ? PR_FEEDBACK_SETTINGS_COPY.saving : PR_FEEDBACK_SETTINGS_COPY.save}
      </Button>
    </footer>
  );
}

export function PRFeedbackAutomationTab({
  graph,
  title,
  editHref,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  title: string;
  editHref?: string;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <PRFeedbackAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        editHref={editHref}
        onRetry={error ? onRetry : undefined}
      />
    );
  }

  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Automation"
      data-testid="pr-feedback-automation"
    >
      <div className="flex shrink-0 items-center justify-between gap-2 px-5 pt-3 pb-2">
        <p className="min-w-0 truncate text-[15px] font-semibold tracking-[-0.02em] text-foreground">{title}</p>
        {editHref ? (
          <Link href={editHref} className={buttonVariants({ size: "sm" })}>
            {PR_FEEDBACK_SETTINGS_COPY.editAutomation}
          </Link>
        ) : null}
      </div>
      <div className="min-h-[18rem] flex-1">
        <SettingsAutomationCanvas graph={graph} />
      </div>
    </section>
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return PR_FEEDBACK_SETTINGS_COPY.automationLoading;
  }
  return error ? PR_FEEDBACK_SETTINGS_COPY.automationError : PR_FEEDBACK_SETTINGS_COPY.automationEmpty;
}

function PRFeedbackAutomationEmpty({
  message,
  editHref,
  onRetry,
}: {
  message: string;
  editHref?: string;
  onRetry?: () => void;
}) {
  return (
    <section
      className="flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
      aria-label="Automation"
      data-testid="pr-feedback-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {PR_FEEDBACK_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
      {editHref ? (
        <Link href={editHref} className={buttonVariants({ size: "sm" })}>
          {PR_FEEDBACK_SETTINGS_COPY.editAutomation}
        </Link>
      ) : null}
    </section>
  );
}
