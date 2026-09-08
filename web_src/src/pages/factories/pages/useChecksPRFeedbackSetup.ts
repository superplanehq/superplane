import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useCreateFactoryPRFeedbackHandler, useFactoryRepositoryStatusChecks } from "@/hooks/useFactoryPRFeedbackData";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import {
  CHECKS_HANDLER_SKIP_INTEGRATIONS,
  catalogStatusCheckNames,
  suggestedIntegrationsForChecks,
} from "./checksPRFeedbackSetup";
import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackSource } from "./prFeedbackSettingsModel";

export function useChecksPRFeedbackSetup(organizationId: string, factoryId: string, repository: string, open: boolean) {
  const [step, setStep] = useState<"checks" | "tools">("checks");
  const [checkNames, setCheckNames] = useState<string[]>([]);
  const [runnerIntegrationIds, setRunnerIntegrationIds] = useState<string[]>([]);
  const [connectName, setConnectName] = useState<string | null>(null);
  const [error, setError] = useState<string>();
  const [catalogApplied, setCatalogApplied] = useState(false);

  const catalogQuery = useFactoryRepositoryStatusChecks(organizationId, factoryId, repository, { enabled: open });
  const catalog = catalogQuery.data ?? [];
  const catalogLoading = catalogQuery.isPending || catalogQuery.isFetching;
  const connectedQuery = useConnectedIntegrations(organizationId, { enabled: open && Boolean(organizationId) });
  const availableQuery = useAvailableIntegrations({ enabled: open && Boolean(organizationId), organizationId });
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);

  useEffect(() => {
    if (!open) {
      return;
    }
    setStep("checks");
    setCheckNames([]);
    setRunnerIntegrationIds([]);
    setConnectName(null);
    setError(undefined);
    setCatalogApplied(false);
  }, [open]);

  useEffect(() => {
    if (!open || catalogApplied || catalogLoading) {
      return;
    }
    setCheckNames(catalogStatusCheckNames(catalog));
    setCatalogApplied(true);
  }, [catalog, catalogApplied, catalogLoading, open]);

  const suggestedNames = useMemo(() => suggestedIntegrationsForChecks(catalog, checkNames), [catalog, checkNames]);
  const connected = connectedQuery.data ?? [];
  const available = (availableQuery.data ?? []).filter(
    (integration) => integration.name && !CHECKS_HANDLER_SKIP_INTEGRATIONS.has(integration.name),
  );
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
            maximumAttempts: 3,
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
    canContinue: !catalogLoading && checkNames.length > 0,
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
