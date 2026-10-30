export const BILLING_BUSINESS_NAME = "Business";
export const BILLING_BUSINESS_PRICE = "$199";
export const BILLING_BUSINESS_PRICE_PERIOD = "per month";
export const BILLING_BUSINESS_DESCRIPTION = "For teams that run software factory workflows.";
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

export function billingSubscriptionEndsCopy(periodEnd?: string, planName = BILLING_BUSINESS_NAME): string | null {
  const dateLabel = billingPeriodEndLabel(periodEnd);
  if (!dateLabel) {
    return null;
  }
  return `${planName} ends on ${dateLabel}. Hosted runs continue until that date.`;
}

export function billingCancelBusinessConfirmCopy(periodEnd?: string, planName = BILLING_BUSINESS_NAME): string {
  const dateLabel = billingPeriodEndLabel(periodEnd);
  if (!dateLabel) {
    return `${planName} stays active until the current period ends. SuperPlane will not renew after that date.`;
  }
  return `${planName} stays active until ${dateLabel}. SuperPlane will not renew after that date.`;
}

export function billingUsagePercentUsed(remainingCents: number, allowanceCents: number): number {
  if (allowanceCents <= 0) {
    return remainingCents <= 0 ? 100 : 0;
  }
  const used = Math.max(0, allowanceCents - remainingCents);
  return Math.min(100, Math.round((used / allowanceCents) * 100));
}

export const BILLING_UPDATE_PAYMENT_METHOD_LABEL = "Update payment method";
export const BILLING_PAYMENT_BLOCKED_SEATS_COPY = "No one can join until the payment is updated.";

/** True while Polar retries a failed renewal payment. */
export function billingIsPastDue(status?: string): boolean {
  return status === "past_due";
}

/** True while no new member can join because payment is failed or paused. */
export function billingPaymentBlocked(status?: string): boolean {
  return billingIsPastDue(status) || status === "paused";
}

/** Banner text for a failed renewal. A grace period adds the date that Team features end. */
export function billingPaymentFailedCopy(failedAt?: string, accessEndsAt?: string): string {
  const failedDate = billingPeriodEndLabel(failedAt);
  const failure = failedDate ? `The renewal payment failed on ${failedDate}.` : "The renewal payment failed.";
  const endDate = billingPeriodEndLabel(accessEndsAt);
  if (endDate) {
    return `${failure} Update the payment method by ${endDate} to keep Team features.`;
  }
  return `${failure} Team features are off until you update the payment method.`;
}

export type CheckoutSeatBounds = {
  minimum: number;
  defaultSeats: number;
  coveredInvitations: number;
  invitationsWithoutSeat: number;
};

// Decision D13: the default covers members plus pending invitations, capped at
// the plan maximum. Only current members set the minimum.
export function checkoutSeatBounds(
  occupied: number | undefined,
  pending: number | undefined,
  maximum: number | undefined,
): CheckoutSeatBounds {
  const members = occupied ?? 0;
  const invitations = Math.max(pending ?? 0, 0);
  const minimum = Math.max(members, 1);
  const requested = Math.max(minimum, members + invitations);
  if (maximum == null) {
    return { minimum, defaultSeats: requested, coveredInvitations: invitations, invitationsWithoutSeat: 0 };
  }
  if (maximum < minimum) {
    return { minimum, defaultSeats: minimum, coveredInvitations: 0, invitationsWithoutSeat: 0 };
  }
  const defaultSeats = Math.min(maximum, requested);
  const coveredInvitations = Math.min(invitations, Math.max(defaultSeats - members, 0));
  return { minimum, defaultSeats, coveredInvitations, invitationsWithoutSeat: invitations - coveredInvitations };
}

export function billingPlanParallelism(limit?: number): string {
  return limit == null ? "Unlimited tasks at once" : `Up to ${limit} ${limit === 1 ? "task" : "tasks"} at once`;
}
