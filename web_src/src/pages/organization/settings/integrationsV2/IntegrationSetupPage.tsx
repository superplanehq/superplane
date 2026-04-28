import type {
  ConfigurationField,
  IntegrationsCapabilityDefinition,
  IntegrationSetupStepDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft } from "lucide-react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useNextIntegrationSetupStep,
  usePreviousIntegrationSetupStep,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { isIntegrationV2SetupEnabled } from "@/lib/integrationV2";
import { showErrorToast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/ui/checkbox";
import { IntegrationSetupInputsStep } from "./IntegrationSetupInputsStep";
import { IntegrationSetupRedirectPromptStep } from "./IntegrationSetupRedirectPromptStep";

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

function getCapabilityTypeLabel(capability: IntegrationsCapabilityDefinition): string {
  if (capability.type === "TYPE_ACTION") return "Action";
  if (capability.type === "TYPE_TRIGGER") return "Trigger";
  return "Unknown";
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

  usePageTitle(["Integrations", integrationLabel, "Setup"]);

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

  const handleSubmitCurrentStep = async () => {
    const integrationId = createdIntegration?.metadata?.id;
    if (!integrationId || !currentStep?.name) {
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
    if (!integrationId || !currentStep?.name) {
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

  return (
    <div className="pt-6">
      <div className="flex items-center gap-4 mb-6">
        <Link
          to={integrationsHref}
          className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          aria-label="Back to integrations"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <IntegrationIcon
          integrationName={integrationName}
          iconSlug={integrationDefinition?.icon}
          className="w-6 h-6 text-gray-700 dark:text-gray-300"
        />
        <div className="min-w-0">
          <h4 className="text-2xl font-medium text-gray-900 dark:text-gray-100">Setup {integrationLabel}</h4>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Complete the setup steps to connect this integration.
          </p>
        </div>
      </div>

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
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Step: Name your integration</h2>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                Choose a unique name for this connection and continue.
              </p>
            </div>
            <div>
              <Label className="mb-2">Integration Name</Label>
              <Input
                value={instanceName}
                onChange={(event) => setInstanceName(event.target.value)}
                placeholder={`${integrationName}-integration`}
              />
            </div>
            <div className="flex items-center gap-3">
              <Button
                onClick={() => void handleNameStepContinue()}
                disabled={createMutation.isPending || !instanceName.trim()}
              >
                {createMutation.isPending ? "Creating..." : "Next"}
              </Button>
            </div>
          </div>
        ) : !createdIntegration && introStep === "capabilities" ? (
          <div className="space-y-5">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Step: Choose integration capabilities
              </h2>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                Select the capabilities this integration should provide.
              </p>
            </div>
            <div className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-700">
              <div className="divide-y divide-gray-200 dark:divide-gray-800">
                {integrationCapabilities.map((capability) => {
                  const capabilityName = capability.name || "";
                  const capabilityId = `capability-${capabilityName}`;
                  const checked = selectedCapabilities.has(capabilityName);

                  return (
                    <label
                      key={capabilityName}
                      htmlFor={capabilityId}
                      className="flex cursor-pointer items-start gap-3 bg-white p-4 transition-colors hover:bg-gray-50 dark:bg-gray-900 dark:hover:bg-gray-800/70"
                    >
                      <Checkbox
                        id={capabilityId}
                        checked={checked}
                        onCheckedChange={(nextChecked) => {
                          setSelectedCapabilities((current) => {
                            const next = new Set(current);
                            if (nextChecked === true) {
                              next.add(capabilityName);
                              return next;
                            }

                            next.delete(capabilityName);
                            return next;
                          });
                        }}
                        className="mt-0.5"
                      />
                      <span className="flex min-w-0 flex-1 flex-col gap-1">
                        <span className="flex flex-wrap items-center gap-2">
                          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {getCapabilityDisplayName(capability)}
                          </span>
                          <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                            {getCapabilityTypeLabel(capability)}
                          </span>
                        </span>
                        {capability.description && (
                          <span className="text-sm text-gray-600 dark:text-gray-400">{capability.description}</span>
                        )}
                      </span>
                    </label>
                  );
                })}
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIntroStep("name")}
                disabled={createMutation.isPending}
              >
                Previous
              </Button>
              <Button
                type="button"
                onClick={() => void handleCreateIntegration()}
                disabled={createMutation.isPending || selectedCapabilities.size === 0}
              >
                {createMutation.isPending ? "Saving..." : "Next"}
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
        ) : (
          <div className="space-y-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Setup complete</h2>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                This integration is ready to use. You can return to the integrations list.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <Button onClick={() => navigate(integrationsHref)}>Back to integrations</Button>
            </div>
          </div>
        )}

        {isAvailableIntegrationsLoading ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading integration metadata...</p>
        ) : null}
      </div>
    </div>
  );
}
