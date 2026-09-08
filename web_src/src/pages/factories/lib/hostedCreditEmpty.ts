import { formatUsdCents, parseWorkOrderMetric } from "./workOrderUsage";

export type HostedCreditBannerKind = "trial" | "trial-empty" | "trial-expired" | "empty";

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

export function hostedCreditBannerKind(args: HostedCreditBannerInput): HostedCreditBannerKind | null {
  const remaining = parseWorkOrderMetric(args.remainingCreditCents);
  const purchased = parseWorkOrderMetric(args.purchasedCreditCents);
  const expiresAt = parseWelcomeCreditExpiresAt(args.welcomeCreditExpiresAt);
  const now = args.now ?? new Date();
  const isTrialOrg = expiresAt != null && purchased === 0;

  if (isTrialOrg) {
    if (expiresAt.getTime() <= now.getTime()) {
      return "trial-expired";
    }
    if (remaining <= 0) {
      return "trial-empty";
    }
    return "trial";
  }

  if (remaining > 0) {
    return null;
  }

  if (
    parseWorkOrderMetric(args.grantTotalCents) > 0 ||
    parseWorkOrderMetric(args.superplaneGrantCents) > 0 ||
    purchased > 0 ||
    args.billingEnabled === true
  ) {
    return "empty";
  }

  return null;
}

export function shouldShowHostedCreditEmptyBanner(args: HostedCreditBannerInput): boolean {
  return hostedCreditBannerKind(args) != null;
}

export function welcomeCreditExpirySentence(expiresAt: Date, now: Date = new Date()): string {
  const ms = expiresAt.getTime() - now.getTime();
  if (ms <= 0) {
    return "Free hosted credit expired.";
  }
  if (ms < 24 * 60 * 60 * 1000) {
    return "It expires today.";
  }
  const days = Math.ceil(ms / (24 * 60 * 60 * 1000));
  if (days === 1) {
    return "It expires in 1 day.";
  }
  return `It expires in ${days} days.`;
}

export function hostedCreditBannerCopy(args: {
  kind: HostedCreditBannerKind;
  billingEnabled: boolean;
  remainingCreditCents?: number;
  welcomeCreditExpiresAt?: Date;
  now?: Date;
}): {
  title: string;
  description: string;
  actionLabel: string;
} {
  if (args.kind === "trial") {
    const expiry = args.welcomeCreditExpiresAt
      ? welcomeCreditExpirySentence(args.welcomeCreditExpiresAt, args.now ?? new Date())
      : "It expires in 14 days.";
    return {
      title: "Trial",
      description: `You have ${formatUsdCents(args.remainingCreditCents ?? 0)} of free hosted credit. ${expiry} Open Billing to buy more credit.`,
      actionLabel: "Open billing",
    };
  }

  if (args.kind === "trial-empty") {
    return {
      title: "Trial credit is empty",
      description: "SuperPlane-hosted runs cannot start. Open Billing to buy hosted credit.",
      actionLabel: "Open billing",
    };
  }

  if (args.kind === "trial-expired") {
    return {
      title: "Trial ended",
      description:
        "Free hosted credit expired. SuperPlane-hosted runs cannot start. Open Billing to buy hosted credit.",
      actionLabel: "Open billing",
    };
  }

  if (args.billingEnabled) {
    return {
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start. Open Billing to review remaining credit.",
      actionLabel: "View billing",
    };
  }

  return {
    title: "Hosted credit is empty",
    description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    actionLabel: "View billing",
  };
}

/**
 * `canManageBilling` is ignored. Billing does not offer checkout on this page yet.
 */
export function hostedCreditEmptyBannerCopy(
  billingEnabled: boolean,
  _canManageBilling: boolean = true,
): {
  title: string;
  description: string;
  actionLabel: string;
} {
  return hostedCreditBannerCopy({ kind: "empty", billingEnabled });
}
