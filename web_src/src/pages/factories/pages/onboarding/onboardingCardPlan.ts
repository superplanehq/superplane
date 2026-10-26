import { organizationsSelectBillingPlan } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

/**
 * Starts the organization on the onboarding default plan. The plan has no
 * price and no checkout, so the server activates it at once when a card is
 * saved.
 */
export async function activateOnboardingPlan(organizationId: string, planKey: string): Promise<void> {
  const response = await organizationsSelectBillingPlan(
    withOrganizationHeader({ organizationId, path: { id: organizationId }, body: { planKey } }),
  );
  if (response.data?.checkoutUrl) {
    throw new Error("This plan needs a checkout. Select it in Settings → Billing.");
  }
}
