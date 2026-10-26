import { useState } from "react";

import { useBillingCardSetup } from "@/hooks/useBillingCardSetup";
import { getApiErrorMessage } from "@/lib/errors";

import { FirstRunCardScreen, type FirstRunCardPhase } from "./first-run/FirstRunCardScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { activateOnboardingPlan } from "./onboardingCardPlan";
import type { OnboardingCardStep } from "./useOnboardingCardStep";

type CardStepFlow = {
  busy: boolean;
  cardStep: OnboardingCardStep;
  finishSetup: () => Promise<unknown>;
};

function cardPhase(args: {
  cardStep: OnboardingCardStep;
  starting: boolean;
  opening: boolean;
  waitingForCard: boolean;
}): FirstRunCardPhase {
  if (args.cardStep.planActive) return "plan-active";
  if (args.starting) return "starting";
  if (args.opening) return "opening";
  if (args.waitingForCard) return "waiting-for-card";
  if (args.cardStep.hasPaymentMethod) return "card-saved";
  return "add-card";
}

/**
 * Saves a card through the Polar portal, then starts the organization on the
 * default plan and continues setup. A saved card from an earlier visit skips
 * straight to the plan.
 */
export function FirstRunCardScreenHost({
  organizationId,
  flow,
  chrome,
  sphere,
}: {
  organizationId: string;
  flow: CardStepFlow;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  const cardSetup = useBillingCardSetup(organizationId);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const { cardStep } = flow;
  const plan = cardStep.plan;

  const startPlan = async () => {
    if (!plan?.key) return;
    setError("");
    setStarting(true);
    try {
      const seats = plan.perSeat ? Math.max(cardStep.occupiedSeats ?? 1, 1) : undefined;
      const checkout = await activateOnboardingPlan(organizationId, plan.key, {
        seats,
        returnPath: `${window.location.pathname}${window.location.search}`,
      });
      if (checkout) return;
      await cardStep.refetch();
      await flow.finishSetup();
    } catch (cause) {
      setError(getApiErrorMessage(cause, "SuperPlane could not start the plan. Try again."));
    } finally {
      setStarting(false);
    }
  };

  return (
    <FirstRunCardScreen
      planName={plan?.name ?? ""}
      trialCreditCents={Number(plan?.trialCreditCents ?? 0)}
      phase={cardPhase({ cardStep, starting, opening: cardSetup.opening, waitingForCard: cardSetup.waitingForCard })}
      error={error || cardSetup.error || undefined}
      busy={flow.busy}
      chrome={chrome}
      sphere={sphere}
      onAddCard={() => cardSetup.open(() => void startPlan())}
      onContinue={() => void (cardStep.planActive ? flow.finishSetup() : startPlan())}
    />
  );
}
