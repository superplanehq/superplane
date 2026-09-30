import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  catalogStatusCheckNames,
  checksToolsAccess,
  isChecksHandlerCIIntegration,
  readyChecksHandlerIntegrationIds,
  selectedChecksUseGitHubActions,
  suggestedIntegrationsForChecks,
} from "./checksPRFeedbackSetup";
import {
  isPRFeedbackMaximumAttemptsValid,
  PR_FEEDBACK_SETTINGS_COPY,
  toggleUniqueString,
  type PRFeedbackSource,
} from "./prFeedbackSettingsModel";

/**
 * Get integration IDs for suggested integrations that are in ready state.
 * Suggested integrations are those inferred from the selected checks.
 */
function getSuggestedIntegrationIds(
  suggestedNames: string[],
  connected: Array<{ status?: { state?: string }; metadata?: { id?: string; integrationName?: string } }>,
): string[] {
  const suggested = new Set(suggestedNames.map((name) => name.toLowerCase()));
  return connected.flatMap((integration) => {
    const id = integration.metadata?.id;
    if (
      !id ||
      integration.status?.state !== "ready" ||
      !isChecksHandlerCIIntegration(integration.metadata?.integrationName)
    ) {
      return [];
    }
    const integrationName = integration.metadata?.integrationName?.trim().toLowerCase();
    if (!integrationName || !suggested.has(integrationName)) {
      return [];
    }
    return [id];
  });
}

export function useChecksPRFeedbackSetup(
  organizationId: string,
  factoryId: string,
  githubIntegrationId: string,
  repository: string,
) {
  const [step, setStep] = useState<"checks" | "tools">("checks");
  const [maximumAttempts, setMaximumAttempts] = useState(3);
  const [connectName, setConnectName] = useState<string | null>(null);
  const [error, setError] = useState<string>();
  const resources = useChecksSetupResources(organizationId, factoryId, githubIntegrationId, repository);
  const suggestedNames = useMemo(
    () => suggestedIntegrationsForChecks(resources.catalog, resources.checkNames),
    [resources.catalog, resources.checkNames],
  );
  const usesGitHubActions = useMemo(
    () => selectedChecksUseGitHubActions(resources.catalog, resources.checkNames),
    [resources.catalog, resources.checkNames],
  );
  const toolsAccess = checksToolsAccess(suggestedNames, usesGitHubActions);
  const connectDefinition = resources.available.find((integration) => integration.name === connectName) ?? null;

  const finish = (source: PRFeedbackSource) =>
    finishChecksSetup({
      repository,
      source,
      checkNames: resources.checkNames,
      maximumAttempts,
      runnerIntegrationIds: resources.runnerIntegrationIds,
      createHandler: resources.createHandler,
      setError,
    });

  return {
    step,
    setStep,
    checkNames: resources.checkNames,
    toggleCheckName: resources.toggleCheckName,
    catalog: resources.catalog,
    catalogQuery: resources.catalogQuery,
    catalogLoading: resources.catalogLoading,
    catalogEmpty: resources.catalogEmpty,
    canContinue:
      !resources.catalogLoading && resources.checkNames.length > 0 && isPRFeedbackMaximumAttemptsValid(maximumAttempts),
    maximumAttempts,
    setMaximumAttempts,
    usesGitHubActions,
    toolsAccess,
    runnerIntegrationIds: resources.runnerIntegrationIds,
    setRunnerIntegrationIds: resources.setRunnerIntegrationIds,
    suggestedNames,
    available: resources.available,
    connected: resources.connected,
    connectedQuery: resources.connectedQuery,
    connectName,
    setConnectName,
    connectDefinition,
    existingNames: resources.existingNames,
    createIntegration: resources.createIntegration,
    error,
    createHandler: resources.createHandler,
    finish,
  };
}

export type ChecksPRFeedbackSetupModel = ReturnType<typeof useChecksPRFeedbackSetup>;

export function integrationDefinitionLabel(definition: IntegrationsIntegrationDefinition | null): string {
  return definition?.label || definition?.name || "integration";
}

function useChecksSetupResources(
  organizationId: string,
  factoryId: string,
  githubIntegrationId: string,
  repository: string,
) {
  const [checkNames, setCheckNames] = useState<string[]>([]);
  const [runnerIntegrationIds, setRunnerIntegrationIds] = useState<string[]>([]);
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

  // Tracks every integration ID ever auto-selected (by the initial seed or by a later
  // suggestion), so a user's deselection is never overridden by a later effect run.
  const offeredSuggestionIds = useRef(new Set<string>());

  useEffect(() => {
    if (seeded.connected || connectedLoading) {
      return;
    }
    // Start with all ready integrations for checks handler
    const allReadyIds = readyChecksHandlerIntegrationIds(connected);
    allReadyIds.forEach((id) => offeredSuggestionIds.current.add(id));
    setRunnerIntegrationIds(allReadyIds);
    setSeeded((current) => ({ ...current, connected: true }));
  }, [connected, connectedLoading, seeded.connected]);

  // Auto-select suggested integrations when checks are selected. Each integration is only
  // offered once: if the user later deselects it, changing checks or refreshing connected
  // tools must not add it back.
  useEffect(() => {
    if (checkNames.length === 0 || seeded.connected === false || connectedLoading) {
      return;
    }
    const suggested = suggestedIntegrationsForChecks(catalog, checkNames);
    if (suggested.length === 0) {
      return;
    }
    // Get IDs of suggested integrations that are ready, excluding ones already offered before.
    const suggestedIds = getSuggestedIntegrationIds(suggested, connected).filter(
      (id) => !offeredSuggestionIds.current.has(id),
    );
    if (suggestedIds.length === 0) {
      return;
    }
    suggestedIds.forEach((id) => offeredSuggestionIds.current.add(id));
    setRunnerIntegrationIds((current) => {
      const seen = new Set(current);
      suggestedIds.forEach((id) => seen.add(id));
      return Array.from(seen);
    });
  }, [checkNames, catalog, seeded.connected, connectedLoading, connected]);

  const catalogEmpty = !catalogLoading && !catalogQuery.isError && catalog.length === 0;
  const available = (availableQuery.data ?? []).filter((integration) => isChecksHandlerCIIntegration(integration.name));
  const existingNames = useMemo(
    () => new Set(connected.map((item) => item.metadata?.name?.trim()).filter((name): name is string => Boolean(name))),
    [connected],
  );

  return {
    checkNames,
    toggleCheckName: (name: string) => setCheckNames((current) => toggleUniqueString(current, name)),
    catalog,
    catalogQuery,
    catalogLoading,
    catalogEmpty,
    runnerIntegrationIds,
    setRunnerIntegrationIds,
    available,
    connected,
    connectedQuery,
    existingNames,
    createIntegration,
    createHandler,
  };
}

async function finishChecksSetup({
  repository,
  source,
  checkNames,
  maximumAttempts,
  runnerIntegrationIds,
  createHandler,
  setError,
}: {
  repository: string;
  source: PRFeedbackSource;
  checkNames: string[];
  maximumAttempts: number;
  runnerIntegrationIds: string[];
  createHandler: ReturnType<typeof useCreateFactoryPRFeedbackHandler>;
  setError: (error: string | undefined) => void;
}) {
  if (checkNames.length === 0) {
    setError(PR_FEEDBACK_SETTINGS_COPY.checkNamesNone);
    return undefined;
  }
  setError(undefined);
  try {
    return await createHandler.mutateAsync({
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
  } catch (cause) {
    setError(getApiErrorMessage(cause, PR_FEEDBACK_SETTINGS_COPY.createError));
    return undefined;
  }
}
