import { Button } from "@/components/ui/button";

import {
  PR_FEEDBACK_SETTINGS_COPY,
  normalizePRFeedbackDraft,
  prFeedbackDraftIsValid,
  type PRFeedbackDraftSettings,
} from "./prFeedbackSettingsModel";

export function PRFeedbackSettingsFooter({
  draft,
  confirmDelete,
  savePending,
  saveDisabled,
  deletePending,
  saveError,
  onSave,
  onDelete,
  onConfirmDelete,
  onClose,
}: {
  draft: PRFeedbackDraftSettings;
  confirmDelete: boolean;
  savePending?: boolean;
  saveDisabled?: boolean;
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
        disabled={savePending || saveDisabled || !prFeedbackDraftIsValid(draft)}
        onClick={async () => {
          try {
            await onSave(normalizePRFeedbackDraft(draft));
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
