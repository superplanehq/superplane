import { useCallback, useEffect, useState } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { showErrorToast } from "@/lib/toast";
import type { FirstRunScreen, OnboardingPageModel } from "./useFirstRunSetupFlow";

export function useOnboardingModelTransition(
  model: Pick<OnboardingPageModel, "saving" | "finish">,
  navigation: { screen: FirstRunScreen; goToScreen: (next: FirstRunScreen) => void },
  organizationId: string,
) {
  const access = useBYOKAccess(organizationId);
  const [ticketsSaved, setTicketsSaved] = useState(false);
  const [pending, setPending] = useState(false);
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
    if (!ticketsSaved || model.saving) return;
    setTicketsSaved(false);
    setPending(true);
    void continueAfterModelAccess().finally(() => setPending(false));
  }, [ticketsSaved, model.saving, continueAfterModelAccess]);
  const screen = navigation.screen === "agent" && !access.allowed ? "complete" : navigation.screen;
  useEffect(() => {
    if (access.ready && screen !== navigation.screen) navigation.goToScreen(screen);
  }, [access.ready, screen, navigation]);
  return {
    screen,
    pending,
    skipAgentScreen: !access.allowed,
    accessError: access.error,
    afterTickets: async () => {
      setTicketsSaved(true);
    },
    continueAfterModelAccess,
  };
}
