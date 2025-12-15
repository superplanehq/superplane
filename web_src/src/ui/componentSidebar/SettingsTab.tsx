import { AuthorizationDomainType, ConfigurationField } from "@/api-client";
import { useCallback, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { isFieldRequired, isFieldVisible, parseDefaultValues, validateFieldForSubmission } from "@/utils/components";

interface SettingsTabProps {
  mode: "create" | "edit";
  nodeName: string;
  nodeLabel?: string;
  configuration: Record<string, unknown>;
  configurationFields: ConfigurationField[];
  onSave: (updatedConfiguration: Record<string, unknown>, updatedNodeName: string) => void;
  onCancel?: () => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
}

export function SettingsTab({
  mode,
  nodeName,
  nodeLabel: _nodeLabel,
  configuration,
  configurationFields,
  onSave,
  onCancel,
  domainId,
  domainType,
}: SettingsTabProps) {
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);

  const defaultValues = useMemo(() => {
    return parseDefaultValues(configurationFields);
  }, [configurationFields]);

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

  const validateAllFields = useCallback((): boolean => {
    const errors = validateNestedFields(configurationFields, nodeConfiguration);

    if (isFieldEmpty(currentNodeName)) {
      errors.add("nodeName");
    }
    setValidationErrors(errors);
    setShowValidation(true);
    return errors.size === 0;
  }, [configurationFields, nodeConfiguration, currentNodeName, validateNestedFields]);

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
      newConfig = defaultValues;
    } else {
      newConfig = { ...defaultValues, ...configuration };
    }

    setNodeConfiguration(filterVisibleFields(newConfig));
    setCurrentNodeName(nodeName);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, defaultValues, filterVisibleFields]);

  const handleSave = () => {
    if (validateAllFields()) {
      onSave(nodeConfiguration, currentNodeName);
    }
  };

  const handleCancel = () => {
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    setValidationErrors(new Set());
    setShowValidation(false);
    onCancel?.();
  };

  return (
    <div className="px-3 py-3">
      <div className="space-y-6">
        {/* Node identification section */}
        <div className="flex flex-col gap-2 h-[60px]">
          <Label
            className={`min-w-[100px] text-left ${
              showValidation && validationErrors.has("nodeName") ? "text-red-600 dark:text-red-400" : ""
            }`}
          >
            Node Name
            <span className="text-red-500 ml-1">*</span>
            {showValidation && validationErrors.has("nodeName") && (
              <span className="text-red-500 text-xs ml-2">- required field</span>
            )}
          </Label>
          <Input
            data-testid="node-name-input"
            type="text"
            value={currentNodeName}
            onChange={(e) => setCurrentNodeName(e.target.value)}
            placeholder="Enter a name for this node"
            autoFocus
            className={`flex-1 shadow-none ${
              showValidation && validationErrors.has("nodeName") ? "border-red-500 border-2" : ""
            }`}
          />
        </div>

        {/* Configuration section */}
        {configurationFields && configurationFields.length > 0 && (
          <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
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
                      ...defaultValues,
                      ...nodeConfiguration,
                      [fieldName]: value,
                    };
                    setNodeConfiguration(filterVisibleFields(newConfig));
                  }}
                  allValues={nodeConfiguration}
                  domainId={domainId}
                  domainType={domainType}
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
                />
              );
            })}
          </div>
        )}
      </div>

      <div className="flex gap-2 justify-end mt-6 pt-6 border-t border-gray-200 dark:border-zinc-700">
        {mode === "create" && (
          <Button data-testid="cancel-node-add-button" variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
        )}
        <Button data-testid="add-node-button" variant="default" onClick={handleSave}>
          {mode === "edit" ? "Save" : "Add"}
        </Button>
      </div>
    </div>
  );
}
