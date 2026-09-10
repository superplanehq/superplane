import { isHostedCreditTrialOrg, parseWelcomeCreditExpiresAt } from "./hostedCreditEmpty";
import { formatUsdCents } from "./workOrderUsage";

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

export interface BillingPlansUsageInput {
  plan?: string;
  remainingCents: number;
  grantTotalCents: number;
  includedRemainingCents: number;
  purchasedCents: number;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  currentPeriodEnd?: string;
  now?: Date;
}

export interface BillingPlansUsageView {
  heading: string;
  remainingLabel: string;
  usedPercent: number;
  footer: string | null;
}

export function billingUsagePercentUsed(remainingCents: number, allowanceCents: number): number {
  if (allowanceCents <= 0) {
    return remainingCents <= 0 ? 100 : 0;
  }
  const used = Math.max(0, allowanceCents - remainingCents);
  return Math.min(100, Math.round((used / allowanceCents) * 100));
}

export function billingPlansUsageView(args: BillingPlansUsageInput): BillingPlansUsageView {
  const now = args.now ?? new Date();
  const trial = isHostedCreditTrialOrg({
    purchasedCreditCents: args.purchasedCents,
    welcomeCreditExpiresAt: args.welcomeCreditExpiresAt,
    plan: args.plan,
    trialEndsAt: args.trialEndsAt,
  });

  if (trial) {
    const allowance = Math.max(args.grantTotalCents, args.remainingCents);
    const expiresAt = parseWelcomeCreditExpiresAt(args.trialEndsAt ?? args.welcomeCreditExpiresAt);
    return {
      heading: "Your trial usage",
      remainingLabel: `${formatUsdCents(args.remainingCents)} remaining`,
      usedPercent: billingUsagePercentUsed(args.remainingCents, allowance),
      footer: trialUsageFooter(expiresAt, now),
    };
  }

  if (args.plan === "business") {
    const periodEnd = parseWelcomeCreditExpiresAt(args.currentPeriodEnd);
    return {
      heading: "Your included usage",
      remainingLabel: `${formatUsdCents(args.includedRemainingCents)} remaining`,
      usedPercent: billingUsagePercentUsed(args.includedRemainingCents, BILLING_INCLUDED_USAGE_CENTS),
      footer: periodEnd ? `Resets ${periodEnd.toLocaleDateString()}` : null,
    };
  }

  return {
    heading: "Your hosted usage",
    remainingLabel: `${formatUsdCents(args.remainingCents)} remaining`,
    usedPercent: billingUsagePercentUsed(args.remainingCents, args.remainingCents),
    footer:
      args.remainingCents > 0
        ? "Subscribe to Business to use this credit."
        : "Subscribe to Business to start hosted runs.",
  };
}

function trialUsageFooter(expiresAt: Date | null, now: Date): string | null {
  if (!expiresAt) {
    return null;
  }
  if (expiresAt.getTime() <= now.getTime()) {
    return "The trial has ended.";
  }
  return `Ends ${expiresAt.toLocaleDateString()}`;
}
