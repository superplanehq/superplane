import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
} from "@/api-client";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { ReactNode } from "react";

import { HostedCreditHeaderKicker } from "../HostedCreditHeaderKicker";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { hostedCreditBannerKind } from "./hostedCreditEmpty";

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
