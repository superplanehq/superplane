import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";
import type { IntakeSetupStep } from "./IntakeSetupWizard";

export function useDatadogIntakeSetup(organizationId: string, factoryId: string) {
  const [step, setStep] = useState<IntakeSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [serviceName, setServiceName] = useState("");
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, datadogIntegrations, datadogDefinition, existingNames } =
    useDatadogConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const servicesQuery = useIntegrationResources(organizationId, integrationId, "service", undefined, {
    enabled: Boolean(integrationId),
  });

  useEffect(() => {
    if (!integrationId && datadogIntegrations.length === 1) {
      setIntegrationId(datadogIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, datadogIntegrations]);

  useEffect(() => {
    if (stayOnConnection || step !== "connection") {
      return;
    }
    const readyId = readyDatadogConnectionId(datadogIntegrations, integrationId);
    if (!readyId) {
      return;
    }
    setIntegrationId(readyId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [stayOnConnection, step, datadogIntegrations, integrationId, connectedQuery]);

  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  };

  const returnToConnection = () => {
    setStayOnConnection(true);
    setStep("connection");
  };

  const createBoundIntake = async () => {
    const resourceId = serviceName.trim();
    if (!integrationId || !resourceId) return false;
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_DATADOG",
        integrationId,
        resourceId,
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, DATADOG_INTAKE_SETUP_COPY.wizardCreateError));
      return false;
    }
  };

  return {
    step,
    setStep,
    integrationId,
    setIntegrationId,
    serviceName,
    setServiceName,
    skipInitialImport,
    setSkipInitialImport,
    connectOpen,
    setConnectOpen,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    servicesQuery,
    datadogIntegrations,
    datadogDefinition,
    existingNames,
    completeConnection,
    returnToConnection,
    createBoundIntake,
  };
}

export function readyDatadogConnectionId(
  integrations: Array<{ metadata?: { id?: string } }>,
  selectedId: string,
): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
}

function useDatadogConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const datadogIntegrations = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) =>
          integration.metadata?.integrationName === "datadog" &&
          integration.status?.state === "ready" &&
          integration.metadata.id,
      ),
    [connectedQuery.data],
  );
  const existingNames = useMemo(
    () =>
      new Set(
        (connectedQuery.data ?? [])
          .map((integration) => integration.metadata?.name?.trim())
          .filter((name): name is string => Boolean(name)),
      ),
    [connectedQuery.data],
  );

  return {
    connectedQuery,
    datadogIntegrations,
    datadogDefinition: availableQuery.data?.find((integration) => integration.name === "datadog"),
    existingNames,
  };
}

export type DatadogIntakeSetupModel = ReturnType<typeof useDatadogIntakeSetup>;
