import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { usesHostedLinearOAuth } from "@/lib/integrations";
import { startDirectLinearConnect } from "@/lib/startDirectLinearConnect";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "react-router";

import { addIntakeLabel, toggleIntakeLabel } from "./intakeSourceSettingsModel";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export type LinearSetupStep = "connection" | "project";

export function useLinearIntakeSetup(organizationId: string, factoryId: string, selectIntegrationId = "") {
  const [step, setStep] = useState<LinearSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectIds, setProjectIds] = useState<string[]>([]);
  const [labels, setLabels] = useState<string[]>([]);
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();
  const pickedReturnedConnection = useRef(false);

  const { connectedQuery, availableQuery, linearIntegrations, linearConnections, linearDefinition, existingNames } =
    useLinearConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: Boolean(integrationId),
  });

  useAdvanceLinearSetup({
    selectIntegrationId,
    pickedReturnedConnection,
    linearConnections,
    linearIntegrations,
    integrationId,
    stayOnConnection,
    step,
    connectedQuery,
    setIntegrationId,
    setConnectOpen,
    setStep,
  });

  const toggleProject = (id: string) => {
    setProjectIds((current) => (current.includes(id) ? current.filter((entry) => entry !== id) : [...current, id]));
  };
  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  };

  const { connecting, connectLinear } = useLinearConnect({
    organizationId,
    integrations: linearIntegrations,
    existingNames,
    linearDefinition,
    definitionLoading: availableQuery.isLoading,
    createIntegration,
    completeConnection,
    setConnectOpen,
    setError,
  });

  const returnToConnection = () => {
    setStayOnConnection(true);
    setStep("connection");
  };
  const createBoundIntake = async () => {
    if (!integrationId || projectIds.length === 0) {
      return false;
    }
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_LINEAR_ISSUES",
        integrationId,
        settings: {
          linearProjectIds: projectIds,
          linearLabels: labels,
        },
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, LINEAR_INTAKE_SETUP_COPY.wizardCreateError));
      return false;
    }
  };

  return {
    organizationId,
    step,
    setStep,
    integrationId,
    setIntegrationId,
    projectIds,
    toggleProject,
    labels,
    addLabel: (label: string) => setLabels((current) => addIntakeLabel(current, label)),
    removeLabel: (label: string) => setLabels((current) => toggleIntakeLabel(current, label)),
    setLabels,
    skipInitialImport,
    setSkipInitialImport,
    connectOpen,
    setConnectOpen,
    connecting,
    hosted: usesHostedLinearOAuth(linearDefinition),
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    projectsQuery,
    linearIntegrations,
    linearDefinition,
    existingNames,
    completeConnection,
    returnToConnection,
    connectLinear,
    createBoundIntake,
  };
}

function useAdvanceLinearSetup({
  selectIntegrationId,
  pickedReturnedConnection,
  linearConnections,
  linearIntegrations,
  integrationId,
  stayOnConnection,
  step,
  connectedQuery,
  setIntegrationId,
  setConnectOpen,
  setStep,
}: {
  selectIntegrationId: string;
  pickedReturnedConnection: { current: boolean };
  linearConnections: Array<{ metadata?: { id?: string }; status?: { state?: string } }>;
  linearIntegrations: Array<{ metadata?: { id?: string } }>;
  integrationId: string;
  stayOnConnection: boolean;
  step: LinearSetupStep;
  connectedQuery: { refetch: () => Promise<unknown> };
  setIntegrationId: (id: string) => void;
  setConnectOpen: (open: boolean) => void;
  setStep: (step: LinearSetupStep) => void;
}) {
  useEffect(() => {
    pickedReturnedConnection.current = false;
  }, [selectIntegrationId, pickedReturnedConnection]);

  useEffect(() => {
    if (!selectIntegrationId || pickedReturnedConnection.current) {
      return;
    }

    const returned = linearConnections.find((integration) => integration.metadata?.id === selectIntegrationId);
    if (!returned || returned.status?.state !== "ready") {
      return;
    }

    pickedReturnedConnection.current = true;
    setIntegrationId(selectIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [
    selectIntegrationId,
    linearConnections,
    connectedQuery,
    pickedReturnedConnection,
    setIntegrationId,
    setConnectOpen,
    setStep,
  ]);

  useEffect(() => {
    if (stayOnConnection || step !== "connection" || selectIntegrationId) {
      return;
    }
    const readyId = readyLinearConnectionId(linearIntegrations, integrationId);
    if (!readyId) {
      return;
    }
    setIntegrationId(readyId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [
    stayOnConnection,
    step,
    selectIntegrationId,
    linearIntegrations,
    integrationId,
    connectedQuery,
    setIntegrationId,
    setConnectOpen,
    setStep,
  ]);
}

export function readyLinearConnectionId(
  integrations: Array<{ metadata?: { id?: string } }>,
  selectedId: string,
): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
}

type LinearConnectParams = {
  organizationId: string;
  integrations: Array<{ metadata?: { id?: string } }>;
  existingNames: Set<string>;
  linearDefinition?: IntegrationsIntegrationDefinition;
  definitionLoading: boolean;
  createIntegration: ReturnType<typeof useCreateIntegration>;
  completeConnection: (integrationId: string) => void;
  setConnectOpen: (open: boolean) => void;
  setError: (message?: string) => void;
};

function useLinearConnect(params: LinearConnectParams) {
  const location = useLocation();
  const [connecting, setConnecting] = useState(false);
  const returnPath = `${location.pathname}${location.search}`;

  const connectLinear = async () => {
    params.setError(undefined);
    if (params.definitionLoading) {
      return;
    }
    if (!usesHostedLinearOAuth(params.linearDefinition)) {
      params.setConnectOpen(true);
      return;
    }

    setConnecting(true);
    try {
      await startDirectLinearConnect({
        organizationId: params.organizationId,
        returnTo: returnPath,
        existingNames: params.existingNames,
        connected: params.integrations,
        onExistingReady: params.completeConnection,
        create: async (payload) => {
          const response = await params.createIntegration.mutateAsync(payload);
          return response.data;
        },
      });
    } catch (cause) {
      params.setError(getApiErrorMessage(cause, LINEAR_INTAKE_SETUP_COPY.wizardConnectError));
    } finally {
      setConnecting(false);
    }
  };

  return { connecting, connectLinear };
}

function useLinearConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const linearConnections = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) => integration.metadata?.integrationName === "linear" && integration.metadata.id,
      ),
    [connectedQuery.data],
  );
  const linearIntegrations = useMemo(
    () => linearConnections.filter((integration) => integration.status?.state === "ready"),
    [linearConnections],
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
    availableQuery,
    linearIntegrations,
    linearConnections,
    linearDefinition: availableQuery.data?.find((integration) => integration.name === "linear"),
    existingNames,
  };
}

export type LinearIntakeSetupModel = ReturnType<typeof useLinearIntakeSetup>;
