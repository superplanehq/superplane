import { useCallback, useEffect, useState } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { showErrorToast } from "@/lib/toast";
import { useOnboardingCardStep, type OnboardingCardStep } from "./useOnboardingCardStep";
import type { FirstRunScreen, OnboardingPageModel } from "./useFirstRunSetupFlow";

export function useOnboardingModelTransition(
  model: Pick<OnboardingPageModel, "saving" | "finish" | "codexSubscriptionOffered" | "codexSubscriptionLoading">,
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
    if (!cardStep.ready || model.codexSubscriptionLoading) return;
    try {
      if (model.codexSubscriptionOffered || (await access.verify())) {
        navigation.goToScreen("agent");
        return;
      }
      await model.finish();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "SuperPlane could not verify model access. Try again.");
    }
  }, [access, navigation, model, cardStep.ready]);
  useEffect(() => {
    if (!ticketsSaved || model.saving || !cardStep.ready || model.codexSubscriptionLoading) return;
    setTicketsSaved(false);
    if (cardStep.required) {
      navigation.goToScreen("card");
      return;
    }
    if (cardStep.requiresAdmin) {
      navigation.goToScreen("complete");
      return;
    }
    setPending(true);
    void continueAfterModelAccess().finally(() => setPending(false));
  }, [
    ticketsSaved,
    model.saving,
    model.codexSubscriptionLoading,
    cardStep.ready,
    cardStep.required,
    cardStep.requiresAdmin,
    navigation,
    continueAfterModelAccess,
  ]);
  const agentOffered = access.allowed || model.codexSubscriptionOffered;
  const screen = screenWithOnboardingAccess(navigation.screen, cardStep, agentOffered, model.codexSubscriptionLoading);
  useEffect(() => {
    if (screen === "card") setCardStepShown(true);
  }, [screen]);
  useEffect(() => {
    if (access.ready && !model.codexSubscriptionLoading && screen !== navigation.screen) navigation.goToScreen(screen);
  }, [access.ready, model.codexSubscriptionLoading, screen, navigation]);
  return {
    screen,
    pending:
      pending ||
      (["agent", "complete"].includes(navigation.screen) && !cardStep.ready && !cardStep.failed) ||
      ((ticketsSaved || navigation.screen === "agent" || navigation.screen === "complete") &&
        model.codexSubscriptionLoading),
    skipAgentScreen: !agentOffered,
    hasCardStep: cardStep.required || cardStepShown,
    cardStep,
    accessError: access.error,
    afterTickets: async () => {
      setTicketsSaved(true);
    },
    continueAfterModelAccess,
  };
}

function screenWithOnboardingAccess(
  screen: FirstRunScreen,
  cardStep: OnboardingCardStep,
  agentOffered: boolean,
  codexSubscriptionLoading: boolean,
): FirstRunScreen {
  if ((screen === "agent" || screen === "complete") && cardStep.failed) screen = "complete";
  if ((screen === "agent" || screen === "complete") && cardStep.ready) {
    if (cardStep.required) screen = "card";
    if (cardStep.requiresAdmin) screen = "complete";
  }
  if (screen === "agent" && !agentOffered && !codexSubscriptionLoading) screen = "complete";
  if (screen === "card" && cardStep.ready && !cardStep.available) screen = "complete";
  return screen;
}
