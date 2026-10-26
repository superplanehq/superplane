import { useCallback, useEffect, useState } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { showErrorToast } from "@/lib/toast";
import type { FirstRunScreen, OnboardingPageModel } from "./useFirstRunSetupFlow";

export function useOnboardingModelTransition(
  model: Pick<OnboardingPageModel, "saving" | "finish" | "codexSubscriptionOffered" | "codexSubscriptionLoading">,
  navigation: { screen: FirstRunScreen; goToScreen: (next: FirstRunScreen) => void },
  organizationId: string,
) {
  const access = useBYOKAccess(organizationId);
  const [ticketsSaved, setTicketsSaved] = useState(false);
  const [pending, setPending] = useState(false);
  const continueAfterModelAccess = useCallback(async () => {
    if (model.codexSubscriptionLoading) return;
    try {
      if (model.codexSubscriptionOffered || (await access.verify())) {
        navigation.goToScreen("agent");
        return;
      }
      await model.finish();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "SuperPlane could not verify model access. Try again.");
    }
  }, [access, navigation, model]);
  useEffect(() => {
    if (!ticketsSaved || model.saving || model.codexSubscriptionLoading) return;
    setTicketsSaved(false);
    setPending(true);
    void continueAfterModelAccess().finally(() => setPending(false));
  }, [ticketsSaved, model.saving, model.codexSubscriptionLoading, continueAfterModelAccess]);
  const agentOffered = access.allowed || model.codexSubscriptionOffered;
  const screen =
    navigation.screen === "agent" && !agentOffered && !model.codexSubscriptionLoading ? "complete" : navigation.screen;
  useEffect(() => {
    if (access.ready && !model.codexSubscriptionLoading && screen !== navigation.screen) navigation.goToScreen(screen);
  }, [access.ready, model.codexSubscriptionLoading, screen, navigation]);
  return {
    screen,
    pending:
      pending ||
      ((ticketsSaved || navigation.screen === "agent" || navigation.screen === "complete") &&
        model.codexSubscriptionLoading),
    skipAgentScreen: !agentOffered,
    accessError: access.error,
    afterTickets: async () => {
      setTicketsSaved(true);
    },
    continueAfterModelAccess,
  };
}
