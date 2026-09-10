import { formatUsdCents, parseWorkOrderMetric } from "./workOrderUsage";

export type HostedCreditBannerKind = "trial" | "trial-empty" | "trial-expired" | "low" | "empty";

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
  args: Pick<HostedCreditBannerInput, "purchasedCreditCents" | "welcomeCreditExpiresAt">,
): boolean {
  const purchased = parseWorkOrderMetric(args.purchasedCreditCents);
  const expiresAt = parseWelcomeCreditExpiresAt(args.welcomeCreditExpiresAt);
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
  const expiresAt = parseWelcomeCreditExpiresAt(args.welcomeCreditExpiresAt);
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

  if (remaining > LOW_HOSTED_CREDIT_THRESHOLD_CENTS) {
    return null;
  }

  if (!hasHadHostedCredit(args, purchased)) {
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
}

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

export function welcomeCreditExpirySentence(expiresAt: Date, now: Date = new Date()): string {
  const ms = expiresAt.getTime() - now.getTime();
  if (ms <= 0) {
    return "Free hosted credit expired.";
  }
  if (ms < ONE_DAY_MS) {
    return "It expires today.";
  }
  const days = Math.ceil(ms / ONE_DAY_MS);
  if (days === 1) {
    return "It expires in 1 day.";
  }
  return `It expires in ${days} days.`;
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
  now?: Date;
}): HostedCreditBannerCopy {
  const now = args.now ?? new Date();
  const tone = hostedCreditBannerTone(args.kind, args.welcomeCreditExpiresAt, now, args.remainingCreditCents);

  if (args.kind === "trial") {
    const remaining = formatUsdCents(args.remainingCreditCents ?? 0);
    const expirySentence = args.welcomeCreditExpiresAt
      ? welcomeCreditExpirySentence(args.welcomeCreditExpiresAt, now)
      : "It expires in 14 days.";
    const expiryLabel = args.welcomeCreditExpiresAt
      ? welcomeCreditExpiryLabel(args.welcomeCreditExpiresAt, now)
      : "14 days remaining";
    return withRunsStopHint(
      {
        title: "Trial",
        description: `You have ${remaining} of free hosted credit. ${expirySentence}`,
        remainingLabel: `${remaining} remaining`,
        expiryLabel,
        actionLabel: "Add credits",
        tone,
      },
      args.remainingCreditCents,
    );
  }

  if (args.kind === "trial-empty") {
    return {
      title: "Trial credit is empty",
      description: "SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone,
    };
  }

  if (args.kind === "trial-expired") {
    return {
      title: "Trial ended",
      description: "Free hosted credit expired. SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone,
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
      },
      args.remainingCreditCents,
    );
  }

  if (args.billingEnabled) {
    return {
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone,
    };
  }

  return {
    title: "Hosted credit is empty",
    description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    actionLabel: "Add credits",
    tone,
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
  now?: Date;
}

/** Copy for the Billing remaining-credit card when Polar has no customer. */
export function hostedCreditBillingBalanceCopy(args: HostedCreditBillingBalanceInput): HostedCreditBillingBalanceCopy {
  if (args.hasBillingCustomer) {
    return { badge: null, description: null };
  }

  const now = args.now ?? new Date();
  const trial = isHostedCreditTrialOrg({
    purchasedCreditCents: args.purchasedCents,
    welcomeCreditExpiresAt: args.welcomeCreditExpiresAt,
  });
  const expiresAt = parseWelcomeCreditExpiresAt(args.welcomeCreditExpiresAt);
  const expired = expiresAt != null && expiresAt.getTime() <= now.getTime();

  if (trial && args.remainingCents > 0 && !expired && expiresAt) {
    return {
      badge: "Trial",
      description:
        `This remaining balance is welcome credit. Welcome credit is a free trial grant. ` +
        `Unused credit expires on ${expiresAt.toLocaleDateString()}. ` +
        `Purchase hosted credit to keep SuperPlane-hosted runs after the trial.`,
    };
  }

  if (args.remainingCents > 0) {
    return { badge: null, description: null };
  }

  if (trial && expired) {
    return {
      badge: "Trial",
      description: expiredWelcomeCreditDescription(args.billingEnabled),
    };
  }

  return {
    badge: trial ? "Trial" : null,
    description: emptyHostedCreditDescription(args.billingEnabled),
  };
}

function expiredWelcomeCreditDescription(billingEnabled: boolean): string {
  if (billingEnabled) {
    return "Welcome credit expired. SuperPlane-hosted runs cannot start. Click Buy more to purchase hosted credit.";
  }
  return "Welcome credit expired. SuperPlane-hosted runs cannot start.";
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
