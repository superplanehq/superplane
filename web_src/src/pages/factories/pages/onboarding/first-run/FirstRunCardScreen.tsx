import type { OrganizationsBillingPlanOption } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { ArrowRight, Check, Gift } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.card;

export type FirstRunCardPhase = "add-card" | "opening" | "waiting-for-card" | "card-saved" | "starting" | "plan-active";

function actionLabel(phase: FirstRunCardPhase, planName?: string): string {
  if (phase === "opening") return copy.opening;
  if (phase === "waiting-for-card") return copy.waitingForCard;
  if (phase === "starting") return copy.starting;
  if (phase === "plan-active" || !planName) return copy.continue;
  return copy.continueWith(planName);
}

function nextStep(phase: FirstRunCardPhase, checkout: boolean): string {
  if (phase === "plan-active") return copy.planActive;
  if (phase === "waiting-for-card") return copy.waitingDetail;
  if (phase === "card-saved") return checkout ? copy.checkoutSaved : copy.cardSaved;
  return checkout ? copy.checkoutNext : copy.cardNext;
}

function PlanChoice({
  plan,
  selected,
  disabled,
  onSelect,
}: {
  plan: OrganizationsBillingPlanOption;
  selected: boolean;
  disabled: boolean;
  onSelect: (key: string) => void;
}) {
  const price = (Number(plan.priceCents ?? 0) / 100).toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 2,
    minimumFractionDigits: 0,
  });
  return (
    <Label
      htmlFor={`onboarding-plan-${plan.key}`}
      className={cn(
        "relative flex min-w-0 cursor-pointer flex-col rounded-lg border p-4 text-foreground transition-colors focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2 focus-within:ring-offset-background dark:text-foreground",
        selected ? "border-foreground bg-muted" : "border-border hover:border-muted-foreground",
        disabled && "pointer-events-none cursor-default opacity-60",
      )}
    >
      <Input
        id={`onboarding-plan-${plan.key}`}
        type="radio"
        name="onboarding-plan"
        value={plan.key}
        checked={selected}
        disabled={disabled}
        onChange={() => plan.key && onSelect(plan.key)}
        aria-label={plan.name}
        className="sr-only"
      />
      <span className="flex items-center justify-between gap-2 text-[14px] font-medium">
        {plan.name}
        <span
          aria-hidden
          className={cn(
            "flex size-4 shrink-0 items-center justify-center rounded-full border",
            selected ? "border-foreground bg-foreground text-background" : "border-muted-foreground",
          )}
        >
          {selected && <Check className="size-3" strokeWidth={3} />}
        </span>
      </span>
      <span className="mt-3 flex items-baseline gap-1.5">
        <span className="text-[34px] font-medium leading-none tracking-[-0.04em]">{price}</span>
        <span className="text-[13px] text-muted-foreground">{copy.monthly}</span>
      </span>
      <span className="mt-3 border-t border-border pt-3 text-[13px] font-medium">
        {copy.capacity(plan.parallelTaskLimit)}
      </span>
    </Label>
  );
}

export function FirstRunCardScreen({
  plans,
  selectedPlanKey,
  welcomeCreditCents = 0,
  phase,
  error,
  busy = false,
  chrome,
  sphere,
  onSelectPlan,
  onContinue,
  onRetryPlans,
}: {
  plans: OrganizationsBillingPlanOption[];
  selectedPlanKey?: string;
  welcomeCreditCents?: number;
  phase: FirstRunCardPhase;
  error?: string;
  busy?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onSelectPlan: (key: string) => void;
  onContinue: () => void;
  onRetryPlans?: () => void;
}) {
  const plan = plans.find((candidate) => candidate.key === selectedPlanKey);
  const inProgress = ["opening", "waiting-for-card", "starting"].includes(phase);
  const cardSaved = phase === "card-saved" || phase === "plan-active";
  const disabled = busy || inProgress;
  const plansDisabled = disabled || phase === "plan-active";
  return (
    <FirstRunShell testId="first-run-card" chrome={chrome} busy={disabled} sphere={sphere} visual="preview">
      <FirstRunHeading greeting={copy.greeting} headline={copy.headline} />

      <div className="mt-5 space-y-3">
        <SignupCreditBanner remainingCents={welcomeCreditCents} />
        <fieldset disabled={plansDisabled}>
          <legend className="sr-only">{copy.headline}</legend>
          <div className="grid grid-cols-1 gap-3 min-[480px]:grid-cols-2">
            {plans.map((option) => (
              <PlanChoice
                key={option.key}
                plan={option}
                selected={option.key === selectedPlanKey}
                disabled={plansDisabled}
                onSelect={onSelectPlan}
              />
            ))}
          </div>
        </fieldset>

        <div className="space-y-2 pt-1">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={busy || (!plan && phase !== "plan-active")}
            loading={inProgress}
            loadingText={actionLabel(phase, plan?.name)}
            onClick={onContinue}
            data-testid={cardSaved ? "first-run-card-continue" : "first-run-add-card"}
          >
            {actionLabel(phase, plan?.name)}
            <ArrowRight className="size-4" aria-hidden />
          </LoadingButton>
          <p role="status" className="text-center text-[12px] leading-5 text-muted-foreground">
            {nextStep(phase, Boolean(plan?.subscriptionRequired))}
          </p>
          <PlanAvailabilityMessage
            error={error}
            noPlans={plans.length === 0}
            disabled={disabled}
            onRetry={onRetryPlans}
          />
        </div>
      </div>
    </FirstRunShell>
  );
}

function PlanAvailabilityMessage({
  error,
  noPlans,
  disabled,
  onRetry,
}: {
  error?: string;
  noPlans: boolean;
  disabled: boolean;
  onRetry?: () => void;
}) {
  if (!error && !noPlans) return null;
  return (
    <div className="space-y-2">
      <p role="alert" className="text-[13px] text-destructive">
        {error || copy.unavailable}
      </p>
      {noPlans && onRetry && (
        <Button variant="outline" className="w-full" disabled={disabled} onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
}

function SignupCreditBanner({ remainingCents }: { remainingCents: number }) {
  if (!(remainingCents >= 1000)) return null;
  return (
    <div className="flex items-start gap-3 rounded-lg border border-[#c5ebc3]/20 bg-[#c5ebc3]/[0.04] px-4 py-3.5">
      <Gift className="mt-0.5 size-4 shrink-0 text-[#c5ebc3]" aria-hidden />
      <div>
        <p className="text-[13px] font-medium text-[#c5ebc3]">{copy.credit}</p>
        <p className="mt-1 text-[12px] leading-5 text-muted-foreground">{copy.creditBody}</p>
      </div>
    </div>
  );
}
