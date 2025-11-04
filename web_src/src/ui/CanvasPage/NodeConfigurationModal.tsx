import { ComponentsConfigurationField } from "@/api-client";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

interface NodeConfigurationModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  nodeLabel?: string;
  configuration: Record<string, unknown>;
  configurationFields: ComponentsConfigurationField[];
  onSave: (updatedConfiguration: Record<string, unknown>, updatedNodeName: string) => void;
  domainId?: string;
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION";
}

export function NodeConfigurationModal({
  isOpen,
  onClose,
  nodeName,
  nodeLabel,
  configuration,
  configurationFields,
  onSave,
  domainId,
  domainType,
}: NodeConfigurationModalProps) {
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);

  const isFieldEmpty = (value: unknown): boolean => {
    if (value === null || value === undefined) return true;
    if (typeof value === "string") return value.trim() === "";
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === "object") return Object.keys(value).length === 0;
    return false;
  };

  // Recursively validate nested fields in objects and lists
  const validateNestedFields = useCallback(
    (fields: ComponentsConfigurationField[], values: Record<string, unknown>, parentPath: string = ""): Set<string> => {
      const errors = new Set<string>();

      fields.forEach((field) => {
        if (!field.name) return;

        const fieldPath = parentPath ? `${parentPath}.${field.name}` : field.name;
        const value = values[field.name];

        // Check if this field itself is required and empty
        if (field.required && isFieldEmpty(value)) {
          errors.add(fieldPath);
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

  // Sync state when props change (e.g., when modal opens for a different node)
  useEffect(() => {
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName]);

  const handleSave = () => {
    if (validateAllFields()) {
      onSave(nodeConfiguration, currentNodeName);
      onClose();
    }
  };

  const handleClose = () => {
    // Reset to original configuration and name on cancel
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    onClose();
  };

  const displayLabel = nodeLabel || nodeName || "Node configuration";

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent
        className="max-w-2xl p-0"
        showCloseButton={false}
        aria-describedby={undefined} /* Disable DialogDescription */
      >
        <DialogHeader className="px-6 pt-6 pb-0 text-left">
          <DialogTitle>New {displayLabel}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="max-h-[80vh]">
          <div className="p-6">
            <div className="space-y-6">
              {/* Node identification section */}
              <div className="flex flex-col  gap-2 h-[60px]">
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
                  className={`flex-1 ${
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
                        onChange={(value) =>
                          setNodeConfiguration({
                            ...nodeConfiguration,
                            [fieldName]: value,
                          })
                        }
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

            <DialogFooter className="mt-6">
              <Button data-testid="cancel-node-add-button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button data-testid="add-node-button" variant="default" onClick={handleSave}>
                Add
              </Button>
            </DialogFooter>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
