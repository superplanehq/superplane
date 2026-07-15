import type {
  AuthorizationDomainType,
  ComponentsIntegrationRef,
  ConfigurationField,
  OrganizationsIntegration,
} from "@/api-client";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import {
  filterVisibleConfiguration,
  isFieldRequired,
  parseDefaultValues,
  validateFieldForSubmission,
} from "@/lib/components";
import { useRealtimeValidation } from "@/hooks/useRealtimeValidation";
import { buildConfigurationDisplayModel } from "./configurationView/buildConfigurationDisplayModel";
import { ConfigurationView } from "./configurationView/ConfigurationView";
import { SimpleTooltip } from "./SimpleTooltip";
import { cn } from "@/lib/utils";

const REQUIRED_FIELD_BADGE_CLASS =
  "ml-2 inline-flex items-center rounded border border-orange-300 px-1 py-0.5 text-[10px] uppercase tracking-wide leading-none text-orange-500 bg-orange-50";

const SETTINGS_TAB_DIVIDER_CLASS = "border-t border-slate-950/15 pt-6 dark:border-gray-700/70";
const FORM_DISABLED_CURSOR_CLASS = "cursor-not-allowed";
const FORM_DISABLED_SURFACE_CLASS =
  "pointer-events-none opacity-70 dark:opacity-60 [&_[disabled]]:opacity-100 [&_[data-slot=control]]:opacity-100";

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
  autocompleteExampleObj?: Record<string, unknown> | null;
  onOpenCreateIntegrationDialog?: () => void;
  onOpenConfigureIntegrationDialog?: (integrationId: string) => void;
  readOnly?: boolean;
  formDisabled?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
}

function buildAutosaveSnapshot(
  configuration: Record<string, unknown>,
  nodeName: string,
  integrationRef?: ComponentsIntegrationRef,
): string {
  return JSON.stringify({
    configuration,
    nodeName,
    integrationRef: integrationRef
      ? {
          id: integrationRef.id || "",
          name: integrationRef.name || "",
        }
      : null,
  });
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
  autocompleteExampleObj,
  onOpenCreateIntegrationDialog,
  onOpenConfigureIntegrationDialog,
  readOnly = false,
  formDisabled = false,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
}: SettingsTabProps) {
  const CONNECT_ANOTHER_INSTANCE_VALUE = "__connect_another_instance__";
  const isReadOnly = readOnly ?? false;
  const isFormDisabled = formDisabled ?? false;
  const isInteractionDisabled = isReadOnly || isFormDisabled;
  const allowIntegrations = canReadIntegrations ?? true;
  const allowCreateIntegrations = canCreateIntegrations ?? true;
  const allowUpdateIntegrations = canUpdateIntegrations ?? true;
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);
  const [selectedIntegration, setSelectedIntegration] = useState<ComponentsIntegrationRef | undefined>(integrationRef);
  const savingRef = useRef(false);
  const autosaveTimerRef = useRef<number | null>(null);
  const autosaveBaselineSnapshotRef = useRef(buildAutosaveSnapshot(configuration || {}, nodeName, integrationRef));
  const pendingAutosaveSnapshotRef = useRef<string | null>(null);
  // Use autocompleteExampleObj directly - current node is already filtered out
  const resolvedAutocompleteExampleObj = autocompleteExampleObj;

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
    return integrations.filter((i) => i.metadata?.integrationName === integrationName);
  }, [integrations, integrationName]);
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

    const filteredConfig = filterVisibleFields(newConfig);
    autosaveBaselineSnapshotRef.current = buildAutosaveSnapshot(filteredConfig, nodeName, integrationRef);
    pendingAutosaveSnapshotRef.current = null;
    setNodeConfiguration(filteredConfig);
    setCurrentNodeName(nodeName);
    setSelectedIntegration(integrationRef);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, defaultValuesWithoutToggles, filterVisibleFields, integrationRef]);

  // Auto-select the first installation if none is selected or selection is invalid
  useEffect(() => {
    if (isInteractionDisabled) {
      return;
    }

    if (integrationsOfType.length === 0) {
      if (selectedIntegration) {
        autosaveBaselineSnapshotRef.current = buildAutosaveSnapshot(nodeConfiguration, currentNodeName, undefined);
        setSelectedIntegration(undefined);
      }
      return;
    }

    const selectedId = selectedIntegration?.id;
    const hasSelected = selectedId
      ? integrationsOfType.some((integration) => integration.metadata?.id === selectedId)
      : false;
    if (hasSelected) {
      return;
    }

    const firstIntegration = integrationsOfType[0];
    const nextIntegration = {
      id: firstIntegration.metadata?.id,
      name: firstIntegration.metadata?.name,
    };
    autosaveBaselineSnapshotRef.current = buildAutosaveSnapshot(nodeConfiguration, currentNodeName, nextIntegration);
    setSelectedIntegration({
      id: firstIntegration.metadata?.id,
      name: firstIntegration.metadata?.name,
    });
  }, [integrationsOfType, isInteractionDisabled, selectedIntegration, nodeConfiguration, currentNodeName]);

  const shouldShowConfiguration = true;
  const shouldAutosaveOnChangeByFieldType = useCallback((fieldType: ConfigurationField["type"] | undefined) => {
    // Text-like editors save on blur; discrete/destructive controls save on change.
    if (!fieldType) {
      return false;
    }
    return ![
      "string",
      "text",
      "xml",
      "expression",
      "number",
      "url",
      "date",
      "datetime",
      "time",
      "cron",
      "git-ref",
      "app",
    ].includes(fieldType);
  }, []);

  const updateAutosaveBaseline = useCallback((snapshot: string) => {
    autosaveBaselineSnapshotRef.current = snapshot;
    pendingAutosaveSnapshotRef.current = null;
  }, []);

  const queuePendingAutosave = useCallback((snapshot: string) => {
    pendingAutosaveSnapshotRef.current = snapshot;
  }, []);

  const flushPendingAutosave = useCallback(() => {
    const pendingSnapshot = pendingAutosaveSnapshotRef.current;
    if (!pendingSnapshot || pendingSnapshot === autosaveBaselineSnapshotRef.current) {
      pendingAutosaveSnapshotRef.current = null;
      return;
    }

    pendingAutosaveSnapshotRef.current = null;
    window.setTimeout(() => {
      void handleSaveRef.current();
    }, 0);
  }, []);

  const handleSave = useCallback(async () => {
    if (isInteractionDisabled) {
      return;
    }

    const snapshot = buildAutosaveSnapshot(nodeConfiguration, currentNodeName, selectedIntegration);
    if (snapshot === autosaveBaselineSnapshotRef.current) {
      pendingAutosaveSnapshotRef.current = null;
      return;
    }

    validateNow();
    if (currentNodeName.trim() === "") {
      return;
    }

    if (savingRef.current) {
      queuePendingAutosave(snapshot);
      return;
    }

    const result = onSave(nodeConfiguration, currentNodeName, selectedIntegration);
    if (!(result instanceof Promise)) {
      updateAutosaveBaseline(snapshot);
      return;
    }

    savingRef.current = true;
    try {
      await result;
      updateAutosaveBaseline(snapshot);
    } finally {
      savingRef.current = false;
      flushPendingAutosave();
    }
  }, [
    isInteractionDisabled,
    validateNow,
    currentNodeName,
    selectedIntegration,
    nodeConfiguration,
    onSave,
    queuePendingAutosave,
    updateAutosaveBaseline,
    flushPendingAutosave,
  ]);

  const handleSaveRef = useRef(handleSave);
  handleSaveRef.current = handleSave;

  const requestAutosave = useCallback(() => {
    if (isInteractionDisabled) {
      return;
    }

    if (autosaveTimerRef.current !== null) {
      window.clearTimeout(autosaveTimerRef.current);
    }
    autosaveTimerRef.current = window.setTimeout(() => {
      autosaveTimerRef.current = null;
      void handleSaveRef.current();
    }, 300);
  }, [isInteractionDisabled]);

  // Flush unsaved changes on unmount (e.g. when user switches away from the Settings tab)
  useEffect(() => {
    return () => {
      if (isInteractionDisabled) {
        return;
      }
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
        autosaveTimerRef.current = null;
      }
      void handleSaveRef.current();
    };
  }, [isInteractionDisabled]);

  useEffect(() => {
    return () => {
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (isInteractionDisabled) {
      return;
    }
    const snapshot = buildAutosaveSnapshot(nodeConfiguration, currentNodeName, selectedIntegration);
    if (snapshot === autosaveBaselineSnapshotRef.current) {
      return;
    }
    if (currentNodeName.trim() === "") {
      return;
    }

    // Safety net for flows that do not blur inputs (e.g. scripted E2E interactions).
    const fallbackTimer = window.setTimeout(() => {
      void handleSaveRef.current();
    }, 1200);

    return () => {
      window.clearTimeout(fallbackTimer);
    };
  }, [isInteractionDisabled, nodeConfiguration, currentNodeName, selectedIntegration]);

  const configurationDisplayModel = useMemo(
    () =>
      buildConfigurationDisplayModel({
        configuration: nodeConfiguration,
        configurationFields,
        integrationName,
        integrationRef,
        integrations,
        allowIntegrations,
      }),
    [allowIntegrations, configurationFields, integrationName, integrationRef, integrations, nodeConfiguration],
  );

  if (isReadOnly && !isFormDisabled) {
    return (
      <div className="p-4 pb-24">
        <div className="space-y-6">
          <ConfigurationView model={configurationDisplayModel} />
          {customField && shouldShowConfiguration && (
            <div className={configurationFields && configurationFields.length > 0 ? "" : SETTINGS_TAB_DIVIDER_CLASS}>
              {customField(nodeConfiguration)}
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div
      className={cn("p-4 pb-24 overflow-x-hidden", isFormDisabled && FORM_DISABLED_CURSOR_CLASS)}
      data-testid="settings-tab-form"
      onBlurCapture={(event) => {
        if (isFormDisabled) {
          return;
        }

        const target = event.target as HTMLElement | null;
        if (!target) {
          return;
        }

        if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
          requestAutosave();
        }
      }}
    >
      <div
        className={cn("space-y-6", isFormDisabled && FORM_DISABLED_SURFACE_CLASS)}
        {...(isFormDisabled ? { inert: true } : {})}
      >
        {/* Node identification section — always visible */}
        <div className="flex flex-col gap-2">
          <Label className="min-w-[100px] text-left">
            Name
            <span className="text-gray-800 ml-1">*</span>
            {hasNodeNameError && <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>}
          </Label>
          <Input
            data-testid="node-name-input"
            type="text"
            value={currentNodeName}
            onChange={(e) => {
              setCurrentNodeName(e.target.value);
              requestAutosave();
            }}
            placeholder="Enter a name for this node"
            autoFocus={!isFormDisabled}
            disabled={isFormDisabled}
            className="shadow-none"
          />
        </div>

        {/* Run title field — rendered right after name, before the separator */}
        {(() => {
          const runTitleField = configurationFields?.find((f) => f.name === "customName");
          if (!runTitleField || !shouldShowConfiguration) return null;
          return (
            <div>
              <ConfigurationFieldRenderer
                allowExpressions={true}
                field={runTitleField}
                value={nodeConfiguration[runTitleField.name!]}
                onChange={(value) => {
                  setNodeConfiguration((prev) => ({
                    ...prev,
                    [runTitleField.name!]: value,
                  }));
                  if (value === undefined || value === null || value === "") {
                    requestAutosave();
                  }
                }}
                allValues={nodeConfiguration}
                domainId={domainId}
                domainType={domainType}
                organizationId={domainId}
                autocompleteExampleObj={resolvedAutocompleteExampleObj}
                realtimeValidationErrors={realtimeValidationErrors}
                enableRealtimeValidation={true}
                readOnly={isFormDisabled}
                preserveEditLayout={isFormDisabled}
              />
            </div>
          );
        })()}

        {/* Integration section — one container, three states: Connect / error or incomplete / ready */}
        {integrationName && (
          <div className={SETTINGS_TAB_DIVIDER_CLASS}>
            {!allowIntegrations ? (
              <div className="bg-gray-50 dark:bg-gray-900/30 border border-gray-200 dark:border-gray-700 rounded-md p-3 text-sm text-gray-600 dark:text-gray-300">
                You don't have permission to view integrations.
              </div>
            ) : integrationsOfType.length === 0 ? (
              /* No integration: Connect XYZ — always use helper so "github" shows as "GitHub" */
              <div className="bg-orange-100 dark:bg-orange-950/30 border border-orange-950/15 rounded-md bg-stripe-diagonal p-3 flex items-center justify-between gap-4">
                <div className="flex items-center gap-2 min-w-0">
                  <IntegrationIcon
                    integrationName={integrationName}
                    iconSlug={integrationDefinition?.icon}
                    className="h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
                  />
                  <span className="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
                    {getIntegrationTypeDisplayName(undefined, integrationName) || integrationName} Integration
                  </span>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onOpenCreateIntegrationDialog}
                  className="flex-shrink-0"
                  disabled={!allowCreateIntegrations || isFormDisabled}
                >
                  Connect
                </Button>
              </div>
            ) : (
              <>
                <div className="flex flex-col gap-2">
                  <Label className="min-w-[100px] text-left">
                    Integration
                    <span className="text-gray-800 ml-1">*</span>
                    {showValidation && validationErrors.has("integration") && (
                      <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>
                    )}
                  </Label>
                  <p className="text-xs text-gray-500">Instance</p>
                  <Select
                    value={selectedIntegration?.id || ""}
                    onValueChange={(value) => {
                      if (isFormDisabled) {
                        return;
                      }
                      if (value === CONNECT_ANOTHER_INSTANCE_VALUE) {
                        if (allowCreateIntegrations && onOpenCreateIntegrationDialog) {
                          onOpenCreateIntegrationDialog();
                        }
                        return;
                      }
                      const integration = integrationsOfType.find((i) => i.metadata?.id === value);
                      if (integration) {
                        setSelectedIntegration({
                          id: integration.metadata?.id,
                          name: integration.metadata?.name,
                        });
                        requestAutosave();
                      }
                    }}
                    disabled={isFormDisabled}
                  >
                    <SelectTrigger className="w-full shadow-none">
                      <SelectValue placeholder="Select an installation" />
                    </SelectTrigger>
                    <SelectContent>
                      {integrationsOfType.map((integration) => {
                        const instanceName = integration.metadata?.name;
                        const typeName = integration.metadata?.integrationName;
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
                      {onOpenCreateIntegrationDialog && allowCreateIntegrations && !isFormDisabled && (
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
                              integrationName={selectedIntegrationFull.metadata?.integrationName}
                              iconSlug={integrationDefinition?.icon}
                              className="mt-0.5 h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
                            />
                            <div className="min-w-0">
                              <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
                                {getIntegrationTypeDisplayName(
                                  undefined,
                                  selectedIntegrationFull.metadata?.integrationName,
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
                            {selectedIntegrationFull.metadata?.id &&
                              onOpenConfigureIntegrationDialog &&
                              !isFormDisabled && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="text-sm py-1.5"
                                  onClick={() =>
                                    onOpenConfigureIntegrationDialog(selectedIntegrationFull.metadata!.id!)
                                  }
                                  disabled={!allowUpdateIntegrations}
                                >
                                  Configure...
                                </Button>
                              )}
                          </div>
                        </div>
                      );

                      if (hasIntegrationError) {
                        return (
                          <SimpleTooltip
                            content={selectedIntegrationFull.status?.stateDescription || ""}
                            interactive={true}
                          >
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
          <div className={cn(SETTINGS_TAB_DIVIDER_CLASS, "space-y-4")}>
            {configurationFields.map((field) => {
              if (!field.name || field.name === "customName") return null;
              const fieldName = field.name;
              return (
                <ConfigurationFieldRenderer
                  allowExpressions={true}
                  key={fieldName}
                  field={field}
                  value={nodeConfiguration[fieldName]}
                  onChange={(value) => {
                    const previousValue = nodeConfiguration[fieldName];
                    setNodeConfiguration((previousConfiguration) => {
                      const newConfig = {
                        ...previousConfiguration,
                        [fieldName]: value,
                      };
                      return filterVisibleFields(newConfig);
                    });
                    const fieldWasCleared = value === undefined || value === null || value === "";
                    // Enabling a togglable field (null/undefined -> value) is a discrete action
                    // and must persist immediately. Otherwise a save-on-blur field type (e.g. text
                    // pre-filled with a default) would keep its value only in local state, so a run
                    // or reload before the editor blurs would drop the enabled value.
                    const togglableEnabled = field.togglable === true && previousValue == null && !fieldWasCleared;
                    if (fieldWasCleared || togglableEnabled || shouldAutosaveOnChangeByFieldType(field.type)) {
                      requestAutosave();
                    }
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
                  readOnly={isFormDisabled}
                  preserveEditLayout={isFormDisabled}
                />
              );
            })}
          </div>
        )}

        {/* Custom field section */}
        {customField && shouldShowConfiguration && (
          <div className={configurationFields && configurationFields.length > 0 ? "" : SETTINGS_TAB_DIVIDER_CLASS}>
            {customField(nodeConfiguration)}
          </div>
        )}
      </div>
    </div>
  );
}
