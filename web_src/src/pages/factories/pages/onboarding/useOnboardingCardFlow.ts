import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getApiErrorMessage } from "@/lib/errors";
import {
  activateOnboardingPlan,
  createOnboardingCardSession,
  resumeOnboardingPlan,
  type OnboardingCardSession,
} from "./onboardingCardPlan";
import type { OnboardingCardStep } from "./useOnboardingCardStep";

export type CardStepFlow = {
  busy: boolean;
  cardStep: OnboardingCardStep;
  finishSetup: () => Promise<unknown>;
};

export function useOnboardingCardFlow(organizationId: string, flow: CardStepFlow) {
  const [returnParams] = useState(() => new URLSearchParams(window.location.search));
  const [returned, setReturned] = useState(
    () => returnParams.has("card_setup") || returnParams.get("subscribed") === "1",
  );
  const [working, setWorking] = useState(returned);
  const [error, setError] = useState("");
  const [session, setSession] = useState<OnboardingCardSession>();
  const autoStarted = useRef(false);
  const checkoutReturn = useRef(returnParams.get("subscribed") === "1");
  const inFlight = useRef(false);
  const { cardStep } = flow;
  const [chosenPlanKey, setSelectedPlanKey] = useState(() => returnParams.get("plan") ?? "pay_as_you_go");
  const selectedPlanKey = cardStep.planActive ? (cardStep.activePlanKey ?? chosenPlanKey) : chosenPlanKey;
  // An organization on a plan outside the starter list, for example Team, can
  // still restore its card. The active plan key is enough to resume setup.
  const plan = useMemo(
    () =>
      cardStep.plans.find((candidate) => candidate.key === selectedPlanKey) ??
      (cardStep.planActive && selectedPlanKey ? { key: selectedPlanKey } : undefined),
    [cardStep.plans, cardStep.planActive, selectedPlanKey],
  );

  const continueSetup = useCallback(
    async (resume: boolean) => {
      if (inFlight.current || !cardStep.ready || !cardStep.available) return;
      inFlight.current = true;
      setWorking(true);
      setError("");
      try {
        if (!plan?.key) throw new Error("The selected plan is unavailable. Choose another plan.");
        const returnPath = onboardingCardReturnPath(plan.key);
        if (resume) {
          if (await resumeOnboardingPlan(organizationId, plan.key, returnPath, checkoutReturn.current)) return;
        } else if (!cardStep.planActive || cardStep.required) {
          if (!cardStep.hasPaymentMethod && (cardStep.planActive || !plan.subscriptionRequired)) {
            setSession(await createOnboardingCardSession(organizationId));
            return;
          }
          if (await activateOnboardingPlan(organizationId, plan.key, { returnPath })) return;
        }
        await cardStep.refetch();
        await flow.finishSetup();
      } catch (cause) {
        setError(getApiErrorMessage(cause, "SuperPlane could not finish setup. Try again."));
      } finally {
        inFlight.current = false;
        setWorking(false);
      }
    },
    [plan, organizationId, cardStep, flow],
  );

  useEffect(() => {
    if (!returned || !cardStep.ready || autoStarted.current) return;
    autoStarted.current = true;
    void continueSetup(true);
  }, [returned, cardStep.ready, continueSetup]);

  const cardSaved = useCallback(() => {
    autoStarted.current = false;
    checkoutReturn.current = false;
    setSession(undefined);
    setWorking(true);
    setReturned(true);
  }, []);

  return {
    session,
    returned,
    error,
    working,
    plan,
    continueSetup,
    cardSaved,
    selectPlan: (key: string) => {
      setSelectedPlanKey(key);
      setError("");
    },
    backToPlans: () => {
      setSession(undefined);
      setReturned(false);
      setError("");
    },
    retryCard: () => {
      setSession(undefined);
      void continueSetup(false);
    },
  };
}

function onboardingCardReturnPath(planKey: string): string {
  const url = new URL(window.location.href);
  url.searchParams.set("step", "card");
  url.searchParams.set("plan", planKey);
  url.searchParams.set("card_setup", "1");
  url.searchParams.delete("subscribed");
  url.searchParams.delete("checkout_id");
  return url.pathname + url.search;
}
