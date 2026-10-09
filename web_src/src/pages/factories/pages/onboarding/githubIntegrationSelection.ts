import type { FactoriesFactory } from "@/api-client";
import { organizationsDescribeIntegration } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { IntegrationSelection, IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { onboardingVcsHost } from "./onboardingStatus";

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

/**
 * A saved VCS integration was verified when its repository was selected.
 * Restore its readiness once the workspace loads, so an OAuth redirect for
 * another integration (Jira, Linear) does not block finishing setup with
 * "Connect GitHub, then select both repositories." The installation name is
 * resolved again at finish time.
 */
export function selectionsWithSavedVcsReady(
  onboarding: FactoriesFactory["onboarding"],
  selections: IntegrationSelections,
): IntegrationSelections {
  const host = onboardingVcsHost(onboarding);
  const id = onboarding?.vcsIntegrationId;
  if (!host || !id) return selections;
  const current = selections[host];
  if (current?.id === id && current.ready) return selections;
  return { ...selections, [host]: { id, name: current?.name || id, ready: true } };
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
