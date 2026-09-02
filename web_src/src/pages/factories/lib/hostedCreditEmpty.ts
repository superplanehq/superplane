import { parseWorkOrderMetric } from "./workOrderUsage";

export function shouldShowHostedCreditEmptyBanner(args: {
  remainingCreditCents?: string | number;
  grantTotalCents?: string | number;
  superplaneGrantCents?: string | number;
  purchasedCreditCents?: string | number;
  billingEnabled?: boolean;
}): boolean {
  if (parseWorkOrderMetric(args.remainingCreditCents) > 0) {
    return false;
  }

  return (
    parseWorkOrderMetric(args.grantTotalCents) > 0 ||
    parseWorkOrderMetric(args.superplaneGrantCents) > 0 ||
    parseWorkOrderMetric(args.purchasedCreditCents) > 0 ||
    args.billingEnabled === true
  );
}

/**
 * `canManageBilling` defaults to `true` so existing owner-focused call sites
 * do not have to pass it. A non-owner cannot start checkout, so the banner
 * points them at Organization Spending instead of promising a button they
 * cannot use; the Spending card explains who to contact.
 */
export function hostedCreditEmptyBannerCopy(
  billingEnabled: boolean,
  canManageBilling: boolean = true,
): {
  title: string;
  description: string;
  actionLabel: string;
} {
  if (billingEnabled) {
    return {
      title: "Hosted credit is empty",
      description: "Add hosted credit to start SuperPlane-hosted runs.",
      actionLabel: canManageBilling ? "Add hosted credit" : "View spending",
    };
  }

  return {
    title: "Hosted credit is empty",
    description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    actionLabel: "View spending",
  };
}
