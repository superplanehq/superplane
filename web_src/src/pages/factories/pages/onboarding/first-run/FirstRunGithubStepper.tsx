import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import type { ReactNode } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";

const copy = FIRST_RUN_COPY.connect;

export type FirstRunGithubStep = "connect" | "organization" | "repository";

const STEP_ORDER: readonly FirstRunGithubStep[] = ["connect", "organization", "repository"];

function stepLabel(step: FirstRunGithubStep, done: boolean, organizationName?: string): string {
  if (step === "connect") return done ? copy.stepConnected : copy.connectGitHub;
  if (step === "organization") {
    return done && organizationName ? copy.stepOrganizationDone(organizationName) : copy.stepOrganization;
  }
  return copy.stepRepository;
}

function StepBadge({ number, done }: { number: number; done: boolean }) {
  if (done) {
    return (
      <span
        className="inline-flex size-5 shrink-0 items-center justify-center rounded-md bg-emerald-600 text-background"
        data-testid="first-run-step-done"
      >
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

/**
 * The three GitHub steps on one card. Finished steps collapse to checkmarked
 * rows, upcoming steps stay dim, and the active step holds its content and
 * primary action inside the card. Each connect page renders this card, so
 * the flow reads as one task even though the pages change.
 */
export function FirstRunGithubStepper({
  current,
  organizationName,
  action,
  organizationStatus,
  children,
}: {
  current: FirstRunGithubStep;
  /** Names the finished organization row, e.g. "Organization: puppies-inc". */
  organizationName?: string;
  /** Control on the active step header, e.g. the Connect button. */
  action?: ReactNode;
  /**
   * Rows that belong to the organization step on every page, e.g. an install
   * request that waits for an admin. A pending organization stays visible
   * under "Choose organization" even while another step is active.
   */
  organizationStatus?: ReactNode;
  children?: ReactNode;
}) {
  const currentIndex = STEP_ORDER.indexOf(current);
  return (
    <div className="rounded-xl border border-border bg-card text-left" data-testid="first-run-github-stepper">
      {STEP_ORDER.map((step, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        return (
          <div
            key={step}
            className={cn("px-4 py-3.5", index > 0 && "border-t border-border")}
            data-testid={`first-run-step-${step}`}
          >
            <div
              className={cn(
                "flex items-center justify-between gap-3 text-[13px]",
                active ? "font-medium text-foreground" : "text-muted-foreground",
              )}
            >
              <span className="flex items-center gap-2.5">
                <StepBadge number={index + 1} done={done} />
                {stepLabel(step, done, organizationName)}
              </span>
              {active ? action : null}
            </div>
            {step === "organization" && organizationStatus ? (
              <div className="mt-3 space-y-3">{organizationStatus}</div>
            ) : null}
            {active && children ? <div className="mt-3 space-y-3">{children}</div> : null}
          </div>
        );
      })}
    </div>
  );
}
