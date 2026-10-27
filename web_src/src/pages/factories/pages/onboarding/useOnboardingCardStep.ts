import type { OrganizationsBillingPlanOption } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { FEATURE_NEW_BILLING } from "@/lib/experimentalFeatures";

export type OnboardingCardStep = {
  /** Billing and feature lookups have an answer. */
  ready: boolean;
  /** The card step belongs to this flow: new billing with an available starter plan. */
  available: boolean;
  /** The organization still has to select a starter plan. */
  required: boolean;
  plans: OrganizationsBillingPlanOption[];
  planActive: boolean;
  hasPaymentMethod: boolean;
  occupiedSeats?: number;
  welcomeCreditCents?: number;
  refetch: () => Promise<unknown>;
};

/**
 * The card step follows the backlog step. The organization saves a card and
 * starts on Pay-as-you-go or Starter. An
 * organization that already has a paid plan, for example from an earlier
 * workspace, does not see the step.
 */
export function useOnboardingCardStep(organizationId: string): OnboardingCardStep {
  const rollout = useExperimentalFeature(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const plans = ["pay_as_you_go", "starter"].flatMap((key) => {
    const plan = billing.data?.availablePlans?.find((candidate) => candidate.key === key && !candidate.perSeat);
    return plan ? [plan] : [];
  });
  const available = rollout.has(FEATURE_NEW_BILLING) && Boolean(billing.data?.billingEnabled) && plans.length > 0;
  const planActive = billing.data?.planCategory === "paid";
  return {
    ready: !billing.isPending && !rollout.isLoading,
    available,
    required: available && !planActive,
    plans,
    planActive,
    hasPaymentMethod: Boolean(billing.data?.hasPaymentMethod),
    occupiedSeats: billing.data?.occupiedSeats,
    welcomeCreditCents: Number(billing.data?.welcomeRemainingCents ?? 0),
    refetch: billing.refetch,
  };
}
