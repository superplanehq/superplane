import { useCallback, useEffect, useState } from "react";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { FEATURE_NEW_BILLING } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { isOrganizationPlanActive } from "./onboardingPlanActive";
import type { FirstRunScreen, OnboardingPageModel } from "./useFirstRunSetupFlow";

export function useOnboardingPlanTransition(
  model: Pick<OnboardingPageModel, "saving" | "finish">,
  navigation: { screen: FirstRunScreen; goToScreen: (next: FirstRunScreen) => void },
  organizationId: string,
) {
  const rollout = useExperimentalFeature(organizationId);
  const access = useBYOKAccess(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const [ticketsSaved, setTicketsSaved] = useState(false);
  const [pending, setPending] = useState(false);
  // Choosing a plan activates it. Keep the step once it is shown, so it does
  // not disappear while the user is on it.
  const [billingStepShown, setBillingStepShown] = useState(false);
  const hasBillingStep =
    rollout.has(FEATURE_NEW_BILLING) &&
    billing.data?.billingEnabled === true &&
    Boolean(billing.data.availablePlans?.length) &&
    (billingStepShown || !isOrganizationPlanActive(billing.data));
  const continueAfterPlan = useCallback(async () => {
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
    const continueFromTickets = async () => {
      if (rollout.isLoading || !rollout.organizationReady) {
        showErrorToast("SuperPlane could not load billing settings. Try again.");
        return;
      }
      const result = await billing.refetch().catch(() => undefined);
      if (!result || result.error || !result.data) {
        showErrorToast("Could not load billing plans. Try again.");
        return;
      }
      if (
        rollout.has(FEATURE_NEW_BILLING) &&
        result.data.billingEnabled &&
        result.data.availablePlans?.length &&
        !isOrganizationPlanActive(result.data)
      ) {
        setBillingStepShown(true);
        navigation.goToScreen("billing");
        return;
      }
      await continueAfterPlan();
    };
    setPending(true);
    void continueFromTickets().finally(() => setPending(false));
  }, [ticketsSaved, model.saving, rollout, billing, navigation, continueAfterPlan]);
  const screen =
    navigation.screen === "agent" && (!access.allowed || (hasBillingStep && !isOrganizationPlanActive(billing.data)))
      ? ("billing" as const)
      : navigation.screen;
  useEffect(() => {
    if (access.ready && screen !== navigation.screen) navigation.goToScreen(screen);
  }, [access.ready, screen, navigation]);
  return {
    screen,
    pending,
    hasBillingStep,
    skipAgentScreen: !access.allowed,
    accessError: access.error,
    afterTickets: async () => {
      setTicketsSaved(true);
    },
    continueAfterPlan,
  };
}
