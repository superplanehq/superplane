import { useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useLocation } from "react-router";

import type { FactoriesFactory } from "@/api-client";
import { factoryAppsKey, useFactoryApps } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getFactoryDefinition, INGESTION_FACTORY_ID, SENTRY_INGESTION_FACTORY_ID } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { findIngestionFactoryApp, installedIngestionFactoryIds } from "./ingestionApp";

export const INGESTION_AUTOMATION_IDS = [INGESTION_FACTORY_ID, SENTRY_INGESTION_FACTORY_ID] as const;
export type IngestionAutomationId = (typeof INGESTION_AUTOMATION_IDS)[number];

function initialSelections(onboarding: FactoriesFactory["onboarding"]): IntegrationSelections {
  const selections: IntegrationSelections = {};
  if (onboarding?.vcsIntegrationId) {
    selections.github = {
      id: onboarding.vcsIntegrationId,
      name: onboarding.vcsIntegrationId,
      ready: false,
    };
  }
  if (onboarding?.agentIntegrationId) {
    selections.claude = {
      id: onboarding.agentIntegrationId,
      name: onboarding.agentIntegrationId,
      ready: false,
    };
  }
  return selections;
}

export function useIngestionSetup() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { data: apps = [] } = useFactoryApps(organizationId, factoryId);
  const { installFactory, isInstalling } = useInstallFactory();
  const [selections, setSelections] = useState<IntegrationSelections>(() => initialSelections(factory?.onboarding));

  const { requestConnect, dialogs, integrationsLoading } = useIntegrationConnectDialog({
    organizationId,
    returnTo: location.pathname,
    integrationNames: ["github", "claude", "sentry"],
    selections,
    onSelectionsChange: setSelections,
  });

  const installedAutomationIds = useMemo(() => installedIngestionFactoryIds(apps), [apps]);
  const installedAutomationApps = useMemo(
    () =>
      new Map(
        INGESTION_AUTOMATION_IDS.map((automationId) => [automationId, findIngestionFactoryApp(apps, automationId)]),
      ),
    [apps],
  );

  function missingIntegration(automationId: IngestionAutomationId): string | undefined {
    return getFactoryDefinition(automationId).integrations.find((name) => !selections[name]?.ready);
  }

  async function setupAutomation(automationId: IngestionAutomationId): Promise<boolean> {
    const missing = missingIntegration(automationId);
    if (missing) {
      requestConnect(missing);
      return false;
    }

    const appRepository = factory?.onboarding?.appRepository;
    const backlogRepository = factory?.onboarding?.backlogRepository ?? appRepository;
    if (!appRepository || !backlogRepository) {
      showErrorToast("Select the app and backlog repositories before you set up ingestion.");
      return false;
    }

    const installed = await installFactory({
      factoryId: automationId,
      workspaceFactoryId: factoryId,
      integrations: selections,
      installParams: { appRepository, backlogRepository },
      startingTaskPrompt: "",
      navigateOnComplete: false,
      startInitialRun: false,
    });
    if (!installed) return false;

    await queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    showSuccessToast(`${getFactoryDefinition(automationId).title} installed.`);
    return true;
  }

  return {
    dialogs,
    installedAutomationApps,
    installedAutomationIds,
    integrationsLoading,
    isInstalling,
    missingIntegration,
    setupAutomation,
  };
}
