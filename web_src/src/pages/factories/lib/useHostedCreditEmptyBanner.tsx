import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
} from "@/api-client";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { useMemo, type ReactNode } from "react";

import { HostedCreditHeaderKicker } from "../HostedCreditHeaderKicker";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { hostedCreditBannerKind } from "./hostedCreditEmpty";
import type { HostedCreditRunContext } from "./workOrderFailureReason";

function hostedCreditChromeState(
  spendData: OrganizationsDescribeOrganizationWorkspaceUsageResponse | undefined,
  billingData: OrganizationsDescribeOrganizationBillingResponse | undefined,
) {
  return {
    kind: spendData
      ? hostedCreditBannerKind({
          ...spendData,
          plan: billingData?.plan,
          trialEndsAt: billingData?.trialEndsAt,
          subscriptionCheckoutEnabled: billingData?.subscriptionCheckoutEnabled,
          creditPurchaseAllowed: billingData?.creditPurchaseAllowed,
        })
      : null,
    remainingCreditCents: parseWorkOrderMetric(spendData?.remainingCreditCents),
    welcomeCreditExpiresAt: billingData?.trialEndsAt ?? spendData?.welcomeCreditExpiresAt,
    billingEnabled: spendData?.billingEnabled === true || billingData?.billingEnabled === true,
  };
}

/** Credit and plan for the task popup, from the same queries as the header chip. */
export function useHostedCreditRunContext(organizationId: string, factoryKey: string): HostedCreditRunContext {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const plan = billing.data?.plan;
  const trialEndsAt = billing.data?.trialEndsAt;
  const welcomeCreditExpiresAt = spend.data?.welcomeCreditExpiresAt;
  const remainingCreditCents = spend.data ? parseWorkOrderMetric(spend.data.remainingCreditCents) : undefined;
  return useMemo(
    () => ({
      plan,
      remainingCreditCents,
      trialEndsAt,
      welcomeCreditExpiresAt,
      billingHref: factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing"),
    }),
    [factoryKey, organizationId, plan, remainingCreditCents, trialEndsAt, welcomeCreditExpiresAt],
  );
}

export function useHostedCreditChrome(organizationId: string, factoryKey: string): { headerKicker?: ReactNode } {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const { kind, remainingCreditCents, welcomeCreditExpiresAt, billingEnabled } = hostedCreditChromeState(
    spend.data,
    billing.data,
  );
  if (!kind) {
    return {};
  }

  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing");

  return {
    headerKicker: (
      <HostedCreditHeaderKicker
        kind={kind}
        spendingHref={spendingHref}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        remainingCreditCents={remainingCreditCents}
        canAddCredit={billingEnabled}
      />
    ),
  };
}
