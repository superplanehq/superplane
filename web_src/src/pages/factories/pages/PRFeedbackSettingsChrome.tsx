import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";

import { SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
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
  canvasId,
  runHrefFor,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <PRFeedbackAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        onRetry={error ? onRetry : undefined}
      />
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="pr-feedback-automation"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      workflowNodes={graph.specNodes}
    />
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return PR_FEEDBACK_SETTINGS_COPY.automationLoading;
  }
  return error ? PR_FEEDBACK_SETTINGS_COPY.automationError : PR_FEEDBACK_SETTINGS_COPY.automationEmpty;
}

function PRFeedbackAutomationEmpty({ message, onRetry }: { message: string; onRetry?: () => void }) {
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
    </section>
  );
}
