import { formatUsdCents, parseWorkOrderMetric } from "./workOrderUsage";

export type HostedCreditWarningLevel = "empty" | "low";

/** At or below this remaining balance, the board and Tasks/Missions pages warn before credit runs out. */
export const LOW_HOSTED_CREDIT_THRESHOLD_CENTS = 2000;

export interface HostedCreditWarningArgs {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  superplaneGrantCents?: string | number;
  purchasedCreditCents?: string | number;
  billingEnabled?: boolean;
}

/**
 * `null` when remaining hosted credit is comfortably above the low-credit
 * threshold, or when the organization never had hosted credit to begin with
 * (no grant, no purchase, billing off — hosted runs were never expected to
 * work here, so warning about running out would be noise).
 */
export function hostedCreditWarningLevel(args: HostedCreditWarningArgs): HostedCreditWarningLevel | null {
  const remainingCents = parseWorkOrderMetric(args.remainingCreditCents);
  if (remainingCents > LOW_HOSTED_CREDIT_THRESHOLD_CENTS) {
    return null;
  }

  const hasHadHostedCredit =
    parseWorkOrderMetric(args.grantTotalCents) > 0 ||
    parseWorkOrderMetric(args.superplaneGrantCents) > 0 ||
    parseWorkOrderMetric(args.purchasedCreditCents) > 0 ||
    args.billingEnabled === true;
  if (!hasHadHostedCredit) {
    return null;
  }

  return remainingCents <= 0 ? "empty" : "low";
}

export function shouldShowHostedCreditEmptyBanner(args: HostedCreditWarningArgs): boolean {
  return hostedCreditWarningLevel(args) !== null;
}

/** Title and description for the banner. The action label is decided by the component, not here. */
export function hostedCreditEmptyBannerCopy(
  level: HostedCreditWarningLevel,
  billingEnabled: boolean,
): {
  title: string;
  description: string;
} {
  const lowCreditBudget = formatUsdCents(LOW_HOSTED_CREDIT_THRESHOLD_CENTS);

  if (level === "low") {
    return billingEnabled
      ? {
          title: "Hosted credit is running low",
          description: `Less than ${lowCreditBudget} left. Add hosted credit to avoid interrupting SuperPlane-hosted runs.`,
        }
      : {
          title: "Hosted credit is running low",
          description: `Less than ${lowCreditBudget} left. Ask an installation admin to add hosted credit soon.`,
        };
  }

  return billingEnabled
    ? {
        title: "Hosted credit is empty",
        description: "Add hosted credit to start SuperPlane-hosted runs.",
      }
    : {
        title: "Hosted credit is empty",
        description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
      };
}
