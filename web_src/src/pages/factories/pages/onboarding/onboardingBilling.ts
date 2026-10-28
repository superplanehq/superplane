import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";

export function onboardingBillingError(
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined,
): string | null {
  if (!billing) return "SuperPlane could not verify billing. Try again.";
  if (!billing.billingEnabled) return null;
  if (billing.planCategory !== "paid") return "Select a billing plan before you complete workspace setup.";
  if (billing.requirePaymentMethod && !billing.hasPaymentMethod)
    return "Add a payment method before you complete workspace setup.";
  return null;
}
