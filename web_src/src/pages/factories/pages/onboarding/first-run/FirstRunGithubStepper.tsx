import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import type { ReactNode } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";

const copy = FIRST_RUN_COPY.connect;

export type FirstRunGithubStep = "connect" | "organization" | "repository";

const STEP_ORDER: readonly FirstRunGithubStep[] = ["connect", "organization", "repository"];

function stepLabel(step: FirstRunGithubStep, done: boolean, organizationName?: string): string {
  if (step === "connect") return copy.connectGitHub;
  if (step === "organization") {
    return done && organizationName ? copy.stepOrganizationDone(organizationName) : copy.stepOrganization;
  }
  return copy.stepRepository;
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
 * The GitHub steps on one card. Finished steps collapse to checkmarked
 * rows, upcoming steps stay quiet, and the active step holds its content
 * and primary action inside the card.
 */
export function FirstRunGithubStepper({
  current,
  organizationName,
  action,
  children,
}: {
  current: FirstRunGithubStep;
  /** Names the finished organization row, e.g. "Organization: acme". */
  organizationName?: string;
  /** Control on the active step header, e.g. the Connect button. */
  action?: ReactNode;
  children?: ReactNode;
}) {
  const currentIndex = STEP_ORDER.indexOf(current);
  return (
    <div
      className="overflow-hidden rounded-xl border border-[#34322b] text-left"
      data-testid="first-run-github-stepper"
    >
      {STEP_ORDER.map((step, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        const expanded = active && Boolean(children);
        const state = done ? "done" : active ? "active" : "upcoming";
        return (
          <div
            key={step}
            className={cn(
              index > 0 && "border-t border-[#34322b]",
              expanded ? "flex flex-col items-start gap-3 p-4" : "flex h-[50px] items-center gap-3 px-4",
            )}
            data-testid={`first-run-step-${step}`}
            data-state={state}
          >
            <div
              className={cn(
                "flex min-w-0 items-center justify-between gap-3 text-[14px]",
                expanded ? "w-full" : "flex-1",
              )}
              data-testid={`first-run-step-${step}-header`}
            >
              <span
                className={cn("flex min-w-0 items-center gap-3", done || !active ? "text-[#9b9993]" : "text-[#eeede9]")}
              >
                {expanded ? (
                  <span className="text-[12px] text-[#9b9993]">{index + 1}</span>
                ) : (
                  <StepMark number={index + 1} done={done} />
                )}
                {stepLabel(step, done, organizationName)}
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
