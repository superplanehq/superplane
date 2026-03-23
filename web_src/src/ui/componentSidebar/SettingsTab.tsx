import {
  AuthorizationDomainType,
  ComponentsIntegrationRef,
  ConfigurationField,
  IntegrationsIntegrationDefinition,
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";
import { useCallback, useEffect, useMemo, useState, ReactNode, useRef } from "react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import {
  filterVisibleConfiguration,
  isFieldRequired,
  parseDefaultValues,
  validateFieldForSubmission,
} from "@/utils/components";
import { useRealtimeValidation } from "@/hooks/useRealtimeValidation";
import { SimpleTooltip } from "./SimpleTooltip";
import type { IntegrationCreatePayload } from "@/ui/IntegrationCreateDialog";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { showErrorToast } from "@/utils/toast";
import { getUsageLimitToastMessage } from "@/utils/usageLimits";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { integrationKeys } from "@/hooks/useIntegrations";
import { useQueryClient } from "@tanstack/react-query";

interface SettingsTabProps {
  mode: "create" | "edit";
  nodeId?: string;
  nodeName: string;
  nodeLabel?: string;
  configuration: Record<string, unknown>;
  configurationFields: ConfigurationField[];
  onSave: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  onCancel?: () => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  customField?: (configuration: Record<string, unknown>) => ReactNode;
  integrationName?: string;
  integrationRef?: ComponentsIntegrationRef;
  integrations?: OrganizationsIntegration[];
  integrationDefinition?: { name?: string; label?: string; icon?: string };
  integrationDefinitionFull?: IntegrationsIntegrationDefinition;
  autocompleteExampleObj?: Record<string, unknown> | null;
  onOpenCreateIntegrationDialog?: (opts?: {
    defaultName?: string;
    browserAction?: OrganizationsBrowserAction;
    createdIntegrationId?: string;
    webhookSetup?: { id: string; webhookUrl: string; config: Record<string, unknown> };
  }) => void;
  onOpenConfigureIntegrationDialog?: (integrationId: string) => void;
  onCreateIntegration?: (payload: IntegrationCreatePayload) => Promise<OrganizationsCreateIntegrationResponse>;
  onIntegrationCreated?: (integrationId: string) => void;
  organizationId?: string;
  readOnly?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
}

export function SettingsTab({
  nodeId: _nodeId,
  nodeName,
  nodeLabel: _nodeLabel,
  configuration,
  configurationFields,
  onSave,
  onCancel: _onCancel,
  domainId,
  domainType,
  customField,
  integrationName,
  integrationRef,
  integrations = [],
  integrationDefinition,
  integrationDefinitionFull,
  autocompleteExampleObj,
  onOpenCreateIntegrationDialog,
  onOpenConfigureIntegrationDialog,
  onCreateIntegration,
  onIntegrationCreated,
  organizationId,
  readOnly = false,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
}: SettingsTabProps) {
  const CONNECT_ANOTHER_INSTANCE_VALUE = "__connect_another_instance__";
  const isReadOnly = readOnly ?? false;
  const allowIntegrations = canReadIntegrations ?? true;
  const allowCreateIntegrations = canCreateIntegrations ?? true;
  const allowUpdateIntegrations = canUpdateIntegrations ?? true;
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);
  const [selectedIntegration, setSelectedIntegration] = useState<ComponentsIntegrationRef | undefined>(integrationRef);
  const [isSaving, setIsSaving] = useState(false);
  const savingRef = useRef(false);
  // Use autocompleteExampleObj directly - current node is already filtered out
  const resolvedAutocompleteExampleObj = autocompleteExampleObj;

  const queryClient = useQueryClient();

  // Inline integration creation state
  const [inlineIntegrationName, setInlineIntegrationName] = useState(integrationDefinitionFull?.name ?? "");
  const [inlineIntegrationConfig, setInlineIntegrationConfig] = useState<Record<string, unknown>>({});
  const [isInlineCreating, setIsInlineCreating] = useState(false);
  const [inlineBrowserAction, setInlineBrowserAction] = useState<OrganizationsBrowserAction | undefined>(undefined);
  const [inlineCreatedIntegrationId, setInlineCreatedIntegrationId] = useState<string | undefined>(undefined);
  const [inlineBrowserActionCompleted, setInlineBrowserActionCompleted] = useState(false);
  const [inlineWebhookSetup, setInlineWebhookSetup] = useState<
    { id: string; webhookUrl: string; config: Record<string, unknown> } | undefined
  >(undefined);
  const [isInlineWebhookCompleting, setIsInlineWebhookCompleting] = useState(false);

  const inlineConfigFields = useMemo(() => {
    return integrationDefinitionFull?.configuration ?? [];
  }, [integrationDefinitionFull?.configuration]);

  useEffect(() => {
    setInlineIntegrationName(integrationDefinitionFull?.name ?? "");
    setInlineIntegrationConfig({});
    setInlineBrowserAction(undefined);
    setInlineCreatedIntegrationId(undefined);
    setInlineBrowserActionCompleted(false);
    setInlineWebhookSetup(undefined);
  }, [integrationDefinitionFull?.name]);

  const handleInlineCreate = useCallback(async () => {
    if (!onCreateIntegration || !integrationDefinitionFull?.name) return;
    const trimmedName = inlineIntegrationName.trim();
    if (!trimmedName) {
      showErrorToast("Integration name is required");
      return;
    }

    setIsInlineCreating(true);
    try {
      const result = await onCreateIntegration({
        integrationName: integrationDefinitionFull.name,
        name: trimmedName,
        configuration: inlineIntegrationConfig,
      });

      const integration = result.integration;
      const browserAction = integration?.status?.browserAction;
      const webhookUrl =
        integration?.status?.metadata &&
        typeof integration.status.metadata === "object" &&
        "webhookUrl" in integration.status.metadata
          ? (integration.status.metadata as { webhookUrl?: string }).webhookUrl
          : undefined;

      if (browserAction && integration?.metadata?.id) {
        setInlineBrowserAction(browserAction);
        setInlineCreatedIntegrationId(integration.metadata.id);
        return;
      }

      if (integration?.metadata?.id && webhookUrl) {
        setInlineWebhookSetup({ id: integration.metadata.id, webhookUrl, config: { ...inlineIntegrationConfig } });
        setInlineCreatedIntegrationId(integration.metadata.id);
        return;
      }

      if (integration?.metadata?.id) {
        onIntegrationCreated?.(integration.metadata.id);
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create integration"));
    } finally {
      setIsInlineCreating(false);
    }
  }, [
    onCreateIntegration,
    integrationDefinitionFull?.name,
    inlineIntegrationName,
    inlineIntegrationConfig,
    onIntegrationCreated,
  ]);

  const handleInlineBrowserActionContinue = useCallback(async () => {
    if (!inlineBrowserAction) return;
    const { url, method, formFields } = inlineBrowserAction;
    if (method?.toUpperCase() === "POST" && formFields) {
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";
      Object.entries(formFields).forEach(([key, value]) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = String(value);
        form.appendChild(input);
      });
      document.body.appendChild(form);
      form.submit();
      document.body.removeChild(form);
    } else if (url) {
      window.open(url, "_blank");
    }

    if (inlineCreatedIntegrationId) {
      const orgId = organizationId ?? domainId ?? "";
      try {
        await organizationsUpdateIntegration(
          withOrganizationHeader({
            path: { id: orgId, integrationId: inlineCreatedIntegrationId },
            body: { configuration: { ...inlineIntegrationConfig, installed: "true" } },
          }),
        );
        await queryClient.invalidateQueries({
          queryKey: integrationKeys.connected(orgId),
        });
      } catch {
        // Resync is best-effort
      }
      setInlineBrowserActionCompleted(true);
    }
  }, [inlineBrowserAction, inlineCreatedIntegrationId, inlineIntegrationConfig, organizationId, domainId, queryClient]);

  const handleInlineWebhookComplete = useCallback(async () => {
    if (!inlineWebhookSetup) return;
    const orgId = organizationId ?? domainId ?? "";
    setIsInlineWebhookCompleting(true);
    try {
      await organizationsUpdateIntegration(
        withOrganizationHeader({
          path: { id: orgId, integrationId: inlineWebhookSetup.id },
          body: { configuration: { ...inlineWebhookSetup.config, ...inlineIntegrationConfig } },
        }),
      );
      await queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(orgId),
      });
      onIntegrationCreated?.(inlineWebhookSetup.id);
    } catch {
      showErrorToast("Failed to complete setup");
    } finally {
      setIsInlineWebhookCompleting(false);
    }
  }, [inlineWebhookSetup, inlineIntegrationConfig, organizationId, domainId, queryClient, onIntegrationCreated]);

  const defaultValues = useMemo(() => {
    return parseDefaultValues(configurationFields);
  }, [configurationFields]);

  const defaultValuesWithoutToggles = useMemo(() => {
    const filtered = { ...defaultValues };
    configurationFields.forEach((field) => {
      if (field.name && field.togglable) {
        delete filtered[field.name];
      }
    });
    return filtered;
  }, [configurationFields, defaultValues]);

  // All installations of this integration type (ready, error, pending)
  const integrationsOfType = useMemo(() => {
    if (!integrationName) return [];
    return integrations.filter((i) => i.spec?.integrationName === integrationName);
  }, [integrations, integrationName]);
  const readyIntegrationsOfType = useMemo(() => {
    return integrationsOfType.filter((i) => i.status?.state === "ready");
  }, [integrationsOfType]);
  const pendingIntegration = useMemo(() => {
    if (readyIntegrationsOfType.length > 0) return undefined;
    return integrationsOfType.find((i) => i.status?.state === "pending" || i.status?.state === "error");
  }, [integrationsOfType, readyIntegrationsOfType]);

  // Seed inline state from an existing pending/errored integration so the user
  // sees the configure form (browser action, webhook, etc.) instead of a blank creation form.
  useEffect(() => {
    if (!pendingIntegration) return;
    const id = pendingIntegration.metadata?.id;
    if (!id) return;
    // Only seed once per integration id to avoid overwriting user edits
    if (inlineCreatedIntegrationId === id) return;

    setInlineCreatedIntegrationId(id);
    setInlineIntegrationName(pendingIntegration.metadata?.name ?? integrationDefinitionFull?.name ?? "");
    if (pendingIntegration.spec?.configuration) {
      setInlineIntegrationConfig({ ...pendingIntegration.spec.configuration });
    }

    const browserAction = pendingIntegration.status?.browserAction;
    if (browserAction) {
      setInlineBrowserAction(browserAction);
      setInlineBrowserActionCompleted(false);
      return;
    }

    const meta = pendingIntegration.status?.metadata;
    const webhookUrl =
      meta && typeof meta === "object" && "webhookUrl" in meta
        ? (meta as { webhookUrl?: string }).webhookUrl
        : undefined;
    if (webhookUrl) {
      setInlineWebhookSetup({
        id,
        webhookUrl,
        config: { ...(pendingIntegration.spec?.configuration ?? {}) },
      });
    }
  }, [pendingIntegration, inlineCreatedIntegrationId, integrationDefinitionFull?.name]);

  const selectedIntegrationFull = useMemo(() => {
    const id = selectedIntegration?.id ?? integrationRef?.id;
    if (!id) return undefined;
    return integrations.find((i) => i.metadata?.id === id);
  }, [integrations, selectedIntegration?.id, integrationRef?.id]);
  const {
    validationErrors: realtimeValidationErrors,
    validateNow,
    clearErrors: _clearRealtimeErrors,
    hasFieldError: hasRealtimeFieldError,
  } = useRealtimeValidation(
    configurationFields,
    { ...nodeConfiguration, nodeName: currentNodeName },
    {
      debounceMs: 200,
      validateOnMount: false,
    },
  );

  // Helper to check if node name has real-time validation error
  const hasNodeNameError = useMemo(() => {
    return hasRealtimeFieldError("nodeName") || currentNodeName.trim() === "";
  }, [hasRealtimeFieldError, currentNodeName]);

  const isFieldEmpty = (value: unknown): boolean => {
    if (value === null || value === undefined) return true;
    if (typeof value === "string") return value.trim() === "";
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === "object") return Object.keys(value).length === 0;
    return false;
  };

  // Recursively validate nested fields in objects and lists
  const validateNestedFields = useCallback(
    (fields: ConfigurationField[], values: Record<string, unknown>, parentPath: string = ""): Set<string> => {
      const errors = new Set<string>();

      fields.forEach((field) => {
        if (!field.name) return;

        const fieldPath = parentPath ? `${parentPath}.${field.name}` : field.name;
        const value = values[field.name];

        // Check if field is required (either always or conditionally)
        const fieldIsRequired = field.required || isFieldRequired(field, values);
        if (fieldIsRequired && isFieldEmpty(value)) {
          errors.add(fieldPath);
        }

        // Check validation rules (cross-field validation)
        if (value !== undefined && value !== null && value !== "") {
          const validationErrors = validateFieldForSubmission(field, value);

          if (validationErrors.length > 0) {
            // Add validation rule errors to the error set
            errors.add(fieldPath);
          }
        }

        // Handle nested validation for different field types
        if (field.type === "list" && Array.isArray(value) && field.typeOptions?.list?.itemDefinition) {
          const itemSchema = field.typeOptions.list.itemDefinition.schema;
          if (itemSchema) {
            value.forEach((item, index) => {
              if (typeof item === "object" && item !== null) {
                const nestedErrors = validateNestedFields(
                  itemSchema,
                  item as Record<string, unknown>,
                  `${fieldPath}[${index}]`,
                );
                nestedErrors.forEach((error) => errors.add(error));
              }
            });
          }
        } else if (
          field.type === "object" &&
          typeof value === "object" &&
          value !== null &&
          field.typeOptions?.object?.schema
        ) {
          const nestedErrors = validateNestedFields(
            field.typeOptions.object.schema,
            value as Record<string, unknown>,
            fieldPath,
          );
          nestedErrors.forEach((error) => errors.add(error));
        }
      });

      return errors;
    },
    [],
  );

  // Function to filter out invisible fields
  const filterVisibleFields = useCallback(
    (config: Record<string, unknown>) => {
      return filterVisibleConfiguration(config, configurationFields);
    },
    [configurationFields],
  );

  // Sync state when props change
  useEffect(() => {
    let newConfig;
    if (Object.values(configuration).length === 0 || !configuration) {
      newConfig = defaultValuesWithoutToggles;
    } else {
      newConfig = { ...defaultValuesWithoutToggles, ...configuration };
    }

    setNodeConfiguration(filterVisibleFields(newConfig));
    setCurrentNodeName(nodeName);
    setSelectedIntegration(integrationRef);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, defaultValuesWithoutToggles, filterVisibleFields, integrationRef]);

  // Auto-select the first ready installation if none is selected or selection is invalid
  useEffect(() => {
    if (readyIntegrationsOfType.length === 0) {
      if (selectedIntegration) {
        setSelectedIntegration(undefined);
      }
      return;
    }

    const selectedId = selectedIntegration?.id;
    const hasSelected = selectedId
      ? readyIntegrationsOfType.some((integration) => integration.metadata?.id === selectedId)
      : false;
    if (hasSelected) {
      return;
    }

    const firstIntegration = readyIntegrationsOfType[0];
    setSelectedIntegration({
      id: firstIntegration.metadata?.id,
      name: firstIntegration.metadata?.name,
    });
  }, [readyIntegrationsOfType, selectedIntegration]);

  const isIntegrationReady =
    !integrationName || !allowIntegrations || selectedIntegrationFull?.status?.state === "ready";
  const shouldShowConfiguration = (!integrationName || !!selectedIntegration?.id) && isIntegrationReady;

  const handleSave = async () => {
    if (isReadOnly || savingRef.current) {
      return;
    }
    validateNow();
    const result = onSave(nodeConfiguration, currentNodeName, selectedIntegration);
    if (result instanceof Promise) {
      savingRef.current = true;
      setIsSaving(true);
      try {
        await result;
      } finally {
        savingRef.current = false;
        setIsSaving(false);
      }
    }
  };

  return (
    <div className="p-4 overflow-y-auto pb-20" style={{ maxHeight: "80vh" }}>
      <div className={`space-y-6 ${isReadOnly ? "pointer-events-none opacity-60" : ""}`} aria-disabled={isReadOnly}>
        {/* Node identification section — always visible */}
        <div className="flex flex-col gap-2">
          <Label className="min-w-[100px] text-left">
            Name
            <span className="text-gray-800 ml-1">*</span>
            {hasNodeNameError && <span className="text-red-500 text-xs ml-2">Required</span>}
          </Label>
          <Input
            data-testid="node-name-input"
            type="text"
            value={currentNodeName}
            onChange={(e) => setCurrentNodeName(e.target.value)}
            placeholder="Enter a name for this node"
            autoFocus
            className="shadow-none"
            disabled={isReadOnly}
          />
        </div>

        {/* Integration section — one container, three states: Connect / error or incomplete / ready */}
        {integrationName && (
          <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
            {!allowIntegrations ? (
              <div className="bg-gray-50 dark:bg-gray-900/30 border border-gray-200 dark:border-gray-700 rounded-md p-3 text-sm text-gray-600 dark:text-gray-300">
                You don't have permission to view integrations.
              </div>
            ) : readyIntegrationsOfType.length === 0 ? (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <IntegrationIcon
                    integrationName={integrationName}
                    iconSlug={integrationDefinition?.icon}
                    className="h-5 w-5 flex-shrink-0 text-gray-500 dark:text-gray-400"
                  />
                  <span className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                    {getIntegrationTypeDisplayName(undefined, integrationName) || integrationName} Integration
                  </span>
                </div>

                {inlineWebhookSetup ? (
                  <>
                    <p className="text-sm text-gray-800 dark:text-gray-200">
                      Copy the webhook URL below and complete the required steps in your integration provider. Then
                      enter any required secrets below.
                    </p>
                    <div>
                      <Label className="text-gray-800 dark:text-gray-100 mb-2">Webhook URL</Label>
                      <div className="flex gap-2">
                        <Input
                          type="text"
                          readOnly
                          value={inlineWebhookSetup.webhookUrl}
                          className="font-mono text-sm shadow-none"
                        />
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              await navigator.clipboard.writeText(inlineWebhookSetup.webhookUrl);
                            } catch {
                              showErrorToast("Failed to copy to clipboard");
                            }
                          }}
                        >
                          Copy
                        </Button>
                      </div>
                    </div>
                    {(integrationDefinitionFull?.configuration ?? [])
                      .filter((f: ConfigurationField) => {
                        if (!f.name) return false;
                        return f.name === "signingSecret" || f.name === "webhookSigningSecret";
                      })
                      .map((field) => (
                        <ConfigurationFieldRenderer
                          key={field.name}
                          field={field}
                          value={inlineIntegrationConfig[field.name!]}
                          onChange={(value) =>
                            setInlineIntegrationConfig((prev) => ({
                              ...prev,
                              [field.name || ""]: value,
                            }))
                          }
                          allValues={inlineIntegrationConfig}
                          domainId={organizationId ?? domainId ?? ""}
                          domainType="DOMAIN_TYPE_ORGANIZATION"
                          organizationId={organizationId ?? domainId ?? ""}
                        />
                      ))}
                    <LoadingButton
                      color="blue"
                      onClick={() => void handleInlineWebhookComplete()}
                      loading={isInlineWebhookCompleting}
                      loadingText="Completing..."
                      disabled={isReadOnly}
                    >
                      Complete Setup
                    </LoadingButton>
                  </>
                ) : inlineBrowserAction ? (
                  <>
                    <IntegrationInstructions
                      description={
                        inlineBrowserActionCompleted
                          ? [integrationDefinitionFull?.instructions?.trim(), inlineBrowserAction.description]
                              .filter(Boolean)
                              .join("\n\n") || "Integration is being configured. It may take a moment to become ready."
                          : inlineBrowserAction.description ||
                            integrationDefinitionFull?.instructions?.trim() ||
                            "Click Continue to authorize this integration."
                      }
                      onContinue={
                        !inlineBrowserActionCompleted && inlineBrowserAction.url
                          ? () => void handleInlineBrowserActionContinue()
                          : undefined
                      }
                    />
                    {!inlineBrowserActionCompleted && inlineConfigFields.length > 0 && (
                      <div className="space-y-4">
                        {inlineConfigFields.map((field: ConfigurationField) => {
                          if (!field.name) return null;
                          return (
                            <ConfigurationFieldRenderer
                              key={field.name}
                              field={field}
                              value={inlineIntegrationConfig[field.name]}
                              onChange={(value) =>
                                setInlineIntegrationConfig((prev) => ({
                                  ...prev,
                                  [field.name || ""]: value,
                                }))
                              }
                              allValues={inlineIntegrationConfig}
                              domainId={organizationId ?? domainId ?? ""}
                              domainType="DOMAIN_TYPE_ORGANIZATION"
                              organizationId={organizationId ?? domainId ?? ""}
                            />
                          );
                        })}
                      </div>
                    )}
                    {inlineBrowserActionCompleted && (
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Waiting for the integration to become ready. This page will update automatically.
                      </p>
                    )}
                  </>
                ) : (
                  <>
                    <div>
                      <Label className="text-gray-800 dark:text-gray-100 mb-2">
                        Integration Name
                        <span className="text-gray-800 ml-1">*</span>
                      </Label>
                      <Input
                        type="text"
                        value={inlineIntegrationName}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setInlineIntegrationName(e.target.value)}
                        placeholder="e.g., my-app-integration"
                        disabled={isReadOnly || !allowCreateIntegrations}
                        className="shadow-none"
                      />
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                        A unique name for this integration
                      </p>
                    </div>

                    {inlineConfigFields.length > 0 && (
                      <div className="space-y-4">
                        {inlineConfigFields.map((field: ConfigurationField) => {
                          if (!field.name) return null;
                          return (
                            <ConfigurationFieldRenderer
                              key={field.name}
                              field={field}
                              value={inlineIntegrationConfig[field.name]}
                              onChange={(value) =>
                                setInlineIntegrationConfig((prev) => ({
                                  ...prev,
                                  [field.name || ""]: value,
                                }))
                              }
                              allValues={inlineIntegrationConfig}
                              domainId={organizationId ?? domainId ?? ""}
                              domainType="DOMAIN_TYPE_ORGANIZATION"
                              organizationId={organizationId ?? domainId ?? ""}
                            />
                          );
                        })}
                      </div>
                    )}

                    <LoadingButton
                      color="blue"
                      onClick={() => void handleInlineCreate()}
                      disabled={!inlineIntegrationName?.trim() || isReadOnly || !allowCreateIntegrations}
                      loading={isInlineCreating}
                      loadingText="Connecting..."
                      className="flex items-center gap-2"
                    >
                      Connect
                    </LoadingButton>
                  </>
                )}
              </div>
            ) : (
              <>
                <div className="flex flex-col gap-2">
                  <Label className="min-w-[100px] text-left">
                    Integration
                    <span className="text-gray-800 ml-1">*</span>
                    {showValidation && validationErrors.has("integration") && (
                      <span className="text-red-500 text-xs ml-2">Required</span>
                    )}
                  </Label>
                  <p className="text-xs text-gray-500">Instance</p>
                  <Select
                    value={selectedIntegration?.id || ""}
                    onValueChange={(value) => {
                      if (value === CONNECT_ANOTHER_INSTANCE_VALUE) {
                        if (!isReadOnly && allowCreateIntegrations && onOpenCreateIntegrationDialog) {
                          onOpenCreateIntegrationDialog();
                        }
                        return;
                      }
                      const integration = readyIntegrationsOfType.find((i) => i.metadata?.id === value);
                      if (integration) {
                        setSelectedIntegration({
                          id: integration.metadata?.id,
                          name: integration.metadata?.name,
                        });
                      }
                    }}
                    disabled={isReadOnly}
                  >
                    <SelectTrigger className="w-full shadow-none">
                      <SelectValue placeholder="Select an installation" />
                    </SelectTrigger>
                    <SelectContent>
                      {readyIntegrationsOfType.map((integration) => {
                        const instanceName = integration.metadata?.name;
                        const typeName = integration.spec?.integrationName;
                        const displayName =
                          instanceName?.toLowerCase() === typeName?.toLowerCase()
                            ? getIntegrationTypeDisplayName(undefined, typeName) || instanceName
                            : instanceName;
                        return (
                          <SelectItem key={integration.metadata?.id} value={integration.metadata?.id || ""}>
                            {displayName || "Unnamed integration"}
                          </SelectItem>
                        );
                      })}
                      {onOpenCreateIntegrationDialog && allowCreateIntegrations && (
                        <>
                          <SelectSeparator />
                          <SelectItem value={CONNECT_ANOTHER_INSTANCE_VALUE}>+ Connect another instance</SelectItem>
                        </>
                      )}
                    </SelectContent>
                  </Select>
                </div>
                {selectedIntegrationFull && (
                  <>
                    <p className="py-2 text-xs text-gray-500">Connection</p>
                    {(() => {
                      const hasIntegrationError =
                        selectedIntegrationFull.status?.state === "error" &&
                        !!selectedIntegrationFull.status?.stateDescription;

                      const integrationStatusCard = (
                        <div
                          className={`border border-gray-300 dark:border-gray-700 rounded-md bg-stripe-diagonal p-3 flex items-center justify-between gap-4 ${
                            selectedIntegrationFull.status?.state === "ready"
                              ? "bg-green-100 dark:bg-green-950/30"
                              : selectedIntegrationFull.status?.state === "error"
                                ? "bg-red-100 dark:bg-red-950/30"
                                : "bg-orange-100 dark:bg-orange-950/30"
                          }`}
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <IntegrationIcon
                              integrationName={selectedIntegrationFull.spec?.integrationName}
                              iconSlug={integrationDefinition?.icon}
                              className="mt-0.5 h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
                            />
                            <div className="min-w-0">
                              <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
                                {getIntegrationTypeDisplayName(
                                  undefined,
                                  selectedIntegrationFull.spec?.integrationName,
                                ) || "Integration"}
                              </h3>
                            </div>
                          </div>
                          <div className="flex items-center gap-2 flex-shrink-0">
                            <span
                              className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                selectedIntegrationFull.status?.state === "ready"
                                  ? "border border-green-950/15 bg-green-100 text-green-800 dark:border-green-950/15 dark:bg-green-900/30 dark:text-green-400"
                                  : selectedIntegrationFull.status?.state === "error"
                                    ? "border border-red-950/15 bg-red-100 text-red-800 dark:border-red-950/15 dark:bg-red-900/30 dark:text-red-400"
                                    : "border border-orange-950/15 bg-orange-100 text-yellow-800 dark:border-orange-950/15 dark:bg-orange-950/30 dark:text-yellow-400"
                              }`}
                            >
                              {selectedIntegrationFull.status?.state
                                ? selectedIntegrationFull.status.state.charAt(0).toUpperCase() +
                                  selectedIntegrationFull.status.state.slice(1)
                                : "Unknown"}
                            </span>
                            {selectedIntegrationFull.metadata?.id && onOpenConfigureIntegrationDialog && (
                              <Button
                                variant="outline"
                                size="sm"
                                className="text-sm py-1.5"
                                onClick={() => onOpenConfigureIntegrationDialog(selectedIntegrationFull.metadata!.id!)}
                                disabled={isReadOnly || !allowUpdateIntegrations}
                              >
                                Configure...
                              </Button>
                            )}
                          </div>
                        </div>
                      );

                      if (hasIntegrationError) {
                        return (
                          <SimpleTooltip content={selectedIntegrationFull.status?.stateDescription || ""}>
                            {integrationStatusCard}
                          </SimpleTooltip>
                        );
                      }

                      return integrationStatusCard;
                    })()}
                  </>
                )}
              </>
            )}
          </div>
        )}

        {/* Configuration section */}
        {configurationFields && configurationFields.length > 0 && shouldShowConfiguration && (
          <div className="border-t border-gray-200 dark:border-gray-700 pt-6 space-y-4">
            {configurationFields.map((field) => {
              if (!field.name) return null;
              const fieldName = field.name;
              return (
                <ConfigurationFieldRenderer
                  allowExpressions={true}
                  key={fieldName}
                  field={field}
                  value={nodeConfiguration[fieldName]}
                  onChange={(value) => {
                    const newConfig = {
                      ...nodeConfiguration,
                      [fieldName]: value,
                    };
                    setNodeConfiguration(filterVisibleFields(newConfig));
                  }}
                  allValues={nodeConfiguration}
                  domainId={domainId}
                  domainType={domainType}
                  organizationId={domainId}
                  integrationId={selectedIntegration?.id}
                  hasError={
                    showValidation &&
                    (validationErrors.has(fieldName) ||
                      // Check for nested errors in this field
                      Array.from(validationErrors).some(
                        (error) => error.startsWith(`${fieldName}.`) || error.startsWith(`${fieldName}[`),
                      ))
                  }
                  validationErrors={showValidation ? validationErrors : undefined}
                  fieldPath={fieldName}
                  realtimeValidationErrors={realtimeValidationErrors}
                  enableRealtimeValidation={true}
                  autocompleteExampleObj={resolvedAutocompleteExampleObj}
                />
              );
            })}
          </div>
        )}

        {/* Custom field section */}
        {customField && shouldShowConfiguration && (
          <div
            className={
              configurationFields && configurationFields.length > 0
                ? ""
                : "border-t border-gray-200 dark:border-gray-700 pt-6"
            }
          >
            {customField(nodeConfiguration)}
          </div>
        )}
      </div>

      <div className="flex gap-2 justify-end mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
        <LoadingButton
          data-testid="save-node-button"
          variant="default"
          onClick={handleSave}
          disabled={isReadOnly}
          loading={isSaving}
          loadingText="Saving..."
        >
          Save
        </LoadingButton>
      </div>
    </div>
  );
}
