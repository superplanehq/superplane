import type { FactoriesFactory } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { useOrganization } from "@/hooks/useOrganizationData";

import type { FactorySettingsScope } from "../../lib/factoryPagePaths";

export interface FactorySettingsNavHeadingLabels {
  accountLabel: string;
  organizationName: string;
  workspaceName: string;
  workspaceKey: string;
}

interface FactorySettingsNavHeading {
  name: string;
  helper?: string;
  testId: string;
}

/** Names shown above each settings nav group: the signed-in user, the workspace, and the organization. */
export function useFactorySettingsNavHeadingLabels(
  organizationId: string,
  factory: FactoriesFactory,
  factoryKey: string,
): FactorySettingsNavHeadingLabels {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  return {
    accountLabel: account?.name?.trim() || "Account",
    organizationName: organization?.metadata?.name?.trim() || "Organization",
    workspaceName: factory.name?.trim() || "Workspace",
    workspaceKey: factory.key ?? factoryKey,
  };
}

export function factorySettingsNavGroupHeading(
  groupId: FactorySettingsScope,
  labels: FactorySettingsNavHeadingLabels,
): FactorySettingsNavHeading {
  if (groupId === "workspace") {
    return {
      name: labels.workspaceName,
      helper: `Workspace · ${labels.workspaceKey}`,
      testId: "factory-settings-workspace-heading",
    };
  }
  if (groupId === "organization") {
    return {
      name: labels.organizationName,
      helper: "Organization",
      testId: "factory-settings-organization-heading",
    };
  }
  return { name: labels.accountLabel, testId: "factory-settings-account-heading" };
}
