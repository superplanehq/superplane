import { useParams } from "react-router";

import { useOrganizationSpendingReport } from "@/hooks/useOrganizationSpendingReport";

import { OrganizationSpendingExplorer } from "./spending-redesign/OrganizationSpendingExplorer";

export function OrganizationSettingsWorkspaceUsagePage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();

  return <OrganizationSpendingExplorer organizationId={organizationId} useReport={useOrganizationSpendingReport} />;
}
