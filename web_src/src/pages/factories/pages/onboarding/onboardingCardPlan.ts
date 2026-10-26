import { organizationsSelectBillingPlan } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

/**
 * Starts the organization on the onboarding default plan. A plan with no
 * checkout activates at once. A plan with a Polar product leaves for checkout.
 * Returns true when the browser leaves for checkout.
 */
export async function activateOnboardingPlan(
  organizationId: string,
  planKey: string,
  options?: { seats?: number; returnPath?: string },
): Promise<boolean> {
  const response = await organizationsSelectBillingPlan(
    withOrganizationHeader({
      organizationId,
      path: { id: organizationId },
      body: {
        planKey,
        ...(options?.seats == null ? {} : { seats: options.seats }),
        ...(options?.returnPath ? { returnPath: options.returnPath } : {}),
      },
    }),
  );
  const checkoutUrl = response.data?.checkoutUrl;
  if (!checkoutUrl) return false;
  window.location.assign(checkoutUrl);
  return true;
}
