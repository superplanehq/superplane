import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsDescribeOrganizationWorkspaceUsageResponse,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { ReactNode } from "react";

import { HostedCreditEmptyBanner, HostedCreditHeaderKicker } from "../HostedCreditEmptyBanner";
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
    subscriptionCheckoutEnabled: billingData?.subscriptionCheckoutEnabled,
  };
}

export function useHostedCreditChrome(
  organizationId: string,
  factoryKey: string,
): { headerKicker?: ReactNode; banner?: ReactNode } {
  const { canAct } = usePermissions();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const { kind, remainingCreditCents, welcomeCreditExpiresAt, billingEnabled, subscriptionCheckoutEnabled } =
    hostedCreditChromeState(spend.data, billing.data);
  if (!kind) {
    return {};
  }

  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing");
  const canManageBilling = canAct("org", "update");

  if (kind === "trial") {
    return {
      headerKicker: (
        <HostedCreditHeaderKicker
          spendingHref={spendingHref}
          welcomeCreditExpiresAt={welcomeCreditExpiresAt}
          remainingCreditCents={remainingCreditCents}
        />
      ),
    };
  }

  return {
    banner: (
      <HostedCreditEmptyBanner
        kind={kind}
        billingEnabled={billingEnabled}
        canManageBilling={canManageBilling}
        remainingCreditCents={remainingCreditCents}
        welcomeCreditExpiresAt={welcomeCreditExpiresAt}
        subscriptionCheckoutEnabled={subscriptionCheckoutEnabled}
        spendingHref={spendingHref}
      />
    ),
  };
}

export function useHostedCreditEmptyBanner(organizationId: string, factoryKey: string) {
  return useHostedCreditChrome(organizationId, factoryKey).banner;
}
