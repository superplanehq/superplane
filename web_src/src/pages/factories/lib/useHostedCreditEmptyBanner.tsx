import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";

import { HostedCreditEmptyBanner } from "../HostedCreditEmptyBanner";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { hostedCreditBannerKind } from "./hostedCreditEmpty";

export function useHostedCreditEmptyBanner(organizationId: string, factoryKey: string) {
  const { canAct } = usePermissions();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const kind = spend.data ? hostedCreditBannerKind(spend.data) : null;
  if (!kind) {
    return undefined;
  }

  return (
    <HostedCreditEmptyBanner
      kind={kind}
      billingEnabled={spend.data?.billingEnabled === true}
      canManageBilling={canAct("org", "update")}
      remainingCreditCents={parseWorkOrderMetric(spend.data?.remainingCreditCents)}
      welcomeCreditExpiresAt={spend.data?.welcomeCreditExpiresAt}
      spendingHref={factorySettingsSectionPath(organizationId, factoryKey, "organization", "billing")}
    />
  );
}
