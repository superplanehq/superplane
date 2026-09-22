import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";

import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { planningSetupChildStep, planningSetupParentStep, type PlanningSetupStep } from "./planningSetupCaption";
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
  const parentStep = planningSetupParentStep(setup.step);

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
          if (parentStep) {
            setup.setStep(parentStep);
            return;
          }
          props.onClose();
        }}
      />
      <div>
        <SetupStepBody setup={setup} />
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

function SetupHeader({ step, onBack }: { step: PlanningSetupStep; onBack: () => void }) {
  return (
    <header className="text-left">
      <button
        type="button"
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
        onClick={onBack}
        data-testid="planning-setup-back"
      >
        <ArrowLeft className="size-4 shrink-0" aria-hidden />
        <span>{step === "refine" ? PLANNING_SETTINGS_COPY.wizardBackToBoard : PLANNING_SETTINGS_COPY.wizardBack}</span>
      </button>
      <h1 className={factoryPageTitleClassName}>{setupStepTitle(step)}</h1>
    </header>
  );
}

function SetupStepBody({ setup }: { setup: PlanningSetupModel }) {
  if (setup.step === "refine") {
    return <RefineStep setup={setup} />;
  }
  if (setup.step === "confidence") {
    return (
      <ScoreStep
        name="planning-setup-confidence"
        checked={setup.confidence}
        ariaLabel={PLANNING_SETTINGS_COPY.wizardStepConfidence}
        onTitle={PLANNING_SETTINGS_COPY.wizardConfidenceOnOption}
        onHelper={PLANNING_SETTINGS_COPY.wizardConfidenceOnHelper}
        offTitle={PLANNING_SETTINGS_COPY.wizardConfidenceOffOption}
        offHelper={PLANNING_SETTINGS_COPY.wizardConfidenceOffHelper}
        onChange={setup.setConfidence}
      />
    );
  }
  return (
    <ScoreStep
      name="planning-setup-clarity"
      checked={setup.clarity}
      ariaLabel={PLANNING_SETTINGS_COPY.wizardStepClarity}
      onTitle={PLANNING_SETTINGS_COPY.wizardClarityOnOption}
      onHelper={PLANNING_SETTINGS_COPY.wizardClarityOnHelper}
      offTitle={PLANNING_SETTINGS_COPY.wizardClarityOffOption}
      offHelper={PLANNING_SETTINGS_COPY.wizardClarityOffHelper}
      onChange={setup.setClarity}
    />
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

function ScoreStep({
  name,
  checked,
  ariaLabel,
  onTitle,
  onHelper,
  offTitle,
  offHelper,
  onChange,
}: {
  name: string;
  checked: boolean;
  ariaLabel: string;
  onTitle: string;
  onHelper: string;
  offTitle: string;
  offHelper: string;
  onChange: (next: boolean) => void;
}) {
  return (
    <section className="flex flex-col gap-2" role="radiogroup" aria-label={ariaLabel}>
      <IntakeSettingsRadioOption
        name={name}
        value="on"
        checked={checked}
        title={onTitle}
        helper={onHelper}
        onChange={() => onChange(true)}
      />
      <IntakeSettingsRadioOption
        name={name}
        value="off"
        checked={!checked}
        title={offTitle}
        helper={offHelper}
        onChange={() => onChange(false)}
      />
    </section>
  );
}

function SetupFooter({ setup, onFinished }: { setup: PlanningSetupModel; onFinished: () => void }) {
  const nextStep = planningSetupChildStep(setup.step, setup.enabled);

  return (
    <footer className="flex items-center justify-end gap-3 pt-2">
      {nextStep ? (
        <Button type="button" onClick={() => setup.setStep(nextStep)} data-testid="planning-setup-continue">
          {PLANNING_SETTINGS_COPY.wizardContinue}
        </Button>
      ) : (
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
      )}
    </footer>
  );
}

function setupStepTitle(step: PlanningSetupStep): string {
  if (step === "refine") {
    return PLANNING_SETTINGS_COPY.wizardStepRefine;
  }
  if (step === "confidence") {
    return PLANNING_SETTINGS_COPY.wizardStepConfidence;
  }
  return PLANNING_SETTINGS_COPY.wizardStepClarity;
}
