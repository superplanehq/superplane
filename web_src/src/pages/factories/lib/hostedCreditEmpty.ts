import { parseWorkOrderMetric } from "./workOrderUsage";

export type HostedCreditBannerKind = "trial" | "trial-expired" | "low" | "empty" | "lapsed";

/** At or below this remaining balance, paid organizations see a low-credit warning. */
export const LOW_HOSTED_CREDIT_THRESHOLD_CENTS = 2000;

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

/** Compact header chip kinds. These sit next to the page title, not in the banner row. */
export type HostedCreditHeaderKickerKind = "trial" | "trial-expired" | "lapsed" | "low" | "empty";

export function isHostedCreditHeaderKickerKind(_kind: HostedCreditBannerKind): _kind is HostedCreditHeaderKickerKind {
  return true;
}

export function hostedCreditHeaderKickerLabel(kind: HostedCreditHeaderKickerKind): string {
  switch (kind) {
    case "trial-expired":
      return "Trial ended";
    case "lapsed":
      return "No plan";
    case "low":
      return "Credit low";
    case "empty":
      return "No credit";
    case "trial":
      return "Trial";
  }
}

/** Action that the header chip calls. Low and empty credit send the user to billing to add credit. */
export function hostedCreditHeaderKickerActionLabel(kind: HostedCreditHeaderKickerKind): string {
  return kind === "low" || kind === "empty" ? "Add credits" : "Subscribe";
}

export function organizationPlanLabel(
  args: Pick<HostedCreditBannerInput, "purchasedCreditCents" | "welcomeCreditExpiresAt" | "plan" | "trialEndsAt"> & {
    now?: Date;
  },
): string | undefined {
  if (args.plan === "business") {
    return "Business";
  }
  if (args.plan === "none") {
    return "No plan";
  }
  if (!isHostedCreditTrialOrg(args)) {
    return undefined;
  }
  const expiresAt = parseWelcomeCreditExpiresAt(args.trialEndsAt ?? args.welcomeCreditExpiresAt);
  const now = args.now ?? new Date();
  if (expiresAt != null && expiresAt.getTime() <= now.getTime()) {
    return "Trial ended";
  }
  return "Trial";
}

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

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
      description: trialBillingDescription(expired, expiresAt),
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

function trialBillingDescription(expired: boolean, expiresAt: Date | null): string {
  if (expired) {
    return "The trial has ended. Hosted runs cannot start. Subscribe to Business to continue.";
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
    return "Hosted credit is empty. Click Top up to purchase hosted credit.";
  }
  return "Hosted credit is empty. SuperPlane-hosted runs cannot start until an installation admin adds credit.";
}
