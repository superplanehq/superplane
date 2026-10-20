export const BILLING_CANCEL_LABEL = "Cancel subscription";
export const BILLING_KEEP_LABEL = "Keep subscription";

export const BILLING_INCLUDED_USAGE_CENTS = 5000;

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
  return `Subscription ends on ${dateLabel}. Hosted runs continue until that date.`;
}

export function billingCancelBusinessConfirmCopy(periodEnd?: string): string {
  const dateLabel = billingPeriodEndLabel(periodEnd);
  if (!dateLabel) {
    return "The subscription stays active until the current period ends. SuperPlane will not renew after that date.";
  }
  return `The subscription stays active until ${dateLabel}. SuperPlane will not renew after that date.`;
}

export function billingUsagePercentUsed(remainingCents: number, allowanceCents: number): number {
  if (allowanceCents <= 0) {
    return remainingCents <= 0 ? 100 : 0;
  }
  const used = Math.max(0, allowanceCents - remainingCents);
  return Math.min(100, Math.round((used / allowanceCents) * 100));
}
