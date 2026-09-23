import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import type { ReactNode } from "react";

import { FirstRunHeading, FirstRunShell } from "./onboarding/first-run/FirstRunShell";
import type { FirstRunSphereProps } from "./onboarding/first-run/FirstRunSpherePane";

export type IntakeSetupStep = "connection" | "project" | "completion";

export type IntakeSetupStepItem = { id: IntakeSetupStep; label: string };

export function IntakeSetupWizard({
  testId,
  integrationName,
  step,
  title,
  helper,
  onBack,
  children,
  footer,
  stepAction,
  steps,
  plain = false,
}: {
  testId: string;
  integrationName: string;
  step: IntakeSetupStep;
  title: string;
  helper: string;
  onBack: () => void;
  children: ReactNode;
  footer: ReactNode;
  /** Rendered on the right of the current step label, e.g. Connect. */
  stepAction?: ReactNode;
  /** Overrides the default Connect and Choose project rows. */
  steps?: IntakeSetupStepItem[];
  /** Renders the step body outside the stepper card. */
  plain?: boolean;
}) {
  const resolvedSteps = steps ?? defaultIntakeSteps(integrationName);
  const stepIndex = Math.max(
    0,
    resolvedSteps.findIndex((item) => item.id === step),
  );

  return (
    <FirstRunShell
      testId={testId}
      chrome={{
        stepIndex,
        stepCount: resolvedSteps.length,
        onBack,
      }}
      sphere={intakeSphere(integrationName, step, `${testId}-sphere`)}
    >
      <FirstRunHeading headline={title}>
        <p className="text-[15px] leading-6 text-muted-foreground">{helper}</p>
      </FirstRunHeading>
      <div className={cn("mt-8", plain ? "flex flex-col gap-6" : "space-y-4")}>
        {plain ? (
          children
        ) : (
          <IntakeSetupStepper testId={`${testId}-stepper`} steps={resolvedSteps} current={step} stepAction={stepAction}>
            {children}
          </IntakeSetupStepper>
        )}
        {footer}
      </div>
    </FirstRunShell>
  );
}

function defaultIntakeSteps(integrationName: string): IntakeSetupStepItem[] {
  return [
    { id: "connection", label: `Connect ${integrationName}` },
    { id: "project", label: "Choose project" },
  ];
}

function IntakeSetupStepper({
  testId,
  steps,
  current,
  children,
  stepAction,
}: {
  testId: string;
  steps: IntakeSetupStepItem[];
  current: IntakeSetupStep;
  children: ReactNode;
  stepAction?: ReactNode;
}) {
  const currentIndex = steps.findIndex((step) => step.id === current);

  return (
    <div className="rounded-xl border border-border bg-card text-left" data-testid={testId}>
      {steps.map((step, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        return (
          <div key={step.id} className={cn("px-4 py-3.5", index > 0 && "border-t border-border")}>
            <div
              className={cn(
                "flex items-center gap-2.5 text-[13px]",
                active ? "font-medium text-foreground" : "text-muted-foreground",
              )}
            >
              <StepBadge number={index + 1} done={done} />
              <span className="min-w-0 flex-1">{step.label}</span>
              {active && stepAction ? <div className="ml-auto shrink-0">{stepAction}</div> : null}
            </div>
            {active && children ? <div className="mt-3 space-y-3">{children}</div> : null}
          </div>
        );
      })}
    </div>
  );
}

function StepBadge({ number, done }: { number: number; done: boolean }) {
  if (done) {
    return (
      <span className="inline-flex size-5 shrink-0 items-center justify-center rounded-md bg-emerald-600 text-background">
        <Check className="size-3" strokeWidth={3} aria-hidden />
      </span>
    );
  }
  return (
    <span className="inline-flex size-5 shrink-0 items-center justify-center rounded-md border border-border bg-accent/40 text-[11px]">
      {number}
    </span>
  );
}

function intakeSphere(integrationName: string, step: IntakeSetupStep, testId: string): FirstRunSphereProps {
  if (step === "connection") {
    return sphereProps(testId, 0.24, `Awaiting ${integrationName} connection`, "Awaiting backlog");
  }
  if (step === "completion") {
    return sphereProps(testId, 0.82, "Awaiting column", `${integrationName} issues`);
  }
  return sphereProps(testId, 0.58, "Awaiting project", `${integrationName} issues`);
}

function sphereProps(testId: string, level: number, caption: string, chipValue: string): FirstRunSphereProps {
  return {
    testId,
    level,
    caption,
    leftChip: {
      label: "Discover",
      value: chipValue,
      tone: "ghost",
    },
  };
}
