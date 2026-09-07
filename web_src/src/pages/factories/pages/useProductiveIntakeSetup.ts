import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

export type ProductiveSetupStep = "connection" | "project" | "complete";

export function useProductiveIntakeSetup(organizationId: string, factoryId: string, open: boolean) {
  const [step, setStep] = useState<ProductiveSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, productiveIntegrations, productiveDefinition, existingNames } =
    useProductiveConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project");

  useEffect(() => {
    if (open) {
      setStep("connection");
      setIntegrationId("");
      setProjectId("");
      setError(undefined);
    }
  }, [open]);

  useEffect(() => {
    if (!integrationId && productiveIntegrations.length === 1) {
      setIntegrationId(productiveIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, productiveIntegrations]);

  // A new connection is the reason the user opened the connect dialog, so the
  // wizard adopts it and moves on instead of asking them to pick it again.
  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
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
      setStep("complete");
    } catch (cause) {
      setError(getApiErrorMessage(cause, "SuperPlane could not create the Productive.io intake."));
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
    createBoundIntake,
  };
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
