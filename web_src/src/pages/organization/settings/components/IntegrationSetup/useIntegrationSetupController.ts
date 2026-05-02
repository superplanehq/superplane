import type {
  IntegrationsCapabilityDefinition,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useDeleteIntegration,
  useIntegration,
  useNextIntegrationSetupStep,
  usePreviousIntegrationSetupStep,
} from "@/hooks/useIntegrations";
import { buildIntegrationCapabilityGroupSections } from "@/lib/capabilities";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { useIntegrationSetupActions } from "./useIntegrationSetupActions";
import { applyResumeDescribeIfChanged, canRevertSetupStep, getCurrentSetupStep, getNextIntegrationName } from "./lib";

type IntegrationSetupRouteState = {
  integrationId?: string;
};

export function useIntegrationSetupController(organizationId: string) {
  const route = useIntegrationSetupRoute(organizationId);
  const queries = useIntegrationSetupQueries(organizationId, route.setupIntegrationId);
  const metadata = useIntegrationSetupMetadata({
    availableIntegrations: queries.availableIntegrations,
    connectedIntegrations: queries.connectedIntegrations,
    integrationName: route.integrationName,
  });
  const state = useIntegrationSetupLocalState({
    integrationName: route.integrationName,
    integrationCapabilities: metadata.integrationCapabilities,
    existingIntegrationNames: metadata.existingIntegrationNames,
    setupIntegrationId: route.setupIntegrationId,
    resumeIntegrationDescribe: queries.resumeIntegrationDescribe,
  });
  const progress = useIntegrationSetupProgress(state.createdIntegration, metadata.integrationLabel);
  const mutations = useIntegrationSetupMutations(organizationId, state.createdIntegration);
  const actions = useIntegrationSetupActions({
    route,
    state,
    progress,
    metadata,
    mutations,
  });

  usePageTitle(["Integrations", progress.setupPageTitle]);
  useSetupCompletionRedirect({ organizationId, route, queries, state, progress });

  return {
    organizationId,
    route,
    queries,
    metadata,
    state,
    progress,
    mutations,
    actions,
  };
}

function useIntegrationSetupRoute(organizationId: string) {
  const navigate = useNavigate();
  const location = useLocation();
  const { integrationName: routeIntegrationName } = useParams<{ integrationName: string }>();
  const integrationName = routeIntegrationName || "";
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const routeState = location.state as IntegrationSetupRouteState | null;

  return {
    navigate,
    integrationName,
    integrationsHref,
    setupIntegrationId: routeState?.integrationId,
  };
}

function useIntegrationSetupQueries(organizationId: string, setupIntegrationId?: string) {
  const { data: availableIntegrations = [], isLoading: isAvailableIntegrationsLoading } = useAvailableIntegrations();
  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId);
  const { data: resumeIntegrationDescribe, isPending: isResumeDescribePending } = useIntegration(
    organizationId,
    setupIntegrationId || "",
  );

  return {
    availableIntegrations,
    connectedIntegrations,
    resumeIntegrationDescribe,
    isResumeDescribePending,
    isAvailableIntegrationsLoading,
  };
}

interface IntegrationSetupMetadataParams {
  availableIntegrations: IntegrationsIntegrationDefinition[];
  connectedIntegrations: OrganizationsIntegration[];
  integrationName: string;
}

function useIntegrationSetupMetadata({
  availableIntegrations,
  connectedIntegrations,
  integrationName,
}: IntegrationSetupMetadataParams) {
  const integrationDefinition = useMemo(
    () => availableIntegrations.find((integration) => integration.name === integrationName),
    [availableIntegrations, integrationName],
  );
  const integrationLabel =
    integrationDefinition?.label || getIntegrationTypeDisplayName(undefined, integrationName) || integrationName;
  const integrationCapabilities = useMemo(
    () => getIntegrationCapabilities(integrationDefinition),
    [integrationDefinition],
  );
  const capabilitySections = useMemo(
    () => buildIntegrationCapabilityGroupSections(integrationDefinition, integrationCapabilities),
    [integrationDefinition, integrationCapabilities],
  );
  const capabilityByName = useMemo(() => getCapabilityByName(integrationCapabilities), [integrationCapabilities]);
  const existingIntegrationNames = useMemo(
    () => getExistingIntegrationNames(connectedIntegrations),
    [connectedIntegrations],
  );

  return {
    integrationDefinition,
    integrationLabel,
    integrationCapabilities,
    capabilitySections,
    capabilityByName,
    existingIntegrationNames,
  };
}

interface IntegrationSetupLocalStateParams {
  integrationName: string;
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  existingIntegrationNames: Set<string>;
  setupIntegrationId?: string;
  resumeIntegrationDescribe?: OrganizationsIntegration | null;
}

function useIntegrationSetupLocalState({
  integrationName,
  integrationCapabilities,
  existingIntegrationNames,
  setupIntegrationId,
  resumeIntegrationDescribe,
}: IntegrationSetupLocalStateParams) {
  const lastResumeDescribeKey = useRef<string | null>(null);
  const [instanceName, setInstanceName] = useState("");
  const [createdIntegration, setCreatedIntegration] = useState<OrganizationsIntegration | null>(null);
  const [stepInputs, setStepInputs] = useState<Record<string, unknown>>({});
  const [selectedCapabilities, setSelectedCapabilities] = useState<Set<string>>(new Set());
  const setters = useMemo(
    () => ({
      setCreatedIntegration,
      setInstanceName,
      setSelectedCapabilities,
      setStepInputs,
    }),
    [],
  );

  useResetSetupState(integrationName, lastResumeDescribeKey, setters);
  useDefaultSelectedCapabilities(createdIntegration, integrationCapabilities, setSelectedCapabilities);
  useDefaultInstanceName(instanceName, integrationName, existingIntegrationNames, setInstanceName);
  useResumeIntegrationDescribe(setupIntegrationId, resumeIntegrationDescribe, lastResumeDescribeKey, setters);

  return {
    instanceName,
    setInstanceName,
    createdIntegration,
    setCreatedIntegration,
    stepInputs,
    setStepInputs,
    selectedCapabilities,
    setSelectedCapabilities,
  };
}

function useIntegrationSetupProgress(createdIntegration: OrganizationsIntegration | null, integrationLabel: string) {
  const currentStep = getCurrentSetupStep(createdIntegration);
  const canRevertCurrentStep = canRevertSetupStep(createdIntegration);
  const integrationReady = createdIntegration?.status?.state === "ready";
  const showSetupStepBack = Boolean(createdIntegration) && (!integrationReady || canRevertCurrentStep);
  const setupPageTitle = useMemo(
    () => getSetupPageTitle(createdIntegration, currentStep, integrationLabel),
    [createdIntegration, currentStep, integrationLabel],
  );

  return {
    currentStep,
    canRevertCurrentStep,
    integrationReady,
    showSetupStepBack,
    setupPageTitle,
  };
}

function useIntegrationSetupMutations(organizationId: string, createdIntegration: OrganizationsIntegration | null) {
  const createMutation = useCreateIntegration(organizationId, "integrations_page");
  const submitStepMutation = useNextIntegrationSetupStep(organizationId);
  const revertStepMutation = usePreviousIntegrationSetupStep(organizationId);
  const deleteIntegrationMutation = useDeleteIntegration(organizationId, createdIntegration?.metadata?.id ?? "");

  return {
    createMutation,
    submitStepMutation,
    revertStepMutation,
    deleteIntegrationMutation,
  };
}

export type IntegrationSetupRoute = ReturnType<typeof useIntegrationSetupRoute>;
export type IntegrationSetupMetadata = ReturnType<typeof useIntegrationSetupMetadata>;
export type IntegrationSetupState = ReturnType<typeof useIntegrationSetupLocalState>;
export type IntegrationSetupProgress = ReturnType<typeof useIntegrationSetupProgress>;
export type IntegrationSetupMutations = ReturnType<typeof useIntegrationSetupMutations>;

interface ResetStateSetters {
  setCreatedIntegration: Dispatch<SetStateAction<OrganizationsIntegration | null>>;
  setInstanceName: Dispatch<SetStateAction<string>>;
  setSelectedCapabilities: Dispatch<SetStateAction<Set<string>>>;
  setStepInputs: Dispatch<SetStateAction<Record<string, unknown>>>;
}

function useResetSetupState(
  integrationName: string,
  lastResumeDescribeKey: MutableRefObject<string | null>,
  setters: ResetStateSetters,
) {
  useEffect(() => {
    lastResumeDescribeKey.current = null;
    setters.setCreatedIntegration(null);
    setters.setStepInputs({});
    setters.setInstanceName("");
    setters.setSelectedCapabilities(new Set());
  }, [integrationName, lastResumeDescribeKey, setters]);
}

function useDefaultSelectedCapabilities(
  createdIntegration: OrganizationsIntegration | null,
  integrationCapabilities: IntegrationsCapabilityDefinition[],
  setSelectedCapabilities: Dispatch<SetStateAction<Set<string>>>,
) {
  useEffect(() => {
    if (createdIntegration) {
      return;
    }

    setSelectedCapabilities((current) => {
      if (current.size > 0) {
        return current;
      }

      return new Set(integrationCapabilities.map((capability) => capability.name).filter(Boolean) as string[]);
    });
  }, [integrationCapabilities, createdIntegration, setSelectedCapabilities]);
}

function useDefaultInstanceName(
  instanceName: string,
  integrationName: string,
  existingIntegrationNames: Set<string>,
  setInstanceName: Dispatch<SetStateAction<string>>,
) {
  useEffect(() => {
    if (instanceName || !integrationName) {
      return;
    }

    setInstanceName(getNextIntegrationName(integrationName, existingIntegrationNames));
  }, [instanceName, integrationName, existingIntegrationNames, setInstanceName]);
}

function useResumeIntegrationDescribe(
  setupIntegrationId: string | undefined,
  resumeIntegrationDescribe: OrganizationsIntegration | null | undefined,
  lastResumeDescribeKey: MutableRefObject<string | null>,
  setters: ResetStateSetters,
) {
  useEffect(() => {
    applyResumeDescribeIfChanged(setupIntegrationId, resumeIntegrationDescribe, lastResumeDescribeKey, (describe) => {
      setters.setCreatedIntegration(describe);
      setters.setStepInputs({});
      setters.setSelectedCapabilities(new Set());
      setters.setInstanceName(describe.metadata?.name || describe.metadata?.integrationName || "");
    });
  }, [setupIntegrationId, resumeIntegrationDescribe, lastResumeDescribeKey, setters]);
}

interface CompletionRedirectParams {
  organizationId: string;
  route: ReturnType<typeof useIntegrationSetupRoute>;
  queries: ReturnType<typeof useIntegrationSetupQueries>;
  state: ReturnType<typeof useIntegrationSetupLocalState>;
  progress: ReturnType<typeof useIntegrationSetupProgress>;
}

function useSetupCompletionRedirect({ organizationId, route, queries, state, progress }: CompletionRedirectParams) {
  useEffect(() => {
    const id = state.createdIntegration?.metadata?.id;
    if (!id || progress.currentStep) {
      return;
    }

    if (
      route.setupIntegrationId &&
      id === route.setupIntegrationId &&
      (queries.isResumeDescribePending || !queries.resumeIntegrationDescribe)
    ) {
      return;
    }

    route.navigate(`/${organizationId}/settings/integrations/${id}`, { replace: true });
  }, [organizationId, route, queries, state.createdIntegration, progress.currentStep]);
}

function getIntegrationCapabilities(integrationDefinition?: IntegrationsIntegrationDefinition) {
  return [...(integrationDefinition?.capabilities || [])]
    .filter((capability) => Boolean(capability.name))
    .sort((left, right) => left.label!.localeCompare(right.label!));
}

function getCapabilityByName(integrationCapabilities: IntegrationsCapabilityDefinition[]) {
  const map = new Map<string, IntegrationsCapabilityDefinition>();
  for (const capability of integrationCapabilities) {
    if (capability.name) {
      map.set(capability.name, capability);
    }
  }
  return map;
}

function getExistingIntegrationNames(connectedIntegrations: OrganizationsIntegration[]) {
  return new Set(
    connectedIntegrations
      .map((integration) => integration.metadata?.name?.trim())
      .filter((name): name is string => Boolean(name)),
  );
}

function getSetupPageTitle(
  createdIntegration: OrganizationsIntegration | null,
  currentStep: ReturnType<typeof getCurrentSetupStep>,
  integrationLabel: string,
) {
  if (!createdIntegration) {
    return `Set up ${integrationLabel}`;
  }

  if (currentStep) {
    const label = currentStep.label?.trim();
    if (currentStep.type === "DONE") {
      return label || "Setup complete";
    }
    if (label) {
      return label;
    }
  }

  return `Setup ${integrationLabel}`;
}
