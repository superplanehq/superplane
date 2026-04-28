import type {
  ConfigurationField,
  IntegrationsCapabilityDefinition,
  IntegrationSetupStepDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft, Check, CircleOff, MoveLeft, MoveRight } from "lucide-react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useDeleteIntegration,
  useNextIntegrationSetupStep,
  usePreviousIntegrationSetupStep,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { isIntegrationV2SetupEnabled } from "@/lib/integrationV2";
import { cn } from "@/lib/utils";
import { showErrorToast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CopyButton } from "@/ui/CopyButton";
import { IntegrationSetupDoneStep } from "./IntegrationSetupDoneStep";
import { IntegrationSetupInputsStep } from "./IntegrationSetupInputsStep";
import { IntegrationSetupRedirectPromptStep } from "./IntegrationSetupRedirectPromptStep";
import { IntegrationSetupStepHistory } from "./IntegrationSetupStepHistory";

interface IntegrationSetupPageProps {
  organizationId: string;
}

type IntegrationSetupRouteState = {
  integrationId?: string;
};

function isMissingValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return true;
  }

  if (typeof value === "string") {
    return value.trim() === "";
  }

  if (Array.isArray(value)) {
    return value.length === 0;
  }

  return false;
}

function getMissingRequiredFields(
  fields: Array<ConfigurationField> | undefined,
  values: Record<string, unknown>,
): Set<string> {
  const missing = new Set<string>();
  if (!fields) {
    return missing;
  }

  fields.forEach((field) => {
    if (!field.name || !field.required) {
      return;
    }

    if (isMissingValue(values[field.name])) {
      missing.add(field.name);
    }
  });

  return missing;
}

function openRedirectPrompt(step: IntegrationSetupStepDefinition | null) {
  const redirectPrompt = step?.redirectPrompt;
  if (!redirectPrompt?.url) {
    return;
  }

  if (redirectPrompt.method?.toUpperCase() === "POST" && redirectPrompt.formFields) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = redirectPrompt.url;
    form.target = "_blank";
    form.style.display = "none";

    Object.entries(redirectPrompt.formFields).forEach(([key, value]) => {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = key;
      input.value = String(value);
      form.appendChild(input);
    });

    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
    return;
  }

  window.open(redirectPrompt.url, "_blank");
}

function getNextIntegrationName(baseName: string, existingNames: Set<string>): string {
  const normalizedBaseName = baseName.trim() || "integration";
  if (!existingNames.has(normalizedBaseName)) {
    return normalizedBaseName;
  }

  let suffix = 2;
  let candidate = `${normalizedBaseName}-${suffix}`;
  while (existingNames.has(candidate)) {
    suffix += 1;
    candidate = `${normalizedBaseName}-${suffix}`;
  }

  return candidate;
}

function getCurrentSetupStep(integration: OrganizationsIntegration | null): IntegrationSetupStepDefinition | null {
  return integration?.status?.setupState?.currentStep ?? null;
}

function canRevertSetupStep(integration: OrganizationsIntegration | null): boolean {
  const previousSteps = integration?.status?.setupState?.previousSteps ?? [];
  return previousSteps.length > 0;
}

function getCapabilityDisplayName(capability: IntegrationsCapabilityDefinition): string {
  return capability.label || capability.name || "Unnamed capability";
}

function getCapabilitySelectionDotClass(selected: boolean) {
  return selected ? "bg-green-500" : "bg-gray-400 dark:bg-gray-500";
}

export function IntegrationSetupPage({ organizationId }: IntegrationSetupPageProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { integrationName: routeIntegrationName } = useParams<{ integrationName: string }>();
  const integrationName = routeIntegrationName || "";
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const routeState = location.state as IntegrationSetupRouteState | null;
  const setupIntegrationId = routeState?.integrationId;

  const { data: availableIntegrations = [], isLoading: isAvailableIntegrationsLoading } = useAvailableIntegrations();
  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId);

  const createMutation = useCreateIntegration(organizationId);
  const submitStepMutation = useNextIntegrationSetupStep(organizationId);
  const revertStepMutation = usePreviousIntegrationSetupStep(organizationId);

  const [instanceName, setInstanceName] = useState("");
  const [createdIntegration, setCreatedIntegration] = useState<OrganizationsIntegration | null>(null);
  const [stepInputs, setStepInputs] = useState<Record<string, unknown>>({});
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [introStep, setIntroStep] = useState<"name" | "capabilities">("name");
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
  const integrationCapabilities = useMemo(() => {
    return [...(integrationDefinition?.capabilities || [])]
      .filter((capability) => Boolean(capability.name))
      .sort((left, right) => getCapabilityDisplayName(left).localeCompare(getCapabilityDisplayName(right)));
  }, [integrationDefinition?.capabilities]);

  const setupPageTitle = useMemo(() => {
    if (!createdIntegration) {
      return introStep === "name" ? "Name your integration" : "Choose integration capabilities";
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
  }, [createdIntegration, currentStep, integrationLabel, introStep]);

  usePageTitle(["Integrations", setupPageTitle]);

  const existingIntegrationNames = useMemo(() => {
    return new Set(
      connectedIntegrations
        .map((integration) => integration.metadata?.name?.trim())
        .filter((name): name is string => Boolean(name)),
    );
  }, [connectedIntegrations]);

  useEffect(() => {
    setCreatedIntegration(null);
    setStepInputs({});
    setValidationErrors(new Set());
    setInstanceName("");
    setIntroStep("name");
    setSelectedCapabilities(new Set());
  }, [integrationName]);

  useEffect(() => {
    if (introStep !== "capabilities") {
      return;
    }

    setSelectedCapabilities((current) => {
      if (current.size > 0) {
        return current;
      }

      return new Set(integrationCapabilities.map((capability) => capability.name).filter(Boolean) as string[]);
    });
  }, [integrationCapabilities, introStep]);

  useEffect(() => {
    if (instanceName || !integrationName) {
      return;
    }

    setInstanceName(getNextIntegrationName(integrationName, existingIntegrationNames));
  }, [instanceName, integrationName, existingIntegrationNames]);

  useEffect(() => {
    if (!setupIntegrationId) {
      return;
    }

    if (createdIntegration?.metadata?.id === setupIntegrationId) {
      return;
    }

    const integration = connectedIntegrations.find(
      (connectedIntegration) => connectedIntegration.metadata?.id === setupIntegrationId,
    );
    if (!integration) {
      return;
    }

    setCreatedIntegration(integration);
    setStepInputs({});
    setValidationErrors(new Set());
    setIntroStep("name");
    setSelectedCapabilities(new Set());
    setInstanceName(integration.metadata?.name || integration.metadata?.integrationName || "");
  }, [connectedIntegrations, createdIntegration?.metadata?.id, setupIntegrationId]);

  useEffect(() => {
    const id = createdIntegration?.metadata?.id;
    if (!id || getCurrentSetupStep(createdIntegration)) {
      return;
    }

    navigate(`/${organizationId}/settings/integrations/${id}`, { replace: true });
  }, [createdIntegration, organizationId, navigate]);

  async function handleCreateIntegration() {
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
      setValidationErrors(new Set());
    } catch {
      // Error is shown by inline alert.
    }
  }

  async function handleNameStepContinue() {
    const trimmedName = instanceName.trim();
    if (!trimmedName) {
      showErrorToast("Integration name is required");
      return;
    }

    if (integrationCapabilities.length === 0) {
      await handleCreateIntegration();
      return;
    }

    setIntroStep("capabilities");
  }

  function toggleCapabilitySelection(capabilityName: string) {
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
  }

  const handleSubmitCurrentStep = async () => {
    const integrationId = createdIntegration?.metadata?.id;
    if (!integrationId || !currentStep) {
      return;
    }

    if (currentStep.type !== "DONE" && !currentStep.name) {
      return;
    }

    if (currentStep.type === "INPUTS") {
      const missingRequiredFields = getMissingRequiredFields(currentStep.inputs, stepInputs);
      if (missingRequiredFields.size > 0) {
        setValidationErrors(missingRequiredFields);
        return;
      }
    }

    try {
      const response = await submitStepMutation.mutateAsync({
        integrationId,
        inputs: currentStep.type === "INPUTS" ? stepInputs : undefined,
      });
      const updatedIntegration = response.data?.integration || null;
      setCreatedIntegration(updatedIntegration);
      setStepInputs({});
      setValidationErrors(new Set());
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
      setValidationErrors(new Set());
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
      await deleteIntegrationMutation.mutateAsync();
      setCreatedIntegration(null);
      navigate(integrationsHref);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  if (!isIntegrationV2SetupEnabled(integrationName)) {
    return (
      <div className="pt-6 space-y-4">
        <div className="flex items-center gap-4">
          <Link
            to={integrationsHref}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Setup</h4>
        </div>
        <Alert>
          <AlertTitle>Unsupported setup flow</AlertTitle>
          <AlertDescription>
            This integration is not enabled for the new setup flow. Use the standard connect flow from the integrations
            list.
          </AlertDescription>
        </Alert>
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

      {!createdIntegration && introStep === "name" ? (
        <div className="space-y-4">
          <div>
            <Label className="mb-2">Integration Name</Label>
            <Input
              value={instanceName}
              onChange={(event) => setInstanceName(event.target.value)}
              placeholder={`${integrationName}-integration`}
            />
          </div>
          <div className="flex w-fit max-w-full items-center gap-4 pt-2">
            <Button
              onClick={() => void handleNameStepContinue()}
              disabled={createMutation.isPending || !instanceName.trim()}
              className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
            >
              {createMutation.isPending ? "Creating..." : "Next"}
              <MoveRight
                aria-hidden
                className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
              />
            </Button>
          </div>
        </div>
      ) : !createdIntegration && introStep === "capabilities" ? (
        <div className="space-y-5">
          <div className="overflow-x-auto rounded-md border border-gray-300 dark:border-gray-700">
            <table className="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
              <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                {integrationCapabilities.map((capability) => {
                  const capabilityName = capability.name || "";
                  const checked = selectedCapabilities.has(capabilityName);
                  const statusDotClass = getCapabilitySelectionDotClass(checked);

                  return (
                    <tr
                      key={capabilityName}
                      className={cn(
                        "transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background dark:focus-visible:ring-offset-gray-900",
                        createMutation.isPending
                          ? "cursor-not-allowed opacity-70"
                          : "cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/60",
                      )}
                      onClick={() => toggleCapabilitySelection(capabilityName)}
                      onKeyDown={(event) => {
                        if (createMutation.isPending) {
                          return;
                        }
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          toggleCapabilitySelection(capabilityName);
                        }
                      }}
                      tabIndex={createMutation.isPending ? -1 : 0}
                      aria-selected={checked}
                      aria-label={`${checked ? "Selected" : "Not selected"}: ${capabilityName}. Press Enter or Space to toggle.`}
                    >
                      <td className="px-4 py-3 align-middle">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)} aria-hidden />
                          <span className="font-mono text-sm text-gray-800 dark:text-gray-100">{capabilityName}</span>
                          <CopyButton text={capabilityName} />
                        </div>
                      </td>
                      <td className="px-4 py-3 align-middle">
                        {capability.description ? (
                          <div className="text-sm text-gray-600 dark:text-gray-400">{capability.description}</div>
                        ) : null}
                      </td>
                      <td className="px-4 py-3 align-middle">
                        <div className="flex justify-end">
                          {checked ? (
                            <Check className="size-3 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                          ) : (
                            <CircleOff className="size-3 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <div className="flex w-fit max-w-full items-center gap-4 pt-2">
            <Button
              type="button"
              variant="link"
              onClick={() => setIntroStep("name")}
              disabled={createMutation.isPending}
              className="group h-auto shrink-0 gap-1.5 px-0 py-1 font-normal hover:!no-underline"
            >
              <ArrowLeft
                aria-hidden
                className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:-translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
              />
              Previous
            </Button>
            <Button
              type="button"
              onClick={() => void handleCreateIntegration()}
              disabled={createMutation.isPending || selectedCapabilities.size === 0}
              className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
            >
              {createMutation.isPending ? "Saving..." : "Next"}
              <MoveRight
                aria-hidden
                className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
              />
            </Button>
          </div>
        </div>
      ) : currentStep?.type === "INPUTS" ? (
        <IntegrationSetupInputsStep
          organizationId={organizationId}
          step={currentStep}
          values={stepInputs}
          validationErrors={validationErrors}
          onChange={(fieldName, value) => {
            setValidationErrors((currentValidationErrors) => {
              if (!currentValidationErrors.has(fieldName)) {
                return currentValidationErrors;
              }

              const nextValidationErrors = new Set(currentValidationErrors);
              nextValidationErrors.delete(fieldName);
              return nextValidationErrors;
            });
            setStepInputs((currentValues) => ({ ...currentValues, [fieldName]: value }));
          }}
          onSubmit={handleSubmitCurrentStep}
          onBack={canRevertCurrentStep ? handleRevertCurrentStep : undefined}
          isSubmitting={submitStepMutation.isPending}
          isReverting={revertStepMutation.isPending}
        />
      ) : currentStep?.type === "REDIRECT_PROMPT" ? (
        <IntegrationSetupRedirectPromptStep
          step={currentStep}
          onBack={canRevertCurrentStep ? handleRevertCurrentStep : undefined}
          onOpenRedirect={() => openRedirectPrompt(currentStep)}
          onSubmit={handleSubmitCurrentStep}
          isSubmitting={submitStepMutation.isPending}
          isReverting={revertStepMutation.isPending}
        />
      ) : currentStep?.type === "DONE" ? (
        <IntegrationSetupDoneStep
          step={currentStep}
          onFinish={() => void handleSubmitCurrentStep()}
          isSubmitting={submitStepMutation.isPending}
        />
      ) : null}

      {isAvailableIntegrationsLoading ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading integration metadata...</p>
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
            <IntegrationSetupStepHistory
              previousSteps={createdIntegration.status?.setupState?.previousSteps ?? []}
              currentStep={currentStep}
              onDiscard={createdIntegration.metadata?.id ? () => void handleDiscardIntegration() : undefined}
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
