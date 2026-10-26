import {
  organizationsCreatePaymentMethodSession,
  organizationsSelectBillingPlan,
  organizationsSyncOrganizationBilling,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type OnboardingCardSession = { token: string; origin: string };

export async function createOnboardingCardSession(organizationId: string): Promise<OnboardingCardSession> {
  const response = await organizationsCreatePaymentMethodSession(
    withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
  );
  const { sessionToken, portalUrl } = response.data ?? {};
  if (!sessionToken || !portalUrl) throw new Error("Could not open card setup. Try again.");
  const origin = new URL(portalUrl).origin;
  if (!["https://polar.sh", "https://sandbox.polar.sh"].includes(origin))
    throw new Error("Could not open card setup. Try again.");
  return { token: sessionToken, origin };
}

/** A return URL is not proof of payment. Reconcile Polar before continuing. */
export async function resumeOnboardingPlan(
  organizationId: string,
  planKey: string,
  returnPath: string,
  checkoutReturn: boolean,
): Promise<boolean> {
  const response = await organizationsSyncOrganizationBilling(
    withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
  );
  const billing = response.data;
  if (!billing) throw new Error("Could not verify billing. Try again.");
  if (billing.planCategory === "paid") {
    if (billing.catalogKey !== planKey)
      throw new Error("A different plan is active. Review your billing settings before continuing.");
    if (checkoutReturn && !["active", "trialing"].includes(billing.polarSubscriptionStatus ?? ""))
      throw new Error("Your subscription is not active yet. Wait a moment, then try again.");
    return false;
  }
  if (checkoutReturn) throw new Error("Your subscription is not active yet. Wait a moment, then try again.");
  if (!billing.hasPaymentMethod) throw new Error("No card was saved. Go back and add a card.");
  return activateOnboardingPlan(organizationId, planKey, { returnPath });
}

/**
 * Starts the organization on the selected onboarding plan. A plan with no
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
