import type { OrganizationsBillingPlanOption } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";

export type OnboardingCardStep = {
  /** The billing and permission lookups have answers. */
  ready: boolean;
  /** The user can select one of the available starter plans. */
  available: boolean;
  /** The user must select a starter plan during workspace setup. */
  required: boolean;
  /** An organization admin must select the plan after workspace setup. */
  requiresAdmin: boolean;
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
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const plans = ["pay_as_you_go", "starter"].flatMap((key) => {
    const plan = billing.data?.availablePlans?.find((candidate) => candidate.key === key && !candidate.perSeat);
    return plan ? [plan] : [];
  });
  const canManageBilling = !permissionsLoading && canAct("org", "update");
  const hasStarterPlans = Boolean(billing.data?.billingEnabled) && plans.length > 0;
  const available = hasStarterPlans && canManageBilling;
  const planActive = billing.data?.planCategory === "paid";
  return {
    ready: !billing.isPending && !permissionsLoading,
    available,
    required: available && !planActive,
    requiresAdmin: !permissionsLoading && hasStarterPlans && !planActive && !canManageBilling,
    plans,
    planActive,
    hasPaymentMethod: Boolean(billing.data?.hasPaymentMethod),
    occupiedSeats: billing.data?.occupiedSeats,
    welcomeCreditCents: Number(billing.data?.welcomeRemainingCents ?? 0),
    refetch: billing.refetch,
  };
}
