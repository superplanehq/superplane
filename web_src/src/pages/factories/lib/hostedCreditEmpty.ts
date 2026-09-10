import { formatUsdCents, parseWorkOrderMetric } from "./workOrderUsage";

export type HostedCreditBannerKind = "trial" | "trial-empty" | "trial-expired" | "low" | "empty" | "lapsed";

/** At or below this remaining balance, paid organizations see a low-credit warning. */
export const LOW_HOSTED_CREDIT_THRESHOLD_CENTS = 2000;

/** Compact helper shown next to remaining credit when the balance is low. */
export const HOSTED_CREDIT_RUNS_STOP_HINT = "Tasks stop when credit runs out.";

export function isLowHostedCreditRemaining(remainingCents: number | undefined): boolean {
  return remainingCents != null && remainingCents > 0 && remainingCents <= LOW_HOSTED_CREDIT_THRESHOLD_CENTS;
}

export function hostedCreditRunsStopHint(remainingCents: number | undefined): string | undefined {
  if (!isLowHostedCreditRemaining(remainingCents)) {
    return undefined;
  }
  return HOSTED_CREDIT_RUNS_STOP_HINT;
}

export interface HostedCreditBannerInput {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  superplaneGrantCents?: string | number;
  purchasedCreditCents?: string | number;
  billingEnabled?: boolean;
  welcomeCreditExpiresAt?: string;
  plan?: string;
  trialEndsAt?: string;
  subscriptionCheckoutEnabled?: boolean;
  creditPurchaseAllowed?: boolean;
  now?: Date;
}

export function parseWelcomeCreditExpiresAt(value: string | undefined): Date | null {
  if (!value) {
    return null;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  return parsed;
}

export function isWelcomeCreditExpired(value: string | undefined, now: Date = new Date()): boolean {
  const parsed = parseWelcomeCreditExpiresAt(value);
  return parsed != null && parsed.getTime() <= now.getTime();
}

export function isHostedCreditTrialOrg(
  args: Pick<HostedCreditBannerInput, "purchasedCreditCents" | "welcomeCreditExpiresAt" | "plan" | "trialEndsAt">,
): boolean {
  if (args.plan === "business" || args.plan === "none") {
    return false;
  }
  if (args.plan === "trial") {
    return true;
  }
  const purchased = parseWorkOrderMetric(args.purchasedCreditCents);
  const expiresAt = parseWelcomeCreditExpiresAt(args.trialEndsAt ?? args.welcomeCreditExpiresAt);
  return expiresAt != null && purchased === 0;
}

function hasHadHostedCredit(args: HostedCreditBannerInput, purchased: number): boolean {
  return (
    parseWorkOrderMetric(args.grantTotalCents) > 0 ||
    parseWorkOrderMetric(args.superplaneGrantCents) > 0 ||
    purchased > 0 ||
    args.billingEnabled === true
  );
}

export function hostedCreditBannerKind(args: HostedCreditBannerInput): HostedCreditBannerKind | null {
  const remaining = parseWorkOrderMetric(args.remainingCreditCents);
  const purchased = parseWorkOrderMetric(args.purchasedCreditCents);
  const expiresAt = parseWelcomeCreditExpiresAt(args.trialEndsAt ?? args.welcomeCreditExpiresAt);
  const now = args.now ?? new Date();
  const isTrialOrg = isHostedCreditTrialOrg(args);

  if (isTrialOrg && expiresAt) {
    if (expiresAt.getTime() <= now.getTime()) {
      return "trial-expired";
    }
    if (remaining <= 0) {
      return "trial-empty";
    }
    return "trial";
  }

  if (args.plan === "none") {
    return "lapsed";
  }

  if (remaining > LOW_HOSTED_CREDIT_THRESHOLD_CENTS) {
    return null;
  }

  if (!hasHadHostedCredit(args, purchased) && args.plan !== "business") {
    return null;
  }

  return remaining <= 0 ? "empty" : "low";
}

export function shouldShowHostedCreditEmptyBanner(args: HostedCreditBannerInput): boolean {
  return hostedCreditBannerKind(args) != null;
}

export type HostedCreditBannerTone = "info" | "warning";

export interface HostedCreditBannerCopy {
  title: string;
  description: string;
  actionLabel: string;
  remainingLabel?: string;
  expiryLabel?: string;
  consequenceHint?: string;
  tone: HostedCreditBannerTone;
  showAction?: boolean;
  showPricingLink?: boolean;
}

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

export function trialExpirySentence(expiresAt: Date, now: Date = new Date()): string {
  const ms = expiresAt.getTime() - now.getTime();
  if (ms <= 0) {
    return "The trial has ended.";
  }
  if (ms < ONE_DAY_MS) {
    return "The trial ends today.";
  }
  const days = Math.ceil(ms / ONE_DAY_MS);
  if (days === 1) {
    return "The trial ends in 1 day.";
  }
  return `The trial ends in ${days} days.`;
}

/** Duration segment in the title chip, e.g. "13 days". */
export function welcomeCreditHeaderLabel(expiresAt: Date, now: Date = new Date()): string {
  const ms = expiresAt.getTime() - now.getTime();
  if (ms <= 0) {
    return "Ended";
  }
  if (ms < ONE_DAY_MS) {
    return "Ends today";
  }
  const days = Math.ceil(ms / ONE_DAY_MS);
  if (days === 1) {
    return "1 day";
  }
  return `${days} days`;
}

/** Short expiry label for scannable trial chrome. */
export function welcomeCreditExpiryLabel(expiresAt: Date, now: Date = new Date()): string {
  const ms = expiresAt.getTime() - now.getTime();
  if (ms <= 0) {
    return "Expired";
  }
  if (ms < ONE_DAY_MS) {
    return "Expires today";
  }
  const days = Math.ceil(ms / ONE_DAY_MS);
  if (days === 1) {
    return "1 day remaining";
  }
  return `${days} days remaining`;
}

export function hostedCreditBannerTone(
  kind: HostedCreditBannerKind,
  welcomeCreditExpiresAt?: Date,
  now: Date = new Date(),
  remainingCreditCents?: number,
): HostedCreditBannerTone {
  if (kind !== "trial") {
    return "warning";
  }
  if (remainingCreditCents != null && remainingCreditCents <= LOW_HOSTED_CREDIT_THRESHOLD_CENTS) {
    return "warning";
  }
  if (!welcomeCreditExpiresAt) {
    return "info";
  }
  if (welcomeCreditExpiresAt.getTime() - now.getTime() <= ONE_DAY_MS) {
    return "warning";
  }
  return "info";
}

export function hostedCreditBannerCopy(args: {
  kind: HostedCreditBannerKind;
  billingEnabled: boolean;
  remainingCreditCents?: number;
  welcomeCreditExpiresAt?: Date;
  subscriptionCheckoutEnabled?: boolean;
  now?: Date;
}): HostedCreditBannerCopy {
  const now = args.now ?? new Date();
  const tone = hostedCreditBannerTone(args.kind, args.welcomeCreditExpiresAt, now, args.remainingCreditCents);
  const showSubscribe = args.subscriptionCheckoutEnabled !== false && args.billingEnabled;

  if (args.kind === "trial") {
    const remaining = formatUsdCents(args.remainingCreditCents ?? 0);
    const expirySentence = args.welcomeCreditExpiresAt
      ? trialExpirySentence(args.welcomeCreditExpiresAt, now)
      : "The trial ends in 14 days.";
    const expiryLabel = args.welcomeCreditExpiresAt
      ? welcomeCreditExpiryLabel(args.welcomeCreditExpiresAt, now)
      : "14 days remaining";
    return withRunsStopHint(
      {
        title: "Trial",
        description: `You have a 14-day Business trial with ${remaining} for machines and managed models. ${expirySentence} Subscribe to keep hosted runs.`,
        remainingLabel: `${remaining} remaining`,
        expiryLabel,
        actionLabel: "Subscribe",
        tone,
        showAction: showSubscribe,
        showPricingLink: true,
      },
      args.remainingCreditCents,
    );
  }

  if (args.kind === "trial-empty") {
    return {
      title: "Trial credit is used up",
      description: "Hosted runs cannot start. Subscribe to Business to continue.",
      actionLabel: "Subscribe",
      tone,
      showAction: showSubscribe,
      showPricingLink: true,
    };
  }

  if (args.kind === "trial-expired") {
    return {
      title: "Trial ended",
      description: "Hosted runs cannot start. Subscribe to Business to continue.",
      actionLabel: "Subscribe",
      tone,
      showAction: showSubscribe,
      showPricingLink: true,
    };
  }

  if (args.kind === "lapsed") {
    return {
      title: "Business subscription required",
      description: "Hosted runs cannot start. Subscribe to Business to continue.",
      actionLabel: "Subscribe",
      tone,
      showAction: showSubscribe,
      showPricingLink: true,
    };
  }

  if (args.kind === "low") {
    const lowCreditBudget = formatUsdCents(LOW_HOSTED_CREDIT_THRESHOLD_CENTS);
    const remainingLabel =
      args.remainingCreditCents != null && args.remainingCreditCents > 0
        ? `${formatUsdCents(args.remainingCreditCents)} remaining`
        : undefined;
    if (args.billingEnabled) {
      return withRunsStopHint(
        {
          title: "Hosted credit is low",
          description: `Less than ${lowCreditBudget} remains. Add hosted credit to keep SuperPlane-hosted runs.`,
          remainingLabel,
          actionLabel: "Add credits",
          tone,
          showAction: true,
        },
        args.remainingCreditCents,
      );
    }
    return withRunsStopHint(
      {
        title: "Hosted credit is low",
        description: `Less than ${lowCreditBudget} remains. Ask an installation admin to add hosted credit.`,
        remainingLabel,
        actionLabel: "Add credits",
        tone,
        showAction: false,
      },
      args.remainingCreditCents,
    );
  }

  if (args.billingEnabled) {
    return {
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start. Add hosted credit to continue.",
      actionLabel: "Add credits",
      tone,
      showAction: true,
    };
  }

  return {
    title: "Hosted credit is empty",
    description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    actionLabel: "Add credits",
    tone,
    showAction: false,
  };
}

/**
 * `canManageBilling` is ignored. Billing does not offer checkout on this page yet.
 */
export function hostedCreditEmptyBannerCopy(
  billingEnabled: boolean,
  _canManageBilling: boolean = true,
): HostedCreditBannerCopy {
  return hostedCreditBannerCopy({ kind: "empty", billingEnabled });
}

export interface HostedCreditBillingBalanceCopy {
  badge: string | null;
  description: string | null;
}

export interface HostedCreditBillingBalanceInput {
  remainingCents: number;
  purchasedCents: number;
  hasBillingCustomer: boolean;
  billingEnabled: boolean;
  welcomeCreditExpiresAt?: string;
  plan?: string;
  trialEndsAt?: string;
  creditPurchaseAllowed?: boolean;
  now?: Date;
}

/** Copy for the Billing remaining-credit card. */
export function hostedCreditBillingBalanceCopy(args: HostedCreditBillingBalanceInput): HostedCreditBillingBalanceCopy {
  const now = args.now ?? new Date();
  const trial = isHostedCreditTrialOrg({
    purchasedCreditCents: args.purchasedCents,
    welcomeCreditExpiresAt: args.welcomeCreditExpiresAt,
    plan: args.plan,
    trialEndsAt: args.trialEndsAt,
  });
  const expiresAt = parseWelcomeCreditExpiresAt(args.trialEndsAt ?? args.welcomeCreditExpiresAt);
  const expired = expiresAt != null && expiresAt.getTime() <= now.getTime();

  if (args.plan === "business" && args.creditPurchaseAllowed) {
    return { badge: "Business", description: null };
  }

  if (trial) {
    return {
      badge: "Trial",
      description: trialBillingDescription(args.remainingCents, expired, expiresAt),
    };
  }

  if (args.plan === "none") {
    return {
      badge: null,
      description: "Hosted runs cannot start. Subscribe to Business to continue.",
    };
  }

  if (args.remainingCents > 0) {
    return { badge: null, description: null };
  }

  if (args.billingEnabled && args.creditPurchaseAllowed) {
    return {
      badge: null,
      description: "Hosted credit is empty. Add hosted credit to keep SuperPlane-hosted runs.",
    };
  }

  return {
    badge: null,
    description: emptyHostedCreditDescription(args.billingEnabled),
  };
}

function trialBillingDescription(remainingCents: number, expired: boolean, expiresAt: Date | null): string {
  if (expired) {
    return "The trial has ended. Hosted runs cannot start. Subscribe to Business to continue.";
  }
  if (remainingCents <= 0) {
    return "Trial credit is used up. Hosted runs cannot start. Subscribe to Business to continue.";
  }
  if (expiresAt) {
    return (
      `This is trial usage for machines and managed models. ` +
      `The trial ends on ${expiresAt.toLocaleDateString()}. ` +
      `Subscribe to Business to keep hosted runs.`
    );
  }
  return "Subscribe to Business to keep hosted runs.";
}

function emptyHostedCreditDescription(billingEnabled: boolean): string {
  if (billingEnabled) {
    return "Hosted credit is empty. Click Buy more to purchase hosted credit.";
  }
  return "Hosted credit is empty. SuperPlane-hosted runs cannot start until an installation admin adds credit.";
}

function withRunsStopHint(copy: HostedCreditBannerCopy, remainingCents: number | undefined): HostedCreditBannerCopy {
  const consequenceHint = hostedCreditRunsStopHint(remainingCents);
  if (!consequenceHint) {
    return copy;
  }
  return { ...copy, consequenceHint };
}
