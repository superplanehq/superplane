import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";

import wordmark from "@/assets/superplane-wordmark.svg";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.card;

export type FirstRunCardPhase = "add-card" | "opening" | "waiting-for-card" | "card-saved" | "starting" | "plan-active";

type CardStory = { planName: string; credit: string | null; checkout: boolean; phase: FirstRunCardPhase };

type TimelineRow = { id: string; title: string; body: string; done: boolean };

function formatCredit(cents: number): string {
  return (cents / 100).toLocaleString("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });
}

function todayBody({ planName, credit, checkout, phase }: CardStory): string {
  if (phase === "plan-active") return copy.planActive(planName);
  if (phase === "card-saved" || phase === "starting") return copy.cardSaved;
  if (checkout) return copy.todayCheckout(planName);
  if (credit) return copy.todayCredit(credit);
  return copy.todayNoCharge;
}

function timelineRows(story: CardStory): TimelineRow[] {
  const { planName, credit, phase } = story;
  const cardSaved = phase === "card-saved" || phase === "starting" || phase === "plan-active";
  return [
    { id: "today", title: copy.today, body: todayBody(story), done: cardSaved },
    credit
      ? { id: "after", title: copy.afterCredit, body: copy.afterCreditBody(planName), done: false }
      : { id: "after", title: copy.afterToday, body: copy.afterTodayBody(planName), done: false },
    { id: "any-time", title: copy.anyTime, body: copy.anyTimeBody, done: false },
  ];
}

function actionLabel(phase: FirstRunCardPhase): string {
  if (phase === "opening") return copy.opening;
  if (phase === "waiting-for-card") return copy.waitingForCard;
  if (phase === "starting") return copy.starting;
  if (phase === "card-saved" || phase === "plan-active") return copy.continue;
  return copy.addCard;
}

/** The credit as an object: a gift-card tile with the amount and the plan. */
function CreditTile({ planName, credit, phase }: Pick<CardStory, "planName" | "credit" | "phase">) {
  const applied = phase === "plan-active";
  return (
    <div
      className="relative overflow-hidden rounded-xl border border-[#34322b] bg-[linear-gradient(135deg,#2a2923_0%,#181713_60%,#11110e_100%)] p-5"
      data-testid="first-run-credit-tile"
    >
      <div className="flex items-start justify-between gap-4">
        <img src={wordmark} alt="SuperPlane" className="h-[14px] w-auto opacity-80" />
        <span className="rounded-[4px] border border-[#34322b] px-2 py-[3px] font-mono text-[11px] uppercase tracking-[0.04em] text-[#9b9993]">
          {planName}
        </span>
      </div>
      <div className="mt-8">
        <p className="font-mono text-[40px] leading-none tracking-[-0.02em] text-[#eeede9]">{credit ?? planName}</p>
        <p className="mt-2 text-[13px] text-[#9b9993]">
          {credit ? (applied ? copy.tileCreditApplied : copy.tileCredit) : copy.tilePlan}
        </p>
      </div>
    </div>
  );
}

function Timeline({ rows }: { rows: TimelineRow[] }) {
  return (
    <ol className="relative space-y-5 pl-7" data-testid="first-run-card-timeline">
      <span aria-hidden className="absolute top-2 bottom-2 left-[6px] w-px bg-[#34322b]" />
      {rows.map((row) => (
        <li
          key={row.id}
          className="relative"
          data-testid={`first-run-card-${row.id}`}
          data-done={row.done || undefined}
        >
          <span
            aria-hidden
            className={cn(
              "absolute top-[3px] -left-7 inline-flex size-[13px] items-center justify-center rounded-full border",
              row.done ? "border-[#45b88b] bg-[#45b88b] text-[#11110e]" : "border-[#9b9993] bg-[#11110e]",
            )}
          >
            {row.done ? <Check className="size-[9px]" strokeWidth={3.5} /> : null}
          </span>
          <p className="text-[14px] leading-5 text-[#eeede9]">{row.title}</p>
          <p className="mt-0.5 text-[13px] leading-5 text-[#9b9993]">{row.body}</p>
        </li>
      ))}
    </ol>
  );
}

/**
 * The card step. The credit is the object on the screen, and a short
 * timeline says what happens today, after the credit, and later. The card
 * itself is saved in the Polar portal. There is no skip.
 */
export function FirstRunCardScreen({
  planName,
  trialCreditCents,
  checkout = false,
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
  phase: FirstRunCardPhase;
  error?: string;
  busy?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onAddCard: () => void;
  onContinue: () => void;
}) {
  const credit = trialCreditCents > 0 ? formatCredit(trialCreditCents) : null;
  const story: CardStory = { planName, credit, checkout, phase };
  const cardSaved = phase === "card-saved" || phase === "plan-active";
  const inProgress = phase === "opening" || phase === "waiting-for-card" || phase === "starting";
  return (
    <FirstRunShell testId="first-run-card" chrome={chrome} busy={busy} sphere={sphere} visual="preview">
      <FirstRunHeading headline={credit ? copy.headlineWithCredit(credit) : copy.headline(planName)}>
        <p>{credit ? copy.creditIntro(credit) : copy.planIntro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-7">
        <CreditTile planName={planName} credit={credit} phase={phase} />
        <Timeline rows={timelineRows(story)} />
        <div className="space-y-3">
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
          {error ? (
            <p role="alert" className="text-[13px] text-destructive">
              {error}
            </p>
          ) : null}
        </div>
      </div>
    </FirstRunShell>
  );
}
