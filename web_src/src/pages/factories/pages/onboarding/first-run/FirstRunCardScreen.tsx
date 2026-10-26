import { Button } from "@/components/ui/button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.card;

export type FirstRunCardPhase = "add-card" | "opening" | "waiting-for-card" | "card-saved" | "starting" | "plan-active";

function formatCredit(cents: number): string {
  return (cents / 100).toLocaleString("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });
}

function planLine(planName: string, trialCreditCents: number, phase: FirstRunCardPhase): string {
  if (phase === "plan-active") return copy.planActive(planName);
  if (trialCreditCents > 0) return copy.plan(planName, formatCredit(trialCreditCents));
  return copy.planWithoutCredit(planName);
}

function actionLabel(phase: FirstRunCardPhase): string {
  if (phase === "opening") return copy.opening;
  if (phase === "waiting-for-card") return copy.waitingForCard;
  if (phase === "starting") return copy.starting;
  if (phase === "card-saved" || phase === "plan-active") return copy.continue;
  return copy.addCard;
}

/**
 * The card step. The organization saves a card with Polar and then starts
 * on the default plan. There is no skip: the plan needs a saved card.
 */
export function FirstRunCardScreen({
  planName,
  trialCreditCents,
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
  phase: FirstRunCardPhase;
  error?: string;
  busy?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onAddCard: () => void;
  onContinue: () => void;
}) {
  const cardSaved = phase === "card-saved" || phase === "plan-active";
  const inProgress = phase === "opening" || phase === "waiting-for-card" || phase === "starting";
  return (
    <FirstRunShell testId="first-run-card" chrome={chrome} busy={busy} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{cardSaved ? copy.cardSaved : copy.body}</p>
        <p className="text-[15px] leading-6 text-muted-foreground">{planLine(planName, trialCreditCents, phase)}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        <Button
          type="button"
          className="min-w-40"
          disabled={busy || inProgress}
          onClick={cardSaved ? onContinue : onAddCard}
          data-testid={cardSaved ? "first-run-card-continue" : "first-run-add-card"}
        >
          {actionLabel(phase)}
        </Button>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </div>
    </FirstRunShell>
  );
}
