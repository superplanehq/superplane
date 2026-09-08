import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

export type NotionSetupStep = "connection" | "database" | "complete";

export function useNotionIntakeSetup(organizationId: string, factoryId: string, open: boolean) {
  const [step, setStep] = useState<NotionSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [databaseId, setDatabaseId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, notionIntegrations, notionDefinition, existingNames } = useNotionConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const databasesQuery = useIntegrationResources(organizationId, integrationId, "database");

  useEffect(() => {
    if (open) {
      setStep("connection");
      setIntegrationId("");
      setDatabaseId("");
      setError(undefined);
    }
  }, [open]);

  useEffect(() => {
    if (!integrationId && notionIntegrations.length === 1) {
      setIntegrationId(notionIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, notionIntegrations]);

  // A new connection is the reason the user opened the connect dialog, so the
  // wizard adopts it and moves on instead of asking them to pick it again.
  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("database");
    void connectedQuery.refetch();
  };

  // Creating the intake is the last answer the wizard needs: the backend seeds
  // the newest pages of the database, the same way a Productive.io intake starts.
  const createBoundIntake = async () => {
    if (!integrationId || !databaseId) return;
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_NOTION_PAGES",
        integrationId,
        resourceId: databaseId,
      });
      setStep("complete");
    } catch (cause) {
      setError(getApiErrorMessage(cause, "SuperPlane could not create the Notion intake."));
    }
  };

  return {
    step,
    setStep,
    integrationId,
    setIntegrationId,
    databaseId,
    setDatabaseId,
    connectOpen,
    setConnectOpen,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    databasesQuery,
    notionIntegrations,
    notionDefinition,
    existingNames,
    completeConnection,
    createBoundIntake,
  };
}

function useNotionConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const notionIntegrations = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) =>
          integration.metadata?.integrationName === "notion" &&
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
    notionIntegrations,
    notionDefinition: availableQuery.data?.find((integration) => integration.name === "notion"),
    existingNames,
  };
}

export type NotionIntakeSetupModel = ReturnType<typeof useNotionIntakeSetup>;
