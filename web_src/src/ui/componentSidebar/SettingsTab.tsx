import type {
  AuthorizationDomainType,
  ComponentsIntegrationRef,
  ConfigurationField,
  OrganizationsIntegration,
} from "@/api-client";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { filterVisibleConfiguration, parseDefaultValues } from "@/lib/components";
import { useRealtimeValidation } from "@/hooks/useRealtimeValidation";
import { buildConfigurationDisplayModel } from "./configurationView/buildConfigurationDisplayModel";
import { ConfigurationView } from "./configurationView/ConfigurationView";
import { SettingsTabConfigurationFields } from "./SettingsTabConfigurationFields";
import { SettingsTabIntegrationSection } from "./SettingsTabIntegrationSection";
import {
  FORM_DISABLED_CURSOR_CLASS,
  FORM_DISABLED_SURFACE_CLASS,
  REQUIRED_FIELD_BADGE_CLASS,
  SETTINGS_TAB_DIVIDER_CLASS,
} from "./settingsTabConstants";
import { useSettingsTabAutosave } from "./useSettingsTabAutosave";
import { cn } from "@/lib/utils";

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

export function SettingsTab({
  nodeId,
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
  const resolvedAutocompleteExampleObj = autocompleteExampleObj;

  const defaultValues = useMemo(() => parseDefaultValues(configurationFields), [configurationFields]);

  const defaultValuesWithoutToggles = useMemo(() => {
    const filtered = { ...defaultValues };
    configurationFields.forEach((field) => {
      if (field.name && field.togglable) {
        delete filtered[field.name];
      }
    });
    return filtered;
  }, [configurationFields, defaultValues]);

  const integrationsOfType = useMemo(() => {
    if (!integrationName) return [];
    return integrations.filter((integration) => integration.metadata?.integrationName === integrationName);
  }, [integrations, integrationName]);

  const selectedIntegrationFull = useMemo(() => {
    const id = selectedIntegration?.id ?? integrationRef?.id;
    if (!id) return undefined;
    return integrations.find((integration) => integration.metadata?.id === id);
  }, [integrations, selectedIntegration?.id, integrationRef?.id]);

  const {
    validationErrors: realtimeValidationErrors,
    validateNow,
    hasFieldError: hasRealtimeFieldError,
  } = useRealtimeValidation(
    configurationFields,
    { ...nodeConfiguration, nodeName: currentNodeName },
    {
      debounceMs: 200,
      validateOnMount: false,
    },
  );

  const hasNodeNameError = useMemo(() => {
    return hasRealtimeFieldError("nodeName") || currentNodeName.trim() === "";
  }, [hasRealtimeFieldError, currentNodeName]);

  const filterVisibleFields = useCallback(
    (config: Record<string, unknown>) => filterVisibleConfiguration(config, configurationFields),
    [configurationFields],
  );

  const { requestAutosave, setAutosaveBaseline } = useSettingsTabAutosave({
    currentNodeName,
    initialConfiguration: configuration || {},
    initialIntegrationRef: integrationRef,
    initialNodeName: nodeName,
    isInteractionDisabled,
    nodeConfiguration,
    onSave,
    selectedIntegration,
    validateNow,
  });

  useEffect(() => {
    let newConfig;
    if (Object.values(configuration).length === 0 || !configuration) {
      newConfig = defaultValuesWithoutToggles;
    } else {
      newConfig = { ...defaultValuesWithoutToggles, ...configuration };
    }

    const filteredConfig = filterVisibleFields(newConfig);
    setAutosaveBaseline(filteredConfig, nodeName, integrationRef);
    setNodeConfiguration(filteredConfig);
    setCurrentNodeName(nodeName);
    setSelectedIntegration(integrationRef);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName, defaultValuesWithoutToggles, filterVisibleFields, integrationRef, setAutosaveBaseline]);

  useEffect(() => {
    if (isInteractionDisabled) {
      return;
    }

    if (integrationsOfType.length === 0) {
      if (selectedIntegration) {
        setAutosaveBaseline(nodeConfiguration, currentNodeName, undefined);
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
    setAutosaveBaseline(nodeConfiguration, currentNodeName, nextIntegration);
    setSelectedIntegration({
      id: firstIntegration.metadata?.id,
      name: firstIntegration.metadata?.name,
    });
  }, [
    integrationsOfType,
    isInteractionDisabled,
    selectedIntegration,
    nodeConfiguration,
    currentNodeName,
    setAutosaveBaseline,
  ]);

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

  const customFieldContent = useMemo(() => {
    if (!customField) {
      return null;
    }

    return customField(nodeConfiguration);
  }, [customField, nodeConfiguration]);

  const handleConfigurationFieldChange = useCallback(
    (fieldName: string, value: unknown) => {
      setNodeConfiguration((previousConfiguration) => {
        const newConfig = {
          ...previousConfiguration,
          [fieldName]: value,
        };
        return filterVisibleFields(newConfig);
      });
    },
    [filterVisibleFields],
  );

  if (isReadOnly && !isFormDisabled) {
    return (
      <div className="p-4 pb-24">
        <div className="space-y-6">
          <ConfigurationView model={configurationDisplayModel} />
          {customFieldContent ? (
            <div className={configurationFields.length > 0 ? "" : SETTINGS_TAB_DIVIDER_CLASS}>{customFieldContent}</div>
          ) : null}
        </div>
      </div>
    );
  }

  const runTitleField = configurationFields.find((field) => field.name === "customName");

  return (
    <div
      className="p-4 pb-24 overflow-x-hidden"
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
        className={cn(
          "space-y-6",
          isFormDisabled && FORM_DISABLED_SURFACE_CLASS,
          isFormDisabled && FORM_DISABLED_CURSOR_CLASS,
        )}
        {...(isFormDisabled ? { inert: true } : {})}
      >
        <div className="flex flex-col gap-2">
          <Label className="min-w-[100px] text-left">
            Name
            <span className="text-gray-800 ml-1">*</span>
            {hasNodeNameError ? <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span> : null}
          </Label>
          <Input
            data-testid="node-name-input"
            type="text"
            value={currentNodeName}
            onChange={(event) => {
              setCurrentNodeName(event.target.value);
              requestAutosave();
            }}
            placeholder="Enter a name for this node"
            autoFocus={!isFormDisabled}
            disabled={isFormDisabled}
            className="shadow-none"
          />
        </div>

        {runTitleField ? (
          <div>
            <ConfigurationFieldRenderer
              allowExpressions={true}
              field={runTitleField}
              value={nodeConfiguration[runTitleField.name!]}
              onChange={(value) => {
                handleConfigurationFieldChange(runTitleField.name!, value);
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
        ) : null}

        <SettingsTabIntegrationSection
          allowCreateIntegrations={allowCreateIntegrations}
          allowIntegrations={allowIntegrations}
          allowUpdateIntegrations={allowUpdateIntegrations}
          integrationDefinition={integrationDefinition}
          integrationName={integrationName}
          integrationsOfType={integrationsOfType}
          isFormDisabled={isFormDisabled}
          onOpenConfigureIntegrationDialog={onOpenConfigureIntegrationDialog}
          onOpenCreateIntegrationDialog={onOpenCreateIntegrationDialog}
          onSelectIntegration={setSelectedIntegration}
          requestAutosave={requestAutosave}
          selectedIntegration={selectedIntegration}
          selectedIntegrationFull={selectedIntegrationFull}
          showValidation={showValidation}
          validationErrors={validationErrors}
        />

        <SettingsTabConfigurationFields
          autocompleteExampleObj={resolvedAutocompleteExampleObj}
          configurationFields={configurationFields}
          domainId={domainId}
          domainType={domainType}
          isFormDisabled={isFormDisabled}
          nodeConfiguration={nodeConfiguration}
          nodeId={nodeId}
          nodeName={nodeName}
          onConfigurationChange={handleConfigurationFieldChange}
          realtimeValidationErrors={realtimeValidationErrors}
          requestAutosave={requestAutosave}
          selectedIntegrationId={selectedIntegration?.id}
          showValidation={showValidation}
          validationErrors={validationErrors}
        />
      </div>

      {customFieldContent ? (
        <div className={cn(configurationFields.length > 0 ? "mt-6" : SETTINGS_TAB_DIVIDER_CLASS, "space-y-6")}>
          {customFieldContent}
        </div>
      ) : null}
    </div>
  );
}
