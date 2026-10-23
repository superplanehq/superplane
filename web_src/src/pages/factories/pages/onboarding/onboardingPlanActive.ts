import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";

type OrganizationBilling = OrganizationsDescribeOrganizationBillingResponse | undefined;

// The organization has a paid-category plan that allows credit purchases. The
// plan can still need a payment card.
export function hasPlanWithCreditAccess(billing: OrganizationBilling): boolean {
  return billing?.planCategory === "paid" && billing.creditPurchaseAllowed === true;
}

// The organization plan needs no more setup. The plan belongs to the
// organization, so a new workspace does not ask for a plan again.
export function isOrganizationPlanActive(billing: OrganizationBilling): boolean {
  if (!billing || !hasPlanWithCreditAccess(billing)) return false;
  return !billing.requirePaymentMethod || billing.hasPaymentMethod === true;
}
