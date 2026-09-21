import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";

import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { PlanningSetupPreview } from "./PlanningSetupPreview";
import { PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import { usePlanningSetup, type PlanningSetupModel } from "./usePlanningSetup";

interface PlanningSetupDialogProps {
  organizationId: string;
  factoryId: string;
  factory: Parameters<typeof usePlanningSetup>[2];
  onClose: () => void;
  onFinished: () => void;
}

export function PlanningSetupDialog(props: PlanningSetupDialogProps) {
  const setup = usePlanningSetup(props.organizationId, props.factoryId, props.factory);
  const canLeaveStep = setup.step === "scores";

  return (
    <PRFeedbackSetupWizardShell
      testId="planning-setup"
      preview={
        <PlanningSetupPreview
          step={setup.step}
          enabled={setup.enabled}
          clarity={setup.clarity}
          confidence={setup.confidence}
        />
      }
    >
      <SetupHeader
        step={setup.step}
        onBack={() => {
          if (canLeaveStep) {
            setup.setStep("refine");
            return;
          }
          props.onClose();
        }}
      />
      <div>
        {setup.step === "refine" ? <RefineStep setup={setup} /> : <ScoresStep setup={setup} />}
        {setup.error ? (
          <p className="workspace-body-text mt-4 text-destructive" role="alert">
            {setup.error}
          </p>
        ) : null}
      </div>
      <SetupFooter setup={setup} onFinished={props.onFinished} />
    </PRFeedbackSetupWizardShell>
  );
}

function SetupHeader({ step, onBack }: { step: PlanningSetupModel["step"]; onBack: () => void }) {
  const stepTitle =
    step === "refine" ? PLANNING_SETTINGS_COPY.wizardStepRefine : PLANNING_SETTINGS_COPY.wizardStepScores;
  const stepIntro = step === "scores" ? PLANNING_SETTINGS_COPY.wizardScoresIntro : undefined;

  return (
    <header className="text-left">
      <button
        type="button"
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
        onClick={onBack}
        data-testid="planning-setup-back"
      >
        <ArrowLeft className="size-4 shrink-0" aria-hidden />
        <span>{step === "scores" ? PLANNING_SETTINGS_COPY.wizardBack : PLANNING_SETTINGS_COPY.wizardBackToBoard}</span>
      </button>
      <h1 className={factoryPageTitleClassName}>{stepTitle}</h1>
      {stepIntro ? <p className="workspace-body-text mt-2 text-muted-foreground">{stepIntro}</p> : null}
    </header>
  );
}

function RefineStep({ setup }: { setup: PlanningSetupModel }) {
  return (
    <section className="flex flex-col gap-2" role="radiogroup" aria-label={PLANNING_SETTINGS_COPY.wizardStepRefine}>
      <IntakeSettingsRadioOption
        name="planning-setup-refine"
        value="plan"
        checked={setup.enabled}
        title={PLANNING_SETTINGS_COPY.wizardRefineOption}
        helper={PLANNING_SETTINGS_COPY.wizardRefineHelper}
        onChange={() => setup.setEnabled(true)}
      />
      <IntakeSettingsRadioOption
        name="planning-setup-refine"
        value="source"
        checked={!setup.enabled}
        title={PLANNING_SETTINGS_COPY.wizardSourceOption}
        helper={PLANNING_SETTINGS_COPY.wizardSourceHelper}
        onChange={() => setup.setEnabled(false)}
      />
    </section>
  );
}

function ScoresStep({ setup }: { setup: PlanningSetupModel }) {
  return (
    <div className="space-y-5">
      <section
        className="flex flex-col gap-2"
        role="radiogroup"
        aria-label={PLANNING_SETTINGS_COPY.wizardClarityOnOption}
      >
        <IntakeSettingsRadioOption
          name="planning-setup-clarity"
          value="on"
          checked={setup.clarity}
          title={PLANNING_SETTINGS_COPY.wizardClarityOnOption}
          helper={PLANNING_SETTINGS_COPY.wizardClarityOnHelper}
          onChange={() => setup.setClarity(true)}
        />
        <IntakeSettingsRadioOption
          name="planning-setup-clarity"
          value="off"
          checked={!setup.clarity}
          title={PLANNING_SETTINGS_COPY.wizardClarityOffOption}
          helper={PLANNING_SETTINGS_COPY.wizardClarityOffHelper}
          onChange={() => setup.setClarity(false)}
        />
      </section>
      <section
        className="flex flex-col gap-2"
        role="radiogroup"
        aria-label={PLANNING_SETTINGS_COPY.wizardConfidenceOnOption}
      >
        <IntakeSettingsRadioOption
          name="planning-setup-confidence"
          value="on"
          checked={setup.confidence}
          title={PLANNING_SETTINGS_COPY.wizardConfidenceOnOption}
          helper={PLANNING_SETTINGS_COPY.wizardConfidenceOnHelper}
          onChange={() => setup.setConfidence(true)}
        />
        <IntakeSettingsRadioOption
          name="planning-setup-confidence"
          value="off"
          checked={!setup.confidence}
          title={PLANNING_SETTINGS_COPY.wizardConfidenceOffOption}
          helper={PLANNING_SETTINGS_COPY.wizardConfidenceOffHelper}
          onChange={() => setup.setConfidence(false)}
        />
      </section>
    </div>
  );
}

function SetupFooter({ setup, onFinished }: { setup: PlanningSetupModel; onFinished: () => void }) {
  const finishOnThisStep = setup.step === "scores" || !setup.enabled;

  return (
    <footer className="flex items-center justify-end gap-3 pt-2">
      {finishOnThisStep ? (
        <Button
          type="button"
          disabled={setup.saving}
          onClick={() => {
            void setup.finish().then((saved) => {
              if (saved) {
                onFinished();
              }
            });
          }}
          data-testid="planning-setup-finish"
        >
          {setup.saving ? PLANNING_SETTINGS_COPY.wizardFinishing : PLANNING_SETTINGS_COPY.wizardFinish}
        </Button>
      ) : (
        <Button type="button" onClick={() => setup.setStep("scores")} data-testid="planning-setup-continue">
          {PLANNING_SETTINGS_COPY.wizardContinue}
        </Button>
      )}
    </footer>
  );
}
