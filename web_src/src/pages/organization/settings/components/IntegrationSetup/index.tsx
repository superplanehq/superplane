import type { IntegrationsCapabilityDefinition, OrganizationsIntegration } from "@/api-client";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Loader2, MoveLeft } from "lucide-react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { buildIntegrationCapabilityGroupSections } from "@/lib/capabilities";
import { cn } from "@/lib/utils";
import { showErrorToast } from "@/lib/toast";
import { DoneStep } from "./DoneStep";
import { InputsStep } from "./InputsStep";
import { PreCreateIntegrationSetup } from "./PreCreateIntegrationSetup";
import {
  applyResumeDescribeIfChanged,
  canRevertSetupStep,
  getCurrentSetupStep,
  getGroupToggleState,
  getNextIntegrationName,
} from "./integrationSetupHelpers";
import { RedirectPromptStep } from "./RedirectPromptStep";
import { StepHistory } from "./StepHistory";
import { openRedirectPrompt } from "@/lib/integrations";

interface IntegrationSetupProps {
  organizationId: string;
}

type IntegrationSetupRouteState = {
  integrationId?: string;
};

export function IntegrationSetup({ organizationId }: IntegrationSetupProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { integrationName: routeIntegrationName } = useParams<{ integrationName: string }>();
  const integrationName = routeIntegrationName || "";
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const routeState = location.state as IntegrationSetupRouteState | null;
  const setupIntegrationId = routeState?.integrationId;

  const { data: availableIntegrations = [], isLoading: isAvailableIntegrationsLoading } = useAvailableIntegrations();
  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId);
  const { data: resumeIntegrationDescribe, isPending: isResumeDescribePending } = useIntegration(
    organizationId,
    setupIntegrationId || "",
  );
  const lastResumeDescribeKey = useRef<string | null>(null);

  const createMutation = useCreateIntegration(organizationId, "integrations_page");
  const submitStepMutation = useNextIntegrationSetupStep(organizationId);
  const revertStepMutation = usePreviousIntegrationSetupStep(organizationId);

  const [instanceName, setInstanceName] = useState("");
  const [createdIntegration, setCreatedIntegration] = useState<OrganizationsIntegration | null>(null);
  const [stepInputs, setStepInputs] = useState<Record<string, unknown>>({});
  const [selectedCapabilities, setSelectedCapabilities] = useState<Set<string>>(new Set());

  const deleteIntegrationMutation = useDeleteIntegration(organizationId, createdIntegration?.metadata?.id ?? "");

  const integrationDefinition = useMemo(
    () => availableIntegrations.find((integration) => integration.name === integrationName),
    [availableIntegrations, integrationName],
  );

  const integrationLabel =
    integrationDefinition?.label || getIntegrationTypeDisplayName(undefined, integrationName) || integrationName;
  const currentStep = getCurrentSetupStep(createdIntegration);
  const canRevertCurrentStep = canRevertSetupStep(createdIntegration);
  const integrationReady = createdIntegration?.status?.state === "ready";
  const showSetupStepBack = Boolean(createdIntegration) && (!integrationReady || canRevertCurrentStep);
  const integrationCapabilities = useMemo(() => {
    return [...(integrationDefinition?.capabilities || [])]
      .filter((capability) => Boolean(capability.name))
      .sort((left, right) => left.label!.localeCompare(right.label!));
  }, [integrationDefinition?.capabilities]);

  const capabilitySections = useMemo(
    () => buildIntegrationCapabilityGroupSections(integrationDefinition, integrationCapabilities),
    [integrationDefinition, integrationCapabilities],
  );

  const capabilityByName = useMemo(() => {
    const map = new Map<string, IntegrationsCapabilityDefinition>();
    for (const capability of integrationCapabilities) {
      if (capability.name) {
        map.set(capability.name, capability);
      }
    }
    return map;
  }, [integrationCapabilities]);

  const setupPageTitle = useMemo(() => {
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
  }, [createdIntegration, currentStep, integrationLabel]);

  usePageTitle(["Integrations", setupPageTitle]);

  const existingIntegrationNames = useMemo(() => {
    return new Set(
      connectedIntegrations
        .map((integration) => integration.metadata?.name?.trim())
        .filter((name): name is string => Boolean(name)),
    );
  }, [connectedIntegrations]);

  useEffect(() => {
    lastResumeDescribeKey.current = null;
    setCreatedIntegration(null);
    setStepInputs({});
    setInstanceName("");
    setSelectedCapabilities(new Set());
  }, [integrationName]);

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
  }, [integrationCapabilities, createdIntegration]);

  useEffect(() => {
    if (instanceName || !integrationName) {
      return;
    }

    setInstanceName(getNextIntegrationName(integrationName, existingIntegrationNames));
  }, [instanceName, integrationName, existingIntegrationNames]);

  useEffect(() => {
    applyResumeDescribeIfChanged(setupIntegrationId, resumeIntegrationDescribe, lastResumeDescribeKey, (describe) => {
      setCreatedIntegration(describe);
      setStepInputs({});
      setSelectedCapabilities(new Set());
      setInstanceName(describe.metadata?.name || describe.metadata?.integrationName || "");
    });
  }, [setupIntegrationId, resumeIntegrationDescribe]);

  useEffect(() => {
    const id = createdIntegration?.metadata?.id;
    if (!id || getCurrentSetupStep(createdIntegration)) {
      return;
    }

    if (setupIntegrationId && id === setupIntegrationId && (isResumeDescribePending || !resumeIntegrationDescribe)) {
      return;
    }

    navigate(`/${organizationId}/settings/integrations/${id}`, { replace: true });
  }, [
    createdIntegration,
    organizationId,
    navigate,
    setupIntegrationId,
    isResumeDescribePending,
    resumeIntegrationDescribe,
  ]);

  const handleCreateIntegration = useCallback(async () => {
    const trimmedName = instanceName.trim();
    if (!trimmedName) {
      showErrorToast("Integration name is required");
      return;
    }

    if (integrationCapabilities.length > 0 && selectedCapabilities.size === 0) {
      showErrorToast("Select at least one capability");
      return;
    }

    try {
      const response = await createMutation.mutateAsync({
        integrationName,
        name: trimmedName,
        capabilities: Array.from(selectedCapabilities),
      });
      const integration = response.data?.integration || null;
      setCreatedIntegration(integration);
      setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  }, [createMutation, instanceName, integrationCapabilities, integrationName, selectedCapabilities]);

  const toggleCapabilitySelection = useCallback(
    (capabilityName: string) => {
      if (createMutation.isPending) {
        return;
      }

      setSelectedCapabilities((current) => {
        const next = new Set(current);
        if (next.has(capabilityName)) {
          next.delete(capabilityName);
        } else {
          next.add(capabilityName);
        }
        return next;
      });
    },
    [createMutation],
  );

  const toggleCapabilityGroup = useCallback(
    (capabilityNames: string[]) => {
      if (createMutation.isPending || capabilityNames.length === 0) {
        return;
      }

      setSelectedCapabilities((previous) => {
        const state = getGroupToggleState(capabilityNames, previous);
        const next = new Set(previous);
        if (state === "all") {
          capabilityNames.forEach((name) => next.delete(name));
        } else {
          capabilityNames.forEach((name) => next.add(name));
        }
        return next;
      });
    },
    [createMutation],
  );

  const handleSubmitCurrentStep = async () => {
    const integrationId = createdIntegration?.metadata?.id;
    if (!integrationId || !currentStep) {
      return;
    }

    if (currentStep.type !== "DONE" && !currentStep.name) {
      return;
    }

    try {
      const response = await submitStepMutation.mutateAsync({
        integrationId,
        inputs: currentStep.type === "INPUTS" ? stepInputs : undefined,
      });
      const updatedIntegration = response.data?.integration || null;
      setCreatedIntegration(updatedIntegration);
      setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  };

  const handleRevertCurrentStep = async () => {
    const integrationId = createdIntegration?.metadata?.id;
    if (!integrationId || !currentStep) {
      return;
    }

    if (currentStep.type !== "DONE" && !currentStep.name) {
      return;
    }

    try {
      const response = await revertStepMutation.mutateAsync({
        integrationId,
      });
      const updatedIntegration = response.data?.integration || null;
      setCreatedIntegration(updatedIntegration);
      setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  };

  const handleDiscardIntegration = async () => {
    const integrationId = createdIntegration?.metadata?.id;
    if (!integrationId) {
      return;
    }
    if (!window.confirm("Discard this integration? It will be removed and this cannot be undone.")) {
      return;
    }
    try {
      await deleteIntegrationMutation.mutateAsync({
        integrationName: createdIntegration?.metadata?.integrationName ?? "",
      });
      setCreatedIntegration(null);
      navigate(integrationsHref);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  async function handleSetupStepBack() {
    if (!currentStep || !createdIntegration?.metadata?.id) {
      return;
    }

    if (canRevertCurrentStep) {
      await handleRevertCurrentStep();
      return;
    }

    if (
      !window.confirm("Remove this partially configured integration? You'll return to naming and capability selection.")
    ) {
      return;
    }

    try {
      await deleteIntegrationMutation.mutateAsync({
        integrationName: createdIntegration?.metadata?.integrationName ?? "",
      });
      setCreatedIntegration(null);
      setStepInputs({});
    } catch {
      showErrorToast("Failed to remove integration");
    }
  }

  if (isAvailableIntegrationsLoading) {
    return (
      <div className="flex justify-center items-center gap-2 py-16 text-gray-500 dark:text-gray-400">
        <Loader2 className="h-6 w-6 animate-spin" aria-hidden />
        <span className="text-sm">Loading integration metadata...</span>
      </div>
    );
  }

  const activeError = createMutation.error || submitStepMutation.error || revertStepMutation.error;

  const setupHeader = (
    <header className={cn("space-y-3", !createdIntegration && "mb-6 px-4 sm:px-6")}>
      <nav className="text-xs text-gray-500 dark:text-gray-400" aria-label="Setup navigation">
        <Link
          to={integrationsHref}
          className="inline-flex items-center gap-1.5 font-medium leading-none text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
        >
          <MoveLeft aria-hidden className="size-[1em] shrink-0 opacity-80" />
          Integrations
        </Link>
      </nav>

      <div className="flex w-full min-w-0 items-center gap-3">
        <IntegrationIcon
          integrationName={integrationName}
          iconSlug={integrationDefinition?.icon}
          className="h-6 w-6 shrink-0 text-gray-700 dark:text-gray-300"
        />
        <h4 className="min-w-0 truncate text-2xl font-medium text-gray-900 dark:text-gray-100">{setupPageTitle}</h4>
      </div>
    </header>
  );

  const setupCard = (
    <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
      {activeError ? (
        <Alert variant="destructive">
          <AlertTitle>Setup failed</AlertTitle>
          <AlertDescription>{getApiErrorMessage(activeError)}</AlertDescription>
        </Alert>
      ) : null}

      {!createdIntegration ? (
        <PreCreateIntegrationSetup
          instanceName={instanceName}
          onInstanceNameChange={setInstanceName}
          integrationName={integrationName}
          integrationCapabilities={integrationCapabilities}
          capabilitySections={capabilitySections}
          capabilityByName={capabilityByName}
          selectedCapabilities={selectedCapabilities}
          onToggleCapability={toggleCapabilitySelection}
          onToggleCapabilityGroup={toggleCapabilityGroup}
          isCreatePending={createMutation.isPending}
          onCreate={handleCreateIntegration}
        />
      ) : currentStep?.type === "INPUTS" ? (
        <InputsStep
          organizationId={organizationId}
          step={currentStep}
          values={stepInputs}
          onChange={(fieldName, value) => setStepInputs((currentValues) => ({ ...currentValues, [fieldName]: value }))}
          onSubmit={handleSubmitCurrentStep}
          onBack={showSetupStepBack ? () => void handleSetupStepBack() : undefined}
          isSubmitting={submitStepMutation.isPending}
          isReverting={revertStepMutation.isPending || deleteIntegrationMutation.isPending}
        />
      ) : currentStep?.type === "REDIRECT_PROMPT" ? (
        <RedirectPromptStep
          step={currentStep}
          onBack={showSetupStepBack ? () => void handleSetupStepBack() : undefined}
          onOpenRedirect={() => openRedirectPrompt(currentStep)}
          onSubmit={handleSubmitCurrentStep}
          isSubmitting={submitStepMutation.isPending}
          isReverting={revertStepMutation.isPending || deleteIntegrationMutation.isPending}
        />
      ) : currentStep?.type === "DONE" ? (
        <DoneStep
          step={currentStep}
          onFinish={() => void handleSubmitCurrentStep()}
          isSubmitting={submitStepMutation.isPending}
        />
      ) : null}
    </div>
  );

  return (
    <div className="pt-6">
      {createdIntegration ? (
        <div className="space-y-6 px-4 sm:px-6">
          {setupHeader}
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
            <div className="min-w-0 flex-1">{setupCard}</div>
            <StepHistory
              previousSteps={createdIntegration.status?.setupState?.previousSteps ?? []}
              currentStep={currentStep}
              onDiscard={
                createdIntegration.metadata?.id && !integrationReady ? () => void handleDiscardIntegration() : undefined
              }
              discardDisabled={
                deleteIntegrationMutation.isPending ||
                submitStepMutation.isPending ||
                revertStepMutation.isPending ||
                createMutation.isPending
              }
            />
          </div>
        </div>
      ) : (
        <>
          {setupHeader}
          {setupCard}
        </>
      )}
    </div>
  );
}
