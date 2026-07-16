import type { AuthorizationDomainType, ConfigurationField } from "@/api-client";
import type { ComponentProps } from "react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { cn } from "@/lib/utils";
import { SETTINGS_TAB_DIVIDER_CLASS } from "./settingsTabConstants";
import { shouldAutosaveOnChangeByFieldType } from "./settingsTabValidation";

type RealtimeValidationErrors = NonNullable<
  ComponentProps<typeof ConfigurationFieldRenderer>["realtimeValidationErrors"]
>;

type SettingsTabConfigurationFieldsProps = {
  autocompleteExampleObj?: Record<string, unknown> | null;
  configurationFields: ConfigurationField[];
  domainId?: string;
  domainType?: AuthorizationDomainType;
  enableRealtimeValidation?: boolean;
  isFormDisabled: boolean;
  nodeConfiguration: Record<string, unknown>;
  nodeId?: string;
  nodeName: string;
  onConfigurationChange: (fieldName: string, value: unknown) => void;
  realtimeValidationErrors: RealtimeValidationErrors;
  requestAutosave: () => void;
  selectedIntegrationId?: string;
  showValidation: boolean;
  validationErrors: Set<string>;
};

export function SettingsTabConfigurationFields({
  autocompleteExampleObj,
  configurationFields,
  domainId,
  domainType,
  enableRealtimeValidation = true,
  isFormDisabled,
  nodeConfiguration,
  nodeId,
  nodeName,
  onConfigurationChange,
  realtimeValidationErrors,
  requestAutosave,
  selectedIntegrationId,
  showValidation,
  validationErrors,
}: SettingsTabConfigurationFieldsProps) {
  if (configurationFields.length === 0) {
    return null;
  }

  return (
    <div className={cn(SETTINGS_TAB_DIVIDER_CLASS, "space-y-4")}>
      {configurationFields.map((field) => {
        if (!field.name || field.name === "customName") return null;
        const fieldName = field.name;

        return (
          <ConfigurationFieldRenderer
            allowExpressions={true}
            key={`${nodeId ?? nodeName}-${fieldName}`}
            field={field}
            value={nodeConfiguration[fieldName]}
            onChange={(value) => {
              onConfigurationChange(fieldName, value);
              const fieldWasCleared = value === undefined || value === null || value === "";
              const previousValue = nodeConfiguration[fieldName];
              const togglableEnabled = field.togglable === true && previousValue == null && !fieldWasCleared;
              if (fieldWasCleared || togglableEnabled || shouldAutosaveOnChangeByFieldType(field.type)) {
                requestAutosave();
              }
            }}
            allValues={nodeConfiguration}
            domainId={domainId}
            domainType={domainType}
            organizationId={domainId}
            integrationId={selectedIntegrationId}
            hasError={
              showValidation &&
              (validationErrors.has(fieldName) ||
                Array.from(validationErrors).some(
                  (error) => error.startsWith(`${fieldName}.`) || error.startsWith(`${fieldName}[`),
                ))
            }
            validationErrors={showValidation ? validationErrors : undefined}
            fieldPath={fieldName}
            realtimeValidationErrors={realtimeValidationErrors}
            enableRealtimeValidation={enableRealtimeValidation}
            autocompleteExampleObj={autocompleteExampleObj}
            readOnly={isFormDisabled}
            preserveEditLayout={isFormDisabled}
          />
        );
      })}
    </div>
  );
}
