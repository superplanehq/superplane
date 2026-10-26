import type { OrganizationsBillingPlanOption } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { FEATURE_NEW_BILLING } from "@/lib/experimentalFeatures";

export type OnboardingCardStep = {
  /** Billing and feature lookups have an answer. */
  ready: boolean;
  /** The card step belongs to this flow: new billing with a default plan. */
  available: boolean;
  /** The organization still has to start on the default plan. */
  required: boolean;
  plan: OrganizationsBillingPlanOption | undefined;
  planActive: boolean;
  hasPaymentMethod: boolean;
  refetch: () => Promise<unknown>;
};

/**
 * The card step follows the backlog step. The organization saves a card and
 * starts on the plan that the admin marked as the onboarding default. An
 * organization that already has a paid plan, for example from an earlier
 * workspace, does not see the step.
 */
export function useOnboardingCardStep(organizationId: string): OnboardingCardStep {
  const rollout = useExperimentalFeature(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const plan = billing.data?.availablePlans?.find((candidate) => candidate.onboardingDefault);
  const available = rollout.has(FEATURE_NEW_BILLING) && Boolean(billing.data?.billingEnabled) && plan != null;
  const planActive = billing.data?.planCategory === "paid";
  return {
    ready: !billing.isPending && !rollout.isLoading,
    available,
    required: available && !planActive,
    plan,
    planActive,
    hasPaymentMethod: Boolean(billing.data?.hasPaymentMethod),
    refetch: billing.refetch,
  };
}
