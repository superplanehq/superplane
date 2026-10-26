import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import type { ReactNode } from "react";

export type FirstRunStepperStep<Id extends string = string> = {
  id: Id;
  label: string;
  /** Replaces the label once the step is done, e.g. "Organization: acme". */
  doneLabel?: string;
};

export type FirstRunStepState = "done" | "active" | "upcoming";

function stepState(index: number, currentIndex: number): FirstRunStepState {
  if (index < currentIndex) return "done";
  if (index === currentIndex) return "active";
  return "upcoming";
}

function StepMark({ number, done }: { number: number; done: boolean }) {
  if (done) {
    return (
      <span
        className="inline-flex size-[22px] shrink-0 items-center justify-center rounded-[5px] border border-[#45b88b] bg-[#45b88b] text-[#11110e]"
        data-testid="first-run-step-done"
      >
        <Check className="size-3" strokeWidth={3} aria-hidden />
      </span>
    );
  }
  return (
    <span className="inline-flex size-[22px] shrink-0 items-center justify-center rounded-[5px] border border-[#34322b] bg-[#201f1a] text-[12px] text-[#9b9993]">
      {number}
    </span>
  );
}

/**
 * Sequential steps on one card. Finished steps collapse to checkmarked
 * rows, upcoming steps stay quiet, and the active step holds its content
 * and primary action inside the card.
 */
export function FirstRunStepper<Id extends string>({
  steps,
  current,
  action,
  children,
  testId = "first-run-stepper",
}: {
  steps: readonly FirstRunStepperStep<Id>[];
  current: Id;
  /** Control on the active step header, e.g. the Connect button. */
  action?: ReactNode;
  children?: ReactNode;
  testId?: string;
}) {
  const currentIndex = steps.findIndex((step) => step.id === current);
  return (
    <div className="overflow-hidden rounded-xl border border-[#34322b] text-left" data-testid={testId}>
      {steps.map((step, index) => {
        const state = stepState(index, currentIndex);
        const done = state === "done";
        const active = state === "active";
        const expanded = active && Boolean(children);
        return (
          <div
            key={step.id}
            className={cn(
              index > 0 && "border-t border-[#34322b]",
              expanded ? "flex flex-col items-start gap-3 p-4" : "flex h-[50px] items-center gap-3 px-4",
            )}
            data-testid={`first-run-step-${step.id}`}
            data-state={state}
          >
            <div
              className={cn(
                "flex min-w-0 items-center justify-between gap-3 text-[14px]",
                expanded ? "w-full" : "flex-1",
              )}
              data-testid={`first-run-step-${step.id}-header`}
            >
              <span className={cn("flex min-w-0 items-center gap-3", active ? "text-[#eeede9]" : "text-[#9b9993]")}>
                {expanded ? (
                  <span className="text-[12px] text-[#9b9993]">{index + 1}</span>
                ) : (
                  <StepMark number={index + 1} done={done} />
                )}
                <span className="truncate">{done && step.doneLabel ? step.doneLabel : step.label}</span>
              </span>
              {active ? action : null}
            </div>
            {expanded ? <div className="w-full space-y-3">{children}</div> : null}
          </div>
        );
      })}
    </div>
  );
}
