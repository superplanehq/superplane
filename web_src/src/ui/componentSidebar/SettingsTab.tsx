import {
  AuthorizationDomainType,
  ComponentsIntegrationRef,
  ConfigurationField,
  OrganizationsIntegration,
} from "@/api-client";
import { useCallback, useEffect, useMemo, useState, ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { isFieldRequired, isFieldVisible, parseDefaultValues, validateFieldForSubmission } from "@/utils/components";
import { useRealtimeValidation } from "@/hooks/useRealtimeValidation";
import { SimpleTooltip } from "./SimpleTooltip";

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
  ) => void;
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
  autocompleteExampleObj,
  onOpenCreateIntegrationDialog,
  onOpenConfigureIntegrationDialog,
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
    return integrations.filter((i) => i.spec?.integrationName === integrationName);
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
          const validationErrors = validateFieldForSubmission(field, value, values);

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
      const filtered = { ...config };
      configurationFields.forEach((field) => {
        if (field.name && !isFieldVisible(field, config)) {
          delete filtered[field.name];
        }
      });
      return filtered;
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

  // Auto-select the first installation if none is selected or selection is invalid
  useEffect(() => {
    if (integrationsOfType.length === 0) {
      if (selectedIntegration) {
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
    setSelectedIntegration({
      id: firstIntegration.metadata?.id,
      name: firstIntegration.metadata?.name,
    });
  }, [integrationsOfType, selectedIntegration]);

  const isIntegrationReady =
    !integrationName || !allowIntegrations || selectedIntegrationFull?.status?.state === "ready";
  const shouldShowConfiguration = (!integrationName || !!selectedIntegration?.id) && isIntegrationReady;

  const handleSave = () => {
    if (isReadOnly) {
      return;
    }
    validateNow();
    onSave(nodeConfiguration, currentNodeName, selectedIntegration);
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
                  disabled={isReadOnly || !allowCreateIntegrations}
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
                      const integration = integrationsOfType.find((i) => i.metadata?.id === value);
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
                      {integrationsOfType.map((integration) => {
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
        <Button data-testid="save-node-button" variant="default" onClick={handleSave} disabled={isReadOnly}>
          Save
        </Button>
      </div>
    </div>
  );
}
