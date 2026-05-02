import type {
  ConfigurationField,
  IntegrationsCapabilityDefinition,
  IntegrationSetupStepDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, CircleOff, Info, Loader2, Minus, MoveLeft, MoveRight } from "lucide-react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useEffect, useMemo, useRef, useState } from "react";
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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CopyButton } from "@/ui/CopyButton";
import { DoneStep } from "./DoneStep";
import { InputsStep } from "./InputsStep";
import { RedirectPromptStep } from "./RedirectPromptStep";
import { StepHistory } from "./StepHistory";

interface IntegrationSetupProps {
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

function getCapabilitySelectionDotClass(selected: boolean) {
  return selected ? "bg-green-500" : "bg-gray-400 dark:bg-gray-500";
}

function getGroupToggleState(capabilityNames: string[], selected: ReadonlySet<string>): "all" | "some" | "none" {
  if (capabilityNames.length === 0) {
    return "none";
  }

  let count = 0;
  for (const name of capabilityNames) {
    if (selected.has(name)) {
      count++;
    }
  }

  if (count === 0) {
    return "none";
  }
  if (count === capabilityNames.length) {
    return "all";
  }
  return "some";
}

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
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
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
      .sort((left, right) =>
        left.label!.localeCompare(right.label!),
      );
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
    setValidationErrors(new Set());
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
    if (!setupIntegrationId) {
      return;
    }

    if (!resumeIntegrationDescribe || resumeIntegrationDescribe.metadata?.id !== setupIntegrationId) {
      return;
    }

    const setupStepName = resumeIntegrationDescribe.status?.setupState?.currentStep?.name ?? "";
    const resumeKey = `${resumeIntegrationDescribe.metadata?.updatedAt ?? ""}:${setupStepName}`;
    if (lastResumeDescribeKey.current === resumeKey) {
      return;
    }
    lastResumeDescribeKey.current = resumeKey;

    setCreatedIntegration(resumeIntegrationDescribe);
    setStepInputs({});
    setValidationErrors(new Set());
    setSelectedCapabilities(new Set());
    setInstanceName(
      resumeIntegrationDescribe.metadata?.name || resumeIntegrationDescribe.metadata?.integrationName || "",
    );
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

  function toggleCapabilityGroup(capabilityNames: string[]) {
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
      await deleteIntegrationMutation.mutateAsync({ integrationName: createdIntegration?.metadata?.integrationName ?? "" });
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
      await deleteIntegrationMutation.mutateAsync({ integrationName: createdIntegration?.metadata?.integrationName ?? "" });
      setCreatedIntegration(null);
      setStepInputs({});
      setValidationErrors(new Set());
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
        <div className="space-y-6">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
              <Label htmlFor="integration-instance-name" className="mb-0 shrink-0">
                Name
              </Label>
              <Input
                id="integration-instance-name"
                value={instanceName}
                onChange={(event) => setInstanceName(event.target.value)}
                placeholder={`${integrationName}-integration`}
                autoComplete="off"
                className="h-9 w-72 max-w-full"
              />
            </div>
            <div className="flex gap-3 rounded-md border border-gray-300 bg-gray-50 p-3 text-sm leading-relaxed text-gray-600 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-400">
              <Info className="mt-0.5 size-4 shrink-0 text-gray-500 dark:text-gray-500" aria-hidden />
              <p className="min-w-0">
                You can connect the same integration type more than once—for different environments, namespaces, or
                organizations. Use a name that identifies this connection.
              </p>
            </div>
          </div>

          {integrationCapabilities.length > 0 ? (
            <div className="space-y-3">
              <hr className="border-gray-200 dark:border-gray-800" />
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Choose which capabilities to enable for this integration. You need at least one. Use a group row to
                select or clear every capability in that group at once.
              </p>
              <div className="space-y-4">
                {capabilitySections.map((section) => {
                  const groupState = section.label
                    ? getGroupToggleState(section.names, selectedCapabilities)
                    : undefined;
                  const selectedInSection = section.names.filter((name) => selectedCapabilities.has(name)).length;
                  const groupIcon =
                    groupState === undefined ? null : groupState === "all" ? (
                      <Check className="size-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                    ) : groupState === "some" ? (
                      <Minus className="size-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
                    ) : (
                      <CircleOff className="size-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
                    );

                  return (
                    <div
                      key={section.key}
                      className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-700"
                      role={section.label ? "group" : undefined}
                      aria-label={section.label ? `${section.label} capabilities` : undefined}
                    >
                      {section.label ? (
                        <button
                          type="button"
                          disabled={createMutation.isPending}
                          aria-label={
                            groupState === "all"
                              ? `Remove all selections from ${section.label}`
                              : `Select all capabilities in ${section.label}`
                          }
                          className={cn(
                            "flex w-full cursor-pointer flex-wrap items-center justify-between gap-3 border-b border-gray-200 bg-gray-50 px-4 py-3 text-left transition-colors hover:bg-gray-100 dark:border-gray-800 dark:bg-gray-800/50 dark:hover:bg-gray-800",
                            createMutation.isPending &&
                              "!cursor-not-allowed opacity-70 hover:bg-gray-50 dark:hover:bg-gray-800/50",
                          )}
                          onClick={() => toggleCapabilityGroup(section.names)}
                          onKeyDown={(event) => {
                            if (createMutation.isPending) {
                              return;
                            }
                            if (event.key === "Enter" || event.key === " ") {
                              event.preventDefault();
                              toggleCapabilityGroup(section.names);
                            }
                          }}
                        >
                          <div className="min-w-0">
                            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                              {section.label}
                            </span>
                            <span className="ml-2 text-xs tabular-nums text-gray-500 dark:text-gray-400">
                              {selectedInSection}/{section.names.length}
                            </span>
                          </div>
                          <div className="flex shrink-0 items-center">{groupIcon}</div>
                        </button>
                      ) : null}
                      <div className={cn(section.label && "-mt-px", "overflow-x-auto")}>
                        <table className="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
                          <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                            {section.names.map((capabilityName) => {
                              const capability = capabilityByName.get(capabilityName);
                              if (!capability) {
                                return null;
                              }

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
                                      <span
                                        className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)}
                                        aria-hidden
                                      />
                                      <span className="font-mono text-sm text-gray-800 dark:text-gray-100">
                                        {capabilityName}
                                      </span>
                                      <CopyButton text={capabilityName} />
                                    </div>
                                  </td>
                                  <td className="px-4 py-3 align-middle">
                                    {capability.description ? (
                                      <div className="text-sm text-gray-600 dark:text-gray-400">
                                        {capability.description}
                                      </div>
                                    ) : null}
                                  </td>
                                  <td className="px-4 py-3 align-middle">
                                    <div className="flex justify-end">
                                      {checked ? (
                                        <Check
                                          className="size-3 shrink-0 text-green-600 dark:text-green-400"
                                          aria-hidden
                                        />
                                      ) : (
                                        <CircleOff
                                          className="size-3 shrink-0 text-gray-400 dark:text-gray-500"
                                          aria-hidden
                                        />
                                      )}
                                    </div>
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}

          <div className="flex w-fit max-w-full items-center gap-4 pt-2">
            <Button
              type="button"
              onClick={() => void handleCreateIntegration()}
              disabled={
                createMutation.isPending ||
                !instanceName.trim() ||
                (integrationCapabilities.length > 0 && selectedCapabilities.size === 0)
              }
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
      ) : currentStep?.type === "INPUTS" ? (
        <InputsStep
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