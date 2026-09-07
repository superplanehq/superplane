import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";

import { HostedCreditEmptyBanner } from "../HostedCreditEmptyBanner";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { shouldShowHostedCreditEmptyBanner } from "./hostedCreditEmpty";

export function useHostedCreditEmptyBanner(organizationId: string, factoryKey: string) {
  const { canAct } = usePermissions();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  if (!spend.data || !shouldShowHostedCreditEmptyBanner(spend.data)) {
    return undefined;
  }

  return (
    <HostedCreditEmptyBanner
      billingEnabled={spend.data.billingEnabled === true}
      canManageBilling={canAct("org", "update")}
      spendingHref={factorySettingsSectionPath(organizationId, factoryKey, "organization", "spending")}
    />
  );
}
