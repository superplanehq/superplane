export const BILLING_BUSINESS_NAME = "Business";
export const BILLING_BUSINESS_PRICE = "$199";
export const BILLING_BUSINESS_PRICE_PERIOD = "per month";
export const BILLING_BUSINESS_DESCRIPTION = "For teams that run software factory workflows.";
export const BILLING_UPGRADE_LABEL = "Upgrade to Business";

export const BILLING_INCLUDED_USAGE_CENTS = 5000;

export type BillingBusinessPlanAction = "subscribe" | "current" | "hidden";

export function billingBusinessPlanAction(args: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
}): BillingBusinessPlanAction {
  if (args.creditPurchaseAllowed) {
    return "current";
  }
  if (!args.canManageBilling) {
    return "hidden";
  }
  return "subscribe";
}

export function billingUsagePercentUsed(remainingCents: number, allowanceCents: number): number {
  if (allowanceCents <= 0) {
    return remainingCents <= 0 ? 100 : 0;
  }
  const used = Math.max(0, allowanceCents - remainingCents);
  return Math.min(100, Math.round((used / allowanceCents) * 100));
}
