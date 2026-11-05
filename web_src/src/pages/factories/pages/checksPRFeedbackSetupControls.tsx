import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import {
  PR_FEEDBACK_MERGE_CONFLICT_HOURS_MAX,
  PR_FEEDBACK_MERGE_CONFLICT_HOURS_MIN,
} from "./prFeedbackConflictSettings";
import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { PR_FEEDBACK_SETTINGS_COPY, type PRFeedbackSource } from "./prFeedbackSettingsModel";
import type { ChecksPRFeedbackSetupModel } from "./useChecksPRFeedbackSetup";
import { writeDeferredWorkspaceNextStep } from "./workspaceNextStepDeferral";

export function MergeConflictHoursField({ value, onChange }: { value: number; onChange: (value: number) => void }) {
  return (
    <section className="space-y-6">
      <div>
        <h2 className={factoryPageTitleClassName}>
          <label htmlFor="checks-setup-merge-conflict-hours">{PR_FEEDBACK_SETTINGS_COPY.mergeConflictHoursLabel}</label>
        </h2>
        <p className="workspace-body-text mt-2 text-muted-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.mergeConflictHoursHelper}
        </p>
      </div>
      <Input
        id="checks-setup-merge-conflict-hours"
        type="number"
        min={PR_FEEDBACK_MERGE_CONFLICT_HOURS_MIN}
        max={PR_FEEDBACK_MERGE_CONFLICT_HOURS_MAX}
        step={1}
        value={String(value)}
        onChange={(event) => onChange(Number(event.target.value))}
        data-testid="checks-setup-merge-conflict-hours"
      />
    </section>
  );
}

export function MaximumAttemptsField({ value, onChange }: { value: number; onChange: (value: number) => void }) {
  return (
    <section className="space-y-6">
      <div>
        <h2 className={factoryPageTitleClassName}>
          <label htmlFor="checks-setup-maximum-attempts">{PR_FEEDBACK_SETTINGS_COPY.wizardMaximumAttempts}</label>
        </h2>
        <p className="workspace-body-text mt-2 text-muted-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.maximumAttemptsHelper}
        </p>
      </div>
      <Input
        id="checks-setup-maximum-attempts"
        type="number"
        min={1}
        max={10}
        step={1}
        value={String(value)}
        onChange={(event) => onChange(Number(event.target.value))}
        data-testid="checks-setup-maximum-attempts"
      />
    </section>
  );
}

export function SetupFooter({
  setup,
  factoryId,
  source,
  onClose,
  onCreated,
}: {
  setup: ChecksPRFeedbackSetupModel;
  factoryId: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  if (setup.step === "checks" && setup.catalogEmpty && !setup.fixMergeConflicts) {
    return (
      <footer className="flex items-center justify-end gap-3 pt-2">
        <Button
          type="button"
          onClick={() => {
            writeDeferredWorkspaceNextStep(factoryId, "pr-checks-handler");
            onClose();
          }}
          data-testid="checks-setup-finish"
        >
          {PR_FEEDBACK_SETTINGS_COPY.wizardFinish}
        </Button>
      </footer>
    );
  }

  if (setup.step === "checks" && setup.catalogEmpty) {
    return <FinishButton setup={setup} source={source} onClose={onClose} onCreated={onCreated} />;
  }

  if (setup.step === "checks") {
    return (
      <footer className="flex items-center justify-end gap-3 pt-2">
        <Button
          type="button"
          disabled={!setup.canContinue}
          onClick={() => setup.setStep("tools")}
          data-testid="checks-setup-continue"
        >
          {PR_FEEDBACK_SETTINGS_COPY.wizardContinue}
        </Button>
      </footer>
    );
  }

  return (
    <FinishButton
      setup={setup}
      source={source}
      onClose={onClose}
      onCreated={onCreated}
      pendingLabel={PR_FEEDBACK_SETTINGS_COPY.wizardFinishing}
    />
  );
}

function FinishButton({
  setup,
  source,
  onClose,
  onCreated,
  pendingLabel,
}: {
  setup: ChecksPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
  pendingLabel?: string;
}) {
  return (
    <footer className="flex items-center justify-end gap-3 pt-2">
      <Button
        type="button"
        disabled={setup.createHandler.isPending}
        onClick={() => {
          void setup.finish(source).then((handler) => {
            if (handler?.id) {
              onCreated(handler.id);
              onClose();
            }
          });
        }}
        data-testid="checks-setup-finish"
      >
        {setup.createHandler.isPending && pendingLabel ? pendingLabel : PR_FEEDBACK_SETTINGS_COPY.wizardFinish}
      </Button>
    </footer>
  );
}
