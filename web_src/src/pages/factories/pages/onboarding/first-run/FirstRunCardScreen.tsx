import { LoadingButton } from "@/components/ui/loading-button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { FirstRunStepper, type FirstRunStepperStep } from "./FirstRunStepper";
import { ticketSourceName } from "./firstRunTicketIntakeSteps";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.card;

export type FirstRunCardPhase = "add-card" | "opening" | "waiting-for-card" | "card-saved" | "starting" | "plan-active";

/** What setup already has. The card step shows it as finished rows. */
export type FirstRunCardSummary = {
  repository?: string;
  ticketSource?: FirstRunTicketSource | null;
  /** Existing issues the first import takes. 0 when that import is off. */
  initialImportCount?: number;
  automaticIntake?: boolean;
};

type CardStepId = "repository" | "tickets" | "card";

function formatCredit(cents: number): string {
  return (cents / 100).toLocaleString("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });
}

function ticketsDoneLabel(summary: FirstRunCardSummary, source: FirstRunTicketSource): string {
  const name = ticketSourceName(source);
  if (summary.initialImportCount) return copy.importsNewest(summary.initialImportCount, name);
  if (summary.automaticIntake) return copy.importsNewOnly(name);
  return copy.ticketsSource(name);
}

function cardSteps(summary: FirstRunCardSummary): FirstRunStepperStep<CardStepId>[] {
  const steps: FirstRunStepperStep<CardStepId>[] = [];
  if (summary.repository) {
    steps.push({
      id: "repository",
      label: copy.stepRepository,
      doneLabel: FIRST_RUN_COPY.tickets.stepRepositoryDone(summary.repository),
    });
  }
  if (summary.ticketSource) {
    steps.push({ id: "tickets", label: copy.stepTickets, doneLabel: ticketsDoneLabel(summary, summary.ticketSource) });
  }
  steps.push({ id: "card", label: copy.stepCard });
  return steps;
}

function cardLine(phase: FirstRunCardPhase, planName: string, checkout: boolean): string {
  if (phase === "plan-active") return copy.planActive(planName);
  if (phase === "card-saved" || phase === "starting") return copy.cardSaved;
  return checkout ? copy.cardCheckout(planName) : copy.cardNoCharge;
}

function actionLabel(phase: FirstRunCardPhase): string {
  if (phase === "opening") return copy.opening;
  if (phase === "waiting-for-card") return copy.waitingForCard;
  if (phase === "starting") return copy.starting;
  if (phase === "card-saved" || phase === "plan-active") return copy.continue;
  return copy.addCard;
}

/**
 * The card step. The finished steps stay visible above the card row, so
 * the card reads as the last thing setup needs. The organization saves a
 * card with Polar and then starts on the default plan. There is no skip.
 */
export function FirstRunCardScreen({
  planName,
  trialCreditCents,
  checkout = false,
  summary = {},
  phase,
  error,
  busy,
  chrome,
  sphere,
  onAddCard,
  onContinue,
}: {
  planName: string;
  trialCreditCents: number;
  /** The plan needs a Polar checkout after the card. */
  checkout?: boolean;
  summary?: FirstRunCardSummary;
  phase: FirstRunCardPhase;
  error?: string;
  busy?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onAddCard: () => void;
  onContinue: () => void;
}) {
  const credit = trialCreditCents > 0 ? formatCredit(trialCreditCents) : null;
  const cardSaved = phase === "card-saved" || phase === "plan-active";
  const inProgress = phase === "opening" || phase === "waiting-for-card" || phase === "starting";
  return (
    <FirstRunShell testId="first-run-card" chrome={chrome} busy={busy} sphere={sphere} visual="preview">
      <FirstRunHeading
        greeting={copy.nearlyDone}
        headline={credit ? copy.headlineWithCredit(credit) : copy.headline(planName)}
      >
        {credit ? <p>{copy.creditBody(planName, credit)}</p> : null}
        <p>{checkout ? copy.checkoutBody(planName) : copy.addCardBody}</p>
      </FirstRunHeading>

      <div className="mt-8">
        <FirstRunStepper steps={cardSteps(summary)} current="card" testId="first-run-card-stepper">
          <p className="text-[13px] text-[#9b9993]">{cardLine(phase, planName, checkout)}</p>
          {error ? (
            <p role="alert" className="text-[13px] text-destructive">
              {error}
            </p>
          ) : null}
          <LoadingButton
            type="button"
            className="w-full"
            disabled={busy}
            loading={inProgress}
            loadingText={actionLabel(phase)}
            onClick={cardSaved ? onContinue : onAddCard}
            data-testid={cardSaved ? "first-run-card-continue" : "first-run-add-card"}
          >
            {actionLabel(phase)}
          </LoadingButton>
        </FirstRunStepper>
      </div>
    </FirstRunShell>
  );
}
