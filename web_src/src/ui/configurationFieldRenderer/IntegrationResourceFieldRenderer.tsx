import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { useIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";
import { AuthorizationDomainType, ConfigurationField } from "../../api-client";
import { toTestId } from "@/utils/testID";

interface IntegrationResourceFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  allValues?: Record<string, any>;
  domainId?: string;
  domainType?: AuthorizationDomainType;
}

export const IntegrationResourceFieldRenderer = ({
  field,
  value,
  onChange,
  allValues,
  domainId,
  domainType,
}: IntegrationResourceFieldRendererProps) => {
  const resourceType = field.typeOptions?.resource?.type;

  // Find the integration field by looking at the visibility conditions
  // The integration field should be referenced in the visibility condition
  const integrationFieldName = React.useMemo(() => {
    if (!field.visibilityConditions || field.visibilityConditions.length === 0) {
      return undefined;
    }
    // Find the first visibility condition that references a field (should be the integration field)
    const condition = field.visibilityConditions.find((c) => c.field);
    return condition?.field;
  }, [field.visibilityConditions]);

  // Get the selected integration ID from allValues
  const selectedIntegrationId = integrationFieldName ? allValues?.[integrationFieldName] : undefined;

  // Fetch integrations to get the selected integration details
  const { data: integrations } = useIntegrations(domainId ?? "", domainType ?? "DOMAIN_TYPE_ORGANIZATION");

  // Find the selected integration
  const selectedIntegration = React.useMemo(() => {
    if (!integrations || !selectedIntegrationId) return null;
    return integrations.find((integration) => integration.metadata?.id === selectedIntegrationId);
  }, [integrations, selectedIntegrationId]);

  // Fetch resources using the hook
  const {
    data: resources,
    isLoading: isLoadingResources,
    error: resourcesError,
  } = useIntegrationResources(
    domainId ?? "",
    domainType ?? "DOMAIN_TYPE_ORGANIZATION",
    selectedIntegrationId ?? "",
    resourceType ?? "",
  );

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Integration resource field requires domainId and domainType props
      </div>
    );
  }

  if (!selectedIntegrationId) {
    return (
      <Select disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={`Select ${resourceType} integration first`} />
        </SelectTrigger>
      </Select>
    );
  }

  if (isLoadingResources) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading {resourceType} resources...</div>;
  }

  if (resourcesError) {
    return <div className="text-sm text-red-500 dark:text-red-400">Failed to load resources</div>;
  }

  if (!resources || resources.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No resources available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          No {resourceType} resources found in {selectedIntegration?.metadata?.name}
        </p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={toTestId(`integration-resource-field-${field.name}`)}>
        <SelectValue placeholder={`Select ${resourceType}`} />
      </SelectTrigger>
      <SelectContent>
        {resources.map((resource) => (
          <SelectItem key={resource.id ?? resource.name} value={resource.id ?? resource.name ?? ""}>
            {resource.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
