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
 * selected plan and continues setup. A saved card from an earlier visit skips
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
  const [selectedPlanKey, setSelectedPlanKey] = useState(
    () => new URLSearchParams(window.location.search).get("plan") ?? "pay_as_you_go",
  );
  const plan = cardStep.plans.find((candidate) => candidate.key === selectedPlanKey) ?? cardStep.plans[0];

  const startPlan = async () => {
    if (!plan?.key) return;
    setError("");
    setStarting(true);
    try {
      const returnUrl = new URL(window.location.href);
      returnUrl.searchParams.set("plan", plan.key);
      const checkout = await activateOnboardingPlan(organizationId, plan.key, {
        returnPath: `${returnUrl.pathname}${returnUrl.search}`,
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
      plans={cardStep.plans}
      selectedPlanKey={plan?.key}
      onSelectPlan={(key) => {
        setSelectedPlanKey(key);
        setError("");
      }}
      phase={cardPhase({ cardStep, starting, opening: cardSetup.opening, waitingForCard: cardSetup.waitingForCard })}
      error={error || cardSetup.error || undefined}
      busy={flow.busy}
      chrome={chrome}
      sphere={sphere}
      onContinue={() => {
        if (cardStep.planActive) {
          void flow.finishSetup();
          return;
        }
        if (cardStep.hasPaymentMethod) {
          void startPlan();
          return;
        }
        cardSetup.open(() => void startPlan());
      }}
    />
  );
}
