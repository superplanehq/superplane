import { isWelcomeCreditExpired, parseWelcomeCreditExpiresAt } from "./hostedCreditEmpty";
import { CREDIT_GRANT_KIND_ADMIN } from "./hostedCreditGrants";
import { BILLING_INCLUDED_USAGE_CENTS, billingUsagePercentUsed } from "./billingPlans";
import { formatUsdCents, parseWorkOrderMetric } from "./workOrderUsage";

export const BILLING_TRIAL_CREDIT_CENTS = 5000;

export const BILLING_SPEND_ORDER_COPY =
  "Hosted runs spend trial credit first, then included usage, then top-up credit.";

export const BILLING_SPEND_ORDER_WITH_GRANT_COPY =
  "Hosted runs spend trial credit first, then included usage, then top-up credit, then SuperPlane grant.";

export const BILLING_TRIAL_TTL_COPY = "Trial credit expires 14 days after registration.";

export type BillingCreditBucketKey = "trial" | "included" | "topup" | "grant";

export interface BillingCreditBucketsInput {
  welcomeRemainingCents: number;
  includedRemainingCents: number;
  purchasedRemainingCents: number;
  purchasedCents: number;
  adminRemainingCents?: number;
  adminGrantCents?: number;
  plan?: string;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  currentPeriodEnd?: string;
  now?: Date;
}

export interface BillingCreditBucketView {
  key: BillingCreditBucketKey;
  heading: string;
  spendOrderLabel: string;
  remainingCents: number;
  remainingLabel: string;
  usedPercent: number;
  footer: string | null;
}

export interface BillingCreditBarShare {
  key: BillingCreditBucketKey;
  percent: number;
}

const BILLING_BUCKET_SPEND_ORDER_NAME: Record<BillingCreditBucketKey, string> = {
  trial: "trial credit",
  included: "included usage",
  topup: "top-up credit",
  grant: "SuperPlane grant",
};

export function adminCreditGrantCents(grants: Array<{ kind?: string; amountCents?: string | number }>): number {
  return grants.reduce((sum, grant) => {
    if (grant.kind !== CREDIT_GRANT_KIND_ADMIN) {
      return sum;
    }
    return sum + parseWorkOrderMetric(grant.amountCents);
  }, 0);
}

export function billingSpendOrderCopy(buckets: Array<{ key: BillingCreditBucketKey }>): string {
  const names = buckets.map((bucket) => BILLING_BUCKET_SPEND_ORDER_NAME[bucket.key]);
  if (names.length === 0) {
    return "";
  }
  if (names.length === 1) {
    return `Hosted runs spend ${names[0]} first.`;
  }
  return `Hosted runs spend ${names[0]} first, then ${names.slice(1).join(", then ")}.`;
}

export function billingCreditBucketsView(args: BillingCreditBucketsInput): BillingCreditBucketView[] {
  const now = args.now ?? new Date();
  const trialExpiresAt = args.trialEndsAt ?? args.welcomeCreditExpiresAt;
  const periodEnd = parseWelcomeCreditExpiresAt(args.currentPeriodEnd);
  const welcomeRemainingCents = Math.max(0, args.welcomeRemainingCents);
  const adminRemainingCents = Math.max(0, args.adminRemainingCents ?? 0);
  const showTrialCredit = welcomeRemainingCents > 0 && !isWelcomeCreditExpired(trialExpiresAt, now);
  const showSuperPlaneGrant = adminRemainingCents > 0;

  const buckets: Omit<BillingCreditBucketView, "spendOrderLabel">[] = [];

  if (showTrialCredit) {
    buckets.push({
      key: "trial",
      heading: "Trial credit",
      remainingCents: welcomeRemainingCents,
      remainingLabel: `${formatUsdCents(args.welcomeRemainingCents)} remaining`,
      usedPercent: billingUsagePercentUsed(args.welcomeRemainingCents, BILLING_TRIAL_CREDIT_CENTS),
      footer: trialCreditFooter(parseWelcomeCreditExpiresAt(trialExpiresAt)),
    });
  }

  buckets.push({
    key: "included",
    heading: "Included usage",
    remainingCents: Math.max(0, args.includedRemainingCents),
    remainingLabel: `${formatUsdCents(args.includedRemainingCents)} remaining`,
    usedPercent: billingUsagePercentUsed(args.includedRemainingCents, BILLING_INCLUDED_USAGE_CENTS),
    footer: includedUsageFooter(args.plan, periodEnd),
  });

  buckets.push({
    key: "topup",
    heading: "Top-up credit",
    remainingCents: Math.max(0, args.purchasedRemainingCents),
    remainingLabel: `${formatUsdCents(args.purchasedRemainingCents)} remaining`,
    usedPercent: billingUsagePercentUsed(
      args.purchasedRemainingCents,
      Math.max(args.purchasedCents, args.purchasedRemainingCents),
    ),
    footer: null,
  });

  if (showSuperPlaneGrant) {
    buckets.push({
      key: "grant",
      heading: "SuperPlane grant",
      remainingCents: adminRemainingCents,
      remainingLabel: `${formatUsdCents(adminRemainingCents)} remaining`,
      usedPercent: billingUsagePercentUsed(
        adminRemainingCents,
        Math.max(args.adminGrantCents ?? 0, adminRemainingCents),
      ),
      footer: null,
    });
  }

  return buckets.map((bucket, index) => ({
    ...bucket,
    spendOrderLabel: spendOrderLabelAt(index, buckets.length),
  }));
}

export function billingCreditRemainingShares(buckets: BillingCreditBucketView[]): BillingCreditBarShare[] {
  const total = buckets.reduce((sum, bucket) => sum + Math.max(0, bucket.remainingCents), 0);
  if (total <= 0) {
    return buckets.map((bucket) => ({ key: bucket.key, percent: 0 }));
  }

  const raw = buckets.map((bucket) => ({
    key: bucket.key,
    percent: (Math.max(0, bucket.remainingCents) / total) * 100,
  }));
  const rounded = raw.map((share) => ({ ...share, percent: Math.round(share.percent) }));
  const drift = 100 - rounded.reduce((sum, share) => sum + share.percent, 0);
  const adjustIndex = rounded.findIndex((share) => share.percent > 0);
  if (adjustIndex >= 0) {
    rounded[adjustIndex] = {
      ...rounded[adjustIndex],
      percent: Math.max(0, rounded[adjustIndex].percent + drift),
    };
  }
  return rounded;
}

function spendOrderLabelAt(index: number, count: number): string {
  if (index === 0) {
    return "Spend first";
  }
  if (index === count - 1) {
    return "Spend last";
  }
  if (index === 1) {
    return "Spend next";
  }
  if (index === 2) {
    return "Spend third";
  }
  return `Spend ${index + 1}`;
}

function trialCreditFooter(expiresAt: Date | null): string {
  if (!expiresAt) {
    return BILLING_TRIAL_TTL_COPY;
  }
  return `Expires on ${expiresAt.toLocaleDateString()}. ${BILLING_TRIAL_TTL_COPY}`;
}

function includedUsageFooter(plan: string | undefined, periodEnd: Date | null): string {
  if (plan === "business" && periodEnd) {
    return `Resets ${periodEnd.toLocaleDateString()}`;
  }
  return "Included with Business.";
}
