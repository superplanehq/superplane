import { useCallback, useEffect, useState } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { showErrorToast } from "@/lib/toast";
import { useOnboardingCardStep } from "./useOnboardingCardStep";
import type { FirstRunScreen, OnboardingPageModel } from "./useFirstRunSetupFlow";

export function useOnboardingModelTransition(
  model: Pick<OnboardingPageModel, "saving" | "finish">,
  navigation: { screen: FirstRunScreen; goToScreen: (next: FirstRunScreen) => void },
  organizationId: string,
) {
  const access = useBYOKAccess(organizationId);
  const cardStep = useOnboardingCardStep(organizationId);
  const [ticketsSaved, setTicketsSaved] = useState(false);
  const [pending, setPending] = useState(false);
  // The step count stays stable after the card step was shown, even once the
  // plan is active and the step is no longer required.
  const [cardStepShown, setCardStepShown] = useState(false);
  const continueAfterModelAccess = useCallback(async () => {
    try {
      if (await access.verify()) {
        navigation.goToScreen("agent");
        return;
      }
      await model.finish();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "SuperPlane could not verify model access. Try again.");
    }
  }, [access, navigation, model]);
  useEffect(() => {
    if (!ticketsSaved || model.saving || !cardStep.ready) return;
    setTicketsSaved(false);
    if (cardStep.required) {
      navigation.goToScreen("card");
      return;
    }
    setPending(true);
    void continueAfterModelAccess().finally(() => setPending(false));
  }, [ticketsSaved, model.saving, cardStep.ready, cardStep.required, navigation, continueAfterModelAccess]);
  let screen = navigation.screen;
  if (screen === "agent" && !access.allowed) screen = "complete";
  if (screen === "card" && cardStep.ready && !cardStep.available) screen = "complete";
  useEffect(() => {
    if (screen === "card") setCardStepShown(true);
  }, [screen]);
  useEffect(() => {
    if (access.ready && screen !== navigation.screen) navigation.goToScreen(screen);
  }, [access.ready, screen, navigation]);
  return {
    screen,
    pending,
    skipAgentScreen: !access.allowed,
    hasCardStep: cardStep.required || cardStepShown,
    cardStep,
    accessError: access.error,
    afterTickets: async () => {
      setTicketsSaved(true);
    },
    continueAfterModelAccess,
  };
}
