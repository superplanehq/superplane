import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { ConfigurationField } from "../../api-client";
import { useApplicationResources } from "@/hooks/useApplications";
import { toTestId } from "@/utils/testID";

interface AppInstallationResourceFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  organizationId?: string;
  appInstallationId?: string;
}

export const AppInstallationResourceFieldRenderer = ({
  field,
  value,
  onChange,
  organizationId,
  appInstallationId,
}: AppInstallationResourceFieldRendererProps) => {
  const resourceType = field.typeOptions?.resource?.type;

  const {
    data: resources,
    isLoading: isLoadingResources,
    error: resourcesError,
  } = useApplicationResources(organizationId ?? "", appInstallationId ?? "", resourceType ?? "");

  if (!organizationId || !appInstallationId) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        App installation resource field requires organizationId and appInstallationId props
      </div>
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
      <Select disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="No resources available" />
        </SelectTrigger>
      </Select>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={toTestId(`app-installation-resource-field-${field.name}`)}>
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
