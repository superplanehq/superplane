import { useEffect, useRef } from "react";

import type { OrganizationsIntegration } from "@/api-client";
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
  selectSingleInitial: boolean;
  selections: IntegrationSelections;
  selectInstance: (integrationName: string, integrationId: string) => void;
  onConnectionSelected: (integration: OrganizationsIntegration) => void | Promise<void>;
}): IntegrationInstanceSummary {
  const githubConnections =
    args.integrationData.find((integration) => integration.name === "github") ?? EMPTY_GITHUB_CONNECTIONS;

  useSelectNewGithubConnection({
    openSection: args.openSection,
    selectNewest: args.selectNewest,
    selectSingleInitial: args.selectSingleInitial,
    readyInstances: githubConnections.readyInstances,
    selections: args.selections,
    selectInstance: args.selectInstance,
    onConnectionSelected: args.onConnectionSelected,
  });

  return githubConnections;
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
 * gone. Runs when the return URL asks for it (`pick=newest`), and at most
 * once, so the user can still choose a different connection afterwards.
 *
 * Initial account onboarding also runs it for a single ready connection
 * without the URL hint (`selectSingleInitial`). An install request approved
 * later binds the connection outside the wizard round trip, so the return URL
 * hint is gone when the user comes back. The selection callback saves the
 * connection and names the organization after the GitHub account, so it must
 * still run on that visit.
 */
function useSelectNewGithubConnection(args: {
  openSection: WizardStepId;
  selectNewest: boolean;
  selectSingleInitial: boolean;
  readyInstances: IntegrationInstanceSummary["readyInstances"];
  selections: IntegrationSelections;
  selectInstance: (integrationName: string, integrationId: string) => void;
  onConnectionSelected: (integration: OrganizationsIntegration) => void | Promise<void>;
}) {
  const selectedNewConnection = useRef(false);
  const { openSection, selectNewest, selectSingleInitial, readyInstances, selections, selectInstance } = args;
  const { onConnectionSelected } = args;

  useEffect(() => {
    if (selectedNewConnection.current || openSection !== "vcs") return;
    const selectSingle = selectSingleInitial && readyInstances.length === 1;
    if (!selectNewest && !selectSingle) return;
    if (readyInstances.length === 0) return;

    const newest = newestReadyInstance(readyInstances);
    const id = newest?.metadata?.id;
    if (!id) return;

    selectedNewConnection.current = true;
    if (selections.github?.id !== id) selectInstance("github", id);
    void onConnectionSelected(newest);
  }, [
    openSection,
    selectNewest,
    selectSingleInitial,
    readyInstances,
    selections,
    selectInstance,
    onConnectionSelected,
  ]);
}
