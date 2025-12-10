import {
  AuthorizationDomainType,
  ComponentsAppInstallationRef,
  ConfigurationField,
  OrganizationsAppInstallation,
} from "@/api-client";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { isFieldRequired, validateFieldValue } from "@/utils/components";

interface NodeConfigurationModalProps {
  mode: "create" | "edit";
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  nodeLabel?: string;
  configuration: Record<string, unknown>;
  configurationFields: ConfigurationField[];
  onSave: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    appInstallationRef?: ComponentsAppInstallationRef
  ) => void;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
  installedApplications?: OrganizationsAppInstallation[];
}

export function NodeConfigurationModal({
  mode,
  isOpen,
  onClose,
  nodeName,
  nodeLabel,
  configuration,
  configurationFields,
  onSave,
  domainId,
  domainType,
  appName,
  appInstallationRef,
  installedApplications = [],
}: NodeConfigurationModalProps) {
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(configuration || {});
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [selectedAppInstallationId, setSelectedAppInstallationId] = useState<string | undefined>(
    appInstallationRef?.id
  );
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
          const validationErrors = validateFieldValue(field, value, values);

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

    // Validate app installation selection for components/triggers from applications
    if (appName && !selectedAppInstallationId) {
      errors.add("appInstallation");
    }

    setValidationErrors(errors);
    setShowValidation(true);
    return errors.size === 0;
  }, [configurationFields, nodeConfiguration, currentNodeName, validateNestedFields, appName, selectedAppInstallationId]);

  // Sync state when props change (e.g., when modal opens for a different node)
  useEffect(() => {
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    setSelectedAppInstallationId(appInstallationRef?.id);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, appInstallationRef]);

  const handleSave = () => {
    if (validateAllFields()) {
      let appInstallationRefToSave: ComponentsAppInstallationRef | undefined;

      // If this is a component/trigger from an application, include the app installation ref
      if (appName && selectedAppInstallationId) {
        const selectedInstallation = installedApplications.find((app) => app.id === selectedAppInstallationId);
        if (selectedInstallation) {
          appInstallationRefToSave = {
            id: selectedInstallation.id,
            name: selectedInstallation.installationName,
          };
        }
      }

      onSave(nodeConfiguration, currentNodeName, appInstallationRefToSave);
      onClose();
    }
  };

  const handleClose = () => {
    // Reset to original configuration and name on cancel
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    setSelectedAppInstallationId(appInstallationRef?.id);
    onClose();
  };

  const displayLabel = nodeLabel || nodeName || "Node configuration";
  const title = mode === "edit" ? `Edit ${displayLabel}` : `New ${displayLabel}`;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent
        className="max-w-2xl p-0"
        showCloseButton={false}
        aria-describedby={undefined} /* Disable DialogDescription */
      >
        <DialogHeader className="px-6 pt-6 pb-0 text-left">
          <DialogTitle>{title}</DialogTitle>
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
                  className={`flex-1 shadow-none ${
                    showValidation && validationErrors.has("nodeName") ? "border-red-500 border-2" : ""
                  }`}
                />
              </div>

              {/* App Installation selection for components/triggers from applications */}
              {appName && installedApplications.length > 0 && (
                <div className="flex flex-col gap-2 h-[60px]">
                  <Label
                    className={`min-w-[100px] text-left ${
                      showValidation && validationErrors.has("appInstallation") ? "text-red-600 dark:text-red-400" : ""
                    }`}
                  >
                    App Installation
                    <span className="text-red-500 ml-1">*</span>
                    {showValidation && validationErrors.has("appInstallation") && (
                      <span className="text-red-500 text-xs ml-2">- required field</span>
                    )}
                  </Label>
                  <Select value={selectedAppInstallationId} onValueChange={setSelectedAppInstallationId}>
                    <SelectTrigger
                      className={`${
                        showValidation && validationErrors.has("appInstallation") ? "border-red-500 border-2" : ""
                      }`}
                    >
                      <SelectValue placeholder="Select an app installation" />
                    </SelectTrigger>
                    <SelectContent>
                      {installedApplications
                        .filter((app) => app.appName === appName)
                        .map((app) => (
                          <SelectItem key={app.id} value={app.id!}>
                            {app.installationName}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

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
                {mode === "edit" ? "Save" : "Add"}
              </Button>
            </DialogFooter>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
