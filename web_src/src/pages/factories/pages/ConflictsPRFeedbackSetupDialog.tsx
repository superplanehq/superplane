import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";

import { checksPreviewAttemptsLabel } from "./checksPRFeedbackPreview";
import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { PRFeedbackSetupPreviewPane, PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import {
  isPRFeedbackMaximumAttemptsValid,
  PR_FEEDBACK_SETTINGS_COPY,
  type PRFeedbackSource,
} from "./prFeedbackSettingsModel";

interface ConflictsPRFeedbackSetupDialogProps {
  organizationId: string;
  factoryId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}

export function ConflictsPRFeedbackSetupDialog(props: ConflictsPRFeedbackSetupDialogProps) {
  const [maximumAttempts, setMaximumAttempts] = useState(3);
  const [error, setError] = useState<string>();
  const [pending, setPending] = useState(false);
  const createHandler = useCreateFactoryPRFeedbackHandler(props.organizationId, props.factoryId);
  const attemptsValid = isPRFeedbackMaximumAttemptsValid(maximumAttempts);

  async function finish() {
    if (!attemptsValid) {
      setError(PR_FEEDBACK_SETTINGS_COPY.conflictsAttemptsHelper);
      return;
    }
    setError(undefined);
    setPending(true);
    try {
      const handler = await createHandler.mutateAsync({
        source: "SOURCE_PULL_REQUEST_CONFLICTS",
        name: props.source.defaultName,
        settings: {
          subject: props.repository ? { repository: props.repository } : undefined,
          conflicts: { maximumAttempts },
        },
      });
      if (handler.id) {
        props.onCreated(handler.id);
      }
    } catch (cause) {
      setError(getApiErrorMessage(cause, PR_FEEDBACK_SETTINGS_COPY.createError));
    } finally {
      setPending(false);
    }
  }

  return (
    <PRFeedbackSetupWizardShell
      testId="conflicts-pr-feedback-setup"
      preview={
        <PRFeedbackSetupPreviewPane
          label={PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleConflicts}
          caption={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewConflictsCaption}
          testId="conflicts-pr-feedback-preview"
          captionTestId="conflicts-pr-feedback-preview-caption"
        >
          <div className="w-full max-w-sm rounded-xl border border-border bg-background p-5 shadow-sm">
            <p className="text-sm font-medium">Merge conflict</p>
            <p className="workspace-body-text mt-2 text-muted-foreground" data-testid="conflicts-pr-feedback-attempts">
              {checksPreviewAttemptsLabel(maximumAttempts)}
            </p>
          </div>
        </PRFeedbackSetupPreviewPane>
      }
    >
      <div>
        <h1 className={factoryPageTitleClassName}>{PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleConflicts}</h1>
        <p className="workspace-body-text mt-2 text-muted-foreground">{props.source.description}</p>
      </div>
      <section>
        <Label htmlFor="conflicts-maximum-attempts">{PR_FEEDBACK_SETTINGS_COPY.wizardMaximumAttempts}</Label>
        <p className="workspace-body-text mt-1 text-muted-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.conflictsAttemptsHelper}
        </p>
        <Input
          id="conflicts-maximum-attempts"
          className="mt-2"
          type="number"
          min={1}
          max={10}
          step={1}
          value={String(maximumAttempts)}
          onChange={(event) => setMaximumAttempts(Number(event.target.value))}
          data-testid="conflicts-maximum-attempts"
        />
      </section>
      {error ? (
        <p className="workspace-body-text text-destructive" role="alert">
          {error}
        </p>
      ) : null}
      <div className="flex items-center justify-between gap-3">
        <Button type="button" variant="ghost" onClick={props.onClose}>
          {PR_FEEDBACK_SETTINGS_COPY.wizardBack}
        </Button>
        <Button
          type="button"
          onClick={() => void finish()}
          disabled={pending || !attemptsValid}
          data-testid="conflicts-setup-finish"
        >
          {pending ? PR_FEEDBACK_SETTINGS_COPY.wizardFinishing : PR_FEEDBACK_SETTINGS_COPY.wizardFinish}
        </Button>
      </div>
    </PRFeedbackSetupWizardShell>
  );
}
