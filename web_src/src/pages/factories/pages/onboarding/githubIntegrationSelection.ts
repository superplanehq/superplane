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
 * Canvas secret lookup matches app_installations.installation_name.
 * The integration id is not that name. Do not mark the selection ready until
 * the installation name is known, for GitHub and Bitbucket.
 */
export function selectionsWithSavedVcsInstallation(
  onboarding: FactoriesFactory["onboarding"],
  selections: IntegrationSelections,
  installationName: string,
): IntegrationSelections {
  const host = onboardingVcsHost(onboarding);
  const id = onboarding?.vcsIntegrationId?.trim() ?? "";
  const name = usableInstallationName(installationName, id);
  if (!host || !name) return selections;
  const current = selections[host];
  if (current?.id === id && current.name === name && current.ready) return selections;
  return { ...selections, [host]: { id, name, ready: true } };
}

export async function describeInstallationName(organizationId: string, integrationId: string): Promise<string> {
  const response = await organizationsDescribeIntegration(
    withOrganizationHeader({
      organizationId,
      path: { id: organizationId, integrationId },
    }),
  );
  return response.data?.integration?.metadata?.name?.trim() ?? "";
}

const savedInstallationNameRetryDelaysMs = [1_000, 2_000, 4_000];

function usableInstallationName(installationName: string, integrationId: string): string {
  const name = installationName.trim();
  const id = integrationId.trim();
  if (!name || !id || name === id) return "";
  return name;
}

export async function loadSavedInstallationName(
  organizationId: string,
  integrationId: string,
  options?: { cancelled?: () => boolean },
): Promise<string> {
  const id = integrationId.trim();
  if (!organizationId || !id) return "";
  for (let attempt = 0; attempt <= savedInstallationNameRetryDelaysMs.length; attempt += 1) {
    if (options?.cancelled?.()) return "";
    try {
      const name = usableInstallationName(await describeInstallationName(organizationId, id), id);
      if (name) return name;
    } catch {
      // Retry. A later attempt can read the name after the connection recovers.
    }
    const delay = savedInstallationNameRetryDelaysMs[attempt];
    if (delay === undefined) break;
    await new Promise((resolve) => {
      setTimeout(resolve, delay);
    });
  }
  return "";
}
