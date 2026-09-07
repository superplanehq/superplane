import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";

import { HostedCreditEmptyBanner } from "../HostedCreditEmptyBanner";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { hostedCreditWarningLevel } from "./hostedCreditEmpty";

/** Stable no-op: there is no billing page yet, so the board's action button does nothing (for now). */
function noopGoToBilling() {}

/**
 * Where the banner's action button should send the user. Tasks and Missions
 * already have an Organization Spending page to link to; the board does not
 * have a billing destination yet, so its button is a no-op.
 */
type HostedCreditBannerAction = "spending" | "billing";

export function useHostedCreditEmptyBanner(
  organizationId: string,
  factoryKey: string,
  options: { action?: HostedCreditBannerAction } = {},
) {
  const { canAct } = usePermissions();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const level = spend.data ? hostedCreditWarningLevel(spend.data) : null;
  if (!level) {
    return undefined;
  }

  const action = options.action ?? "spending";

  return (
    <HostedCreditEmptyBanner
      level={level}
      billingEnabled={spend.data?.billingEnabled === true}
      canManageBilling={canAct("org", "update")}
      spendingHref={
        action === "spending"
          ? factorySettingsSectionPath(organizationId, factoryKey, "organization", "spending")
          : undefined
      }
      onGoToBilling={action === "billing" ? noopGoToBilling : undefined}
    />
  );
}
