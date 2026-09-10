import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { ReactNode } from "react";

import { HostedCreditEmptyBanner, HostedCreditHeaderKicker } from "../HostedCreditEmptyBanner";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { hostedCreditBannerKind } from "./hostedCreditEmpty";

export function useHostedCreditChrome(
  organizationId: string,
  factoryKey: string,
): { headerKicker?: ReactNode; banner?: ReactNode } {
  const { canAct } = usePermissions();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const kind = spend.data
    ? hostedCreditBannerKind({
        ...spend.data,
        plan: billing.data?.plan,
        trialEndsAt: billing.data?.trialEndsAt,
        subscriptionCheckoutEnabled: billing.data?.subscriptionCheckoutEnabled,
        creditPurchaseAllowed: billing.data?.creditPurchaseAllowed,
      })
    : null;
  if (!kind) {
    return {};
  }

  const spendingHref = factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing");
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  const welcomeCreditExpiresAt = billing.data?.trialEndsAt ?? spend.data?.welcomeCreditExpiresAt;
  const billingEnabled = spend.data?.billingEnabled === true || billing.data?.billingEnabled === true;
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
        subscriptionCheckoutEnabled={billing.data?.subscriptionCheckoutEnabled}
        spendingHref={spendingHref}
      />
    ),
  };
}

export function useHostedCreditEmptyBanner(organizationId: string, factoryKey: string) {
  return useHostedCreditChrome(organizationId, factoryKey).banner;
}
