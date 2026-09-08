import { useEffect, useRef } from "react";

import type { OrganizationsIntegration } from "@/api-client";
import { pendingGitHubInstallations } from "@/lib/hostedGitHubInstall";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import type { IntegrationInstanceSummary } from "@/pages/home/homeIntegrationStatus";

import type { WizardStepId } from "./onboardingFixtures";

const EMPTY_GITHUB_CONNECTIONS: IntegrationInstanceSummary = {
  name: "github",
  allInstances: [],
  readyInstances: [],
};

/**
 * Resolves the GitHub connection summary for the wizard and selects the
 * connection the user just created, once, on return from GitHub.
 */
export function useOnboardingGithubConnections(args: {
  integrationData: IntegrationInstanceSummary[];
  openSection: WizardStepId;
  selectNewest: boolean;
  selections: IntegrationSelections;
  selectInstance: (integrationName: string, integrationId: string) => void;
  onConnectionSelected: (integration: OrganizationsIntegration) => void | Promise<void>;
}): IntegrationInstanceSummary {
  const githubConnections =
    args.integrationData.find((integration) => integration.name === "github") ?? EMPTY_GITHUB_CONNECTIONS;

  useSelectNewGithubConnection({
    openSection: args.openSection,
    selectNewest: args.selectNewest && !accountPickerPending(githubConnections.allInstances),
    readyInstances: githubConnections.readyInstances,
    selections: args.selections,
    selectInstance: args.selectInstance,
    onConnectionSelected: args.onConnectionSelected,
  });

  return githubConnections;
}

/**
 * True when a connect still waits for the user to pick a GitHub account. The
 * OAuth return also carries `pick=newest`, so without this check the wizard
 * would select an older ready connection and skip the account picker.
 */
function accountPickerPending(instances: OrganizationsIntegration[]): boolean {
  return instances.some(
    (instance) =>
      instance.status?.state !== "ready" && pendingGitHubInstallations(instance.status?.metadata).length >= 1,
  );
}

function newestReadyInstance(instances: OrganizationsIntegration[]): OrganizationsIntegration | undefined {
  return [...instances].sort((left, right) => {
    const leftAt = left.metadata?.createdAt ?? "";
    const rightAt = right.metadata?.createdAt ?? "";
    return rightAt.localeCompare(leftAt);
  })[0];
}

/**
 * Selects the newest ready GitHub connection after the return from GitHub, then
 * reports the selection so the wizard can continue.
 *
 * The round trip reloads the page, so the in-memory "just connected" hint is
 * gone. Runs only when the return URL asks for it (`pick=newest`), and at most
 * once, so the user can still choose a different connection afterwards.
 *
 * No other path auto-selects. An install request approved outside the round
 * trip shows the account picker again, and the user selects the account.
 */
function useSelectNewGithubConnection(args: {
  openSection: WizardStepId;
  selectNewest: boolean;
  readyInstances: IntegrationInstanceSummary["readyInstances"];
  selections: IntegrationSelections;
  selectInstance: (integrationName: string, integrationId: string) => void;
  onConnectionSelected: (integration: OrganizationsIntegration) => void | Promise<void>;
}) {
  const selectedNewConnection = useRef(false);
  const { openSection, selectNewest, readyInstances, selections, selectInstance } = args;
  const { onConnectionSelected } = args;

  useEffect(() => {
    if (selectedNewConnection.current || openSection !== "vcs") return;
    if (!selectNewest || readyInstances.length === 0) return;

    const newest = newestReadyInstance(readyInstances);
    const id = newest?.metadata?.id;
    if (!id) return;

    selectedNewConnection.current = true;
    if (selections.github?.id !== id) selectInstance("github", id);
    void onConnectionSelected(newest);
  }, [openSection, selectNewest, readyInstances, selections, selectInstance, onConnectionSelected]);
}
