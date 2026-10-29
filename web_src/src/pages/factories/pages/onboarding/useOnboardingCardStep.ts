import type { OrganizationsBillingPlanOption } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { onboardingBillingError } from "./onboardingBilling";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";

export type OnboardingCardStep = {
  /** The billing and permission lookups have answers. */
  ready: boolean;
  /** The billing lookup failed and no request is running. The user can retry. */
  failed: boolean;
  /** Billing is enabled and the user can manage the plan. */
  available: boolean;
  /** The user must select a starter plan during workspace setup. */
  required: boolean;
  /** An organization admin must finish billing before workspace setup. */
  requiresAdmin: boolean;
  plans: OrganizationsBillingPlanOption[];
  planActive: boolean;
  activePlanKey?: string;
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
  const { data: currentBilling = {} } = billing;
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const plans = ["pay_as_you_go", "starter"].flatMap((key) => {
    const plan = currentBilling.availablePlans?.find((candidate) => candidate.key === key && !candidate.perSeat);
    return plan ? [plan] : [];
  });
  const canManageBilling = !permissionsLoading && canAct("org", "update");
  const available = Boolean(currentBilling.billingEnabled) && canManageBilling;
  const planActive = currentBilling.planCategory === "paid";
  const billingRequired = Boolean(onboardingBillingError(billing.data));
  return {
    ready: billing.isSuccess && !permissionsLoading,
    failed: billing.isError && !billing.isFetching,
    available,
    required: available && billingRequired,
    requiresAdmin: !permissionsLoading && billingRequired && !canManageBilling,
    plans,
    planActive,
    activePlanKey: currentBilling.catalogKey,
    hasPaymentMethod: Boolean(currentBilling.hasPaymentMethod),
    occupiedSeats: currentBilling.occupiedSeats,
    welcomeCreditCents: Number(currentBilling.welcomeRemainingCents ?? 0),
    refetch: billing.refetch,
  };
}
