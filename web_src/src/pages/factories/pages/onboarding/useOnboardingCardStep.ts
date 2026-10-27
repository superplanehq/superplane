import type { OrganizationsBillingPlanOption } from "@/api-client";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";

export type OnboardingCardStep = {
  /** The billing lookup has an answer. */
  ready: boolean;
  /** The card step belongs to this flow: billing is on and a starter plan is available. */
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
  const billing = useOrganizationBilling(organizationId);
  const plans = ["pay_as_you_go", "starter"].flatMap((key) => {
    const plan = billing.data?.availablePlans?.find((candidate) => candidate.key === key && !candidate.perSeat);
    return plan ? [plan] : [];
  });
  const available = Boolean(billing.data?.billingEnabled) && plans.length > 0;
  const planActive = billing.data?.planCategory === "paid";
  return {
    ready: !billing.isPending,
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
