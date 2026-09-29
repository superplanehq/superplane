export const BILLING_BUSINESS_NAME = "Business";
export const BILLING_BUSINESS_PRICE = "$199";
export const BILLING_BUSINESS_PRICE_PERIOD = "per month";
export const BILLING_BUSINESS_DESCRIPTION = "For teams that run software factory workflows.";
export const BILLING_UPGRADE_LABEL = "Upgrade to Business";
export const BILLING_CANCEL_LABEL = "Cancel Business";
export const BILLING_KEEP_LABEL = "Keep Business";

export const BILLING_INCLUDED_USAGE_CENTS = 5000;

export type BillingBusinessPlanAction = "subscribe" | "current" | "ending" | "hidden";

export function billingBusinessPlanAction(args: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  cancelAtPeriodEnd?: boolean;
}): BillingBusinessPlanAction {
  if (args.creditPurchaseAllowed) {
    if (args.cancelAtPeriodEnd) {
      return "ending";
    }
    return "current";
  }
  if (!args.canManageBilling) {
    return "hidden";
  }
  return "subscribe";
}

export function billingCanCancelBusiness(args: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  cancelAtPeriodEnd?: boolean;
}): boolean {
  return args.canManageBilling && args.creditPurchaseAllowed && args.planSource === "polar" && !args.cancelAtPeriodEnd;
}

export function billingCanKeepBusiness(args: {
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  planSource?: string;
  cancelAtPeriodEnd?: boolean;
}): boolean {
  return (
    args.canManageBilling &&
    args.creditPurchaseAllowed &&
    args.planSource === "polar" &&
    Boolean(args.cancelAtPeriodEnd)
  );
}

export function billingPeriodEndLabel(periodEnd?: string): string | null {
  if (!periodEnd) {
    return null;
  }
  const date = new Date(periodEnd);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date.toLocaleDateString();
}

export function billingSubscriptionEndsCopy(periodEnd?: string): string | null {
  const dateLabel = billingPeriodEndLabel(periodEnd);
  if (!dateLabel) {
    return null;
  }
  return `Business ends on ${dateLabel}. Hosted runs continue until that date.`;
}

export function billingCancelBusinessConfirmCopy(periodEnd?: string): string {
  const dateLabel = billingPeriodEndLabel(periodEnd);
  if (!dateLabel) {
    return "Business stays active until the current period ends. SuperPlane will not renew after that date.";
  }
  return `Business stays active until ${dateLabel}. SuperPlane will not renew after that date.`;
}

export function billingUsagePercentUsed(remainingCents: number, allowanceCents: number): number {
  if (allowanceCents <= 0) {
    return remainingCents <= 0 ? 100 : 0;
  }
  const used = Math.max(0, allowanceCents - remainingCents);
  return Math.min(100, Math.round((used / allowanceCents) * 100));
}
