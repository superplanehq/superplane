import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import {
  catalogStatusCheckNames,
  checksToolsAccess,
  isChecksHandlerCIIntegration,
  readyChecksHandlerIntegrationIds,
  selectedChecksUseGitHubActions,
  suggestedIntegrationsForChecks,
} from "./checksPRFeedbackSetup";
import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackSource } from "./prFeedbackSettingsModel";

export function useChecksPRFeedbackSetup(
  organizationId: string,
  factoryId: string,
  githubIntegrationId: string,
  repository: string,
) {
  const [step, setStep] = useState<"checks" | "tools">("checks");
  const [checkNames, setCheckNames] = useState<string[]>([]);
  const [maximumAttempts, setMaximumAttempts] = useState(3);
  const [runnerIntegrationIds, setRunnerIntegrationIds] = useState<string[]>([]);
  const [connectName, setConnectName] = useState<string | null>(null);
  const [error, setError] = useState<string>();
  const [seeded, setSeeded] = useState({ catalog: false, connected: false });

  const catalogParameters = repository.trim() ? { repository: repository.trim() } : undefined;
  const catalogQuery = useIntegrationResources(organizationId, githubIntegrationId, "status_check", catalogParameters);
  const catalog = useMemo(() => catalogQuery.data ?? [], [catalogQuery.data]);
  const catalogLoading = catalogQuery.isPending || catalogQuery.isFetching;
  const connectedQuery = useConnectedIntegrations(organizationId, { enabled: Boolean(organizationId) });
  const availableQuery = useAvailableIntegrations({ enabled: Boolean(organizationId), organizationId });
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);
  const connected = useMemo(() => connectedQuery.data ?? [], [connectedQuery.data]);
  const connectedLoading = connectedQuery.isPending || connectedQuery.isFetching;

  useEffect(() => {
    if (seeded.catalog || catalogLoading) {
      return;
    }
    setCheckNames(catalogStatusCheckNames(catalog));
    setSeeded((current) => ({ ...current, catalog: true }));
  }, [catalog, catalogLoading, seeded.catalog]);

  useEffect(() => {
    if (seeded.connected || connectedLoading) {
      return;
    }
    setRunnerIntegrationIds(readyChecksHandlerIntegrationIds(connected));
    setSeeded((current) => ({ ...current, connected: true }));
  }, [connected, connectedLoading, seeded.connected]);

  const suggestedNames = useMemo(() => suggestedIntegrationsForChecks(catalog, checkNames), [catalog, checkNames]);
  const usesGitHubActions = useMemo(() => selectedChecksUseGitHubActions(catalog, checkNames), [catalog, checkNames]);
  const toolsAccess = checksToolsAccess(suggestedNames, usesGitHubActions);
  const catalogEmpty = !catalogLoading && !catalogQuery.isError && catalog.length === 0;
  const available = (availableQuery.data ?? []).filter((integration) => isChecksHandlerCIIntegration(integration.name));
  const existingNames = useMemo(
    () => new Set(connected.map((item) => item.metadata?.name?.trim()).filter((name): name is string => Boolean(name))),
    [connected],
  );
  const connectDefinition = available.find((integration) => integration.name === connectName) ?? null;

  const toggleCheckName = (name: string) => {
    setCheckNames((current) => toggleUniqueString(current, name));
  };

  const finish = async (source: PRFeedbackSource) => {
    if (checkNames.length === 0) {
      setError(PR_FEEDBACK_SETTINGS_COPY.checkNamesNone);
      return undefined;
    }
    setError(undefined);
    try {
      const handler = await createHandler.mutateAsync({
        source: "SOURCE_PULL_REQUEST_CHECKS",
        name: source.defaultName,
        settings: {
          subject: repository ? { repository } : undefined,
          checks: {
            names: checkNames,
            maximumAttempts,
            runnerIntegrationIds,
          },
        },
      });
      return handler;
    } catch (cause) {
      setError(getApiErrorMessage(cause, PR_FEEDBACK_SETTINGS_COPY.createError));
      return undefined;
    }
  };

  return {
    step,
    setStep,
    checkNames,
    toggleCheckName,
    catalog,
    catalogQuery,
    catalogLoading,
    catalogEmpty,
    canContinue: !catalogLoading && checkNames.length > 0 && maximumAttempts >= 1 && maximumAttempts <= 10,
    maximumAttempts,
    setMaximumAttempts,
    usesGitHubActions,
    toolsAccess,
    runnerIntegrationIds,
    setRunnerIntegrationIds,
    suggestedNames,
    available,
    connected,
    connectedQuery,
    connectName,
    setConnectName,
    connectDefinition,
    existingNames,
    createIntegration,
    error,
    createHandler,
    finish,
  };
}

export type ChecksPRFeedbackSetupModel = ReturnType<typeof useChecksPRFeedbackSetup>;

export function integrationDefinitionLabel(definition: IntegrationsIntegrationDefinition | null): string {
  return definition?.label || definition?.name || "integration";
}
