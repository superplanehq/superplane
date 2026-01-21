import {
  AuthorizationDomainType,
  ComponentsAppInstallationRef,
  ConfigurationField,
  OrganizationsAppInstallation,
} from "@/api-client";
import { useCallback, useEffect, useMemo, useState, ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { AlertTriangle } from "lucide-react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { isFieldRequired, isFieldVisible, parseDefaultValues, validateFieldForSubmission } from "@/utils/components";
import { useRealtimeValidation } from "@/hooks/useRealtimeValidation";

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
    appInstallationRef?: ComponentsAppInstallationRef,
  ) => void;
  onCancel?: () => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  customField?: (configuration: Record<string, unknown>) => ReactNode;
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
  installedApplications?: OrganizationsAppInstallation[];
  autocompleteExampleObj?: Record<string, unknown> | null;
}

export function SettingsTab({
  nodeId,
  nodeName,
  nodeLabel,
  configuration,
  configurationFields,
  onSave,
  onCancel: _onCancel,
  domainId,
  domainType,
  customField,
  appName,
  appInstallationRef,
  installedApplications = [],
  autocompleteExampleObj,
}: SettingsTabProps) {
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);
  const [selectedAppInstallation, setSelectedAppInstallation] = useState<ComponentsAppInstallationRef | undefined>(
    appInstallationRef,
  );
  const resolvedAutocompleteExampleObj = useMemo(() => {
    if (!autocompleteExampleObj) {
      return autocompleteExampleObj;
    }

    if (!nodeId) {
      return autocompleteExampleObj;
    }

    const nodeNameLabel = nodeLabel || nodeName;
    if (!nodeNameLabel) {
      return autocompleteExampleObj;
    }

    const base = autocompleteExampleObj as Record<string, unknown>;
    const existingNames = base.__nodeNames;
    const nextNodeNames =
      typeof existingNames === "object" && existingNames !== null && !Array.isArray(existingNames)
        ? { ...(existingNames as Record<string, unknown>), [nodeId]: nodeNameLabel }
        : { [nodeId]: nodeNameLabel };

    let nextGlobals: Record<string, unknown> = { ...base, __nodeNames: nextNodeNames };
    const currentValue = base[nodeId];
    if (typeof currentValue === "object" && currentValue !== null && !Array.isArray(currentValue)) {
      const currentRecord = currentValue as Record<string, unknown>;
      if (typeof currentRecord.__nodeName !== "string") {
        nextGlobals = {
          ...nextGlobals,
          [nodeId]: { ...currentRecord, __nodeName: nodeNameLabel },
        };
      }
    }

    return nextGlobals;
  }, [autocompleteExampleObj, nodeId, nodeLabel, nodeName]);

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

  // Filter installations by app name
  const availableInstallations = useMemo(() => {
    if (!appName) return [];
    return installedApplications.filter((app) => app.spec?.appName === appName && app.status?.state === "ready");
  }, [installedApplications, appName]);
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
    setSelectedAppInstallation(appInstallationRef);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, defaultValuesWithoutToggles, filterVisibleFields, appInstallationRef]);

  // Auto-select the first installation if none is selected or selection is invalid
  useEffect(() => {
    if (availableInstallations.length === 0) {
      if (selectedAppInstallation) {
        setSelectedAppInstallation(undefined);
      }
      return;
    }

    const selectedId = selectedAppInstallation?.id;
    const hasSelected = selectedId
      ? availableInstallations.some((installation) => installation.metadata?.id === selectedId)
      : false;
    if (hasSelected) {
      return;
    }

    const firstInstallation = availableInstallations[0];
    setSelectedAppInstallation({
      id: firstInstallation.metadata?.id,
      name: firstInstallation.metadata?.name,
    });
  }, [availableInstallations, selectedAppInstallation]);

  const shouldShowConfiguration = !appName || !!selectedAppInstallation?.id;

  const handleSave = () => {
    validateNow();
    onSave(nodeConfiguration, currentNodeName, selectedAppInstallation);
  };

  return (
    <div className="p-4 overflow-y-auto pb-20" style={{ maxHeight: "80vh" }}>
      <div className="space-y-6">
        {/* Node identification section */}
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
          />
        </div>

        {/* App Installation section */}
        {appName && (
          <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
            {availableInstallations.length === 0 ? (
              // Warning when no installations available
              <Alert className="bg-orange-50 dark:bg-amber-950">
                <AlertTriangle className="h-4 w-4" />
                <AlertTitle className="text-amber-900 dark:text-amber-100">App Installation Required</AlertTitle>
                <AlertDescription className="text-amber-900 dark:text-amber-200">
                  This component requires a {appName} installation.{" "}
                  <a
                    href={`/${domainId}/settings/applications`}
                    className="text-blue-600 dark:text-blue-400 underline hover:text-blue-800 dark:hover:text-blue-300"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Install {appName}
                  </a>{" "}
                  to configure this component.
                </AlertDescription>
              </Alert>
            ) : (
              // Select when installations are available
              <div className="flex flex-col gap-2">
                <Label className="min-w-[100px] text-left">
                  App Installation
                  <span className="text-gray-800 ml-1">*</span>
                  {showValidation && validationErrors.has("appInstallation") && (
                    <span className="text-red-500 text-xs ml-2">Required</span>
                  )}
                </Label>
                <Select
                  value={selectedAppInstallation?.id || ""}
                  onValueChange={(value) => {
                    const installation = availableInstallations.find((app) => app.metadata?.id === value);
                    if (installation) {
                      setSelectedAppInstallation({
                        id: installation.metadata?.id,
                        name: installation.metadata?.name,
                      });
                    }
                  }}
                >
                  <SelectTrigger className="w-full shadow-none">
                    <SelectValue placeholder="Select an installation" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableInstallations.map((installation) => (
                      <SelectItem key={installation.metadata?.id} value={installation.metadata?.id || ""}>
                        {installation.metadata?.name || "Unnamed installation"}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
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
                  appInstallationId={selectedAppInstallation?.id}
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
        <Button data-testid="save-node-button" variant="default" onClick={handleSave}>
          Save
        </Button>
      </div>
    </div>
  );
}
