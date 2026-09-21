import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";

export type ProductiveSetupStep = "connection" | "project";

export function useProductiveIntakeSetup(organizationId: string, factoryId: string) {
  const [step, setStep] = useState<ProductiveSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, productiveIntegrations, productiveDefinition, existingNames } =
    useProductiveConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: Boolean(integrationId),
  });

  useEffect(() => {
    if (stayOnConnection || step !== "connection") {
      return;
    }
    const readyId = readyProductiveConnectionId(productiveIntegrations, integrationId);
    if (!readyId) {
      return;
    }
    setIntegrationId(readyId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [stayOnConnection, step, productiveIntegrations, integrationId, connectedQuery]);

  // A new connection is the reason the user opened the connect dialog, so the
  // wizard adopts it and moves on instead of asking them to pick it again.
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

  // Creating the intake is the last answer the wizard needs: the backend seeds
  // the newest open tasks of the project, the same way a GitHub intake starts.
  const createBoundIntake = async () => {
    if (!integrationId || !projectId) return;
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_PRODUCTIVE_TASKS",
        integrationId,
        resourceId: projectId,
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, PRODUCTIVE_INTAKE_SETUP_COPY.wizardCreateError));
      return false;
    }
  };

  return {
    step,
    setStep,
    integrationId,
    setIntegrationId,
    projectId,
    setProjectId,
    connectOpen,
    setConnectOpen,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    projectsQuery,
    productiveIntegrations,
    productiveDefinition,
    existingNames,
    completeConnection,
    returnToConnection,
    createBoundIntake,
  };
}

export function readyProductiveConnectionId(
  integrations: Array<{ metadata?: { id?: string } }>,
  selectedId: string,
): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
}

function useProductiveConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const productiveIntegrations = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) =>
          integration.metadata?.integrationName === "productive" &&
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
    productiveIntegrations,
    productiveDefinition: availableQuery.data?.find((integration) => integration.name === "productive"),
    existingNames,
  };
}

export type ProductiveIntakeSetupModel = ReturnType<typeof useProductiveIntakeSetup>;
