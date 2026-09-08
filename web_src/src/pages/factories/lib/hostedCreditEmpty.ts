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
  if (billingEnabled) {
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
