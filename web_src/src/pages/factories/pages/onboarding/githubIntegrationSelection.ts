import { organizationsDescribeIntegration } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { IntegrationSelection, IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

// Canvas secret lookup matches app_installations.installation_name.
// The product label "GitHub" is not that name.
export function githubIntegrationSelection(integrationId: string, installationName: string): IntegrationSelection {
  const id = integrationId.trim();
  const name = installationName.trim();
  if (!id || !name) {
    throw new Error("GitHub integration name is missing");
  }
  return { id, name, ready: true };
}

export function selectionsWithGitHubInstallation(
  selections: IntegrationSelections,
  installationName: string,
): IntegrationSelections {
  const github = selections.github;
  if (!github?.id) return selections;
  return {
    ...selections,
    github: githubIntegrationSelection(github.id, installationName),
  };
}

export async function describeGitHubInstallationName(organizationId: string, integrationId: string): Promise<string> {
  const response = await organizationsDescribeIntegration(
    withOrganizationHeader({
      organizationId,
      path: { id: organizationId, integrationId },
    }),
  );
  return response.data?.integration?.metadata?.name?.trim() ?? "";
}
