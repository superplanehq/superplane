import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Select, SelectTrigger, SelectValue } from "../select";
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
  const useNameAsValue = field.typeOptions?.resource?.useNameAsValue ?? false;

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

  const options: AutoCompleteOption[] = resources
    .map((resource) => {
      const optionValue = useNameAsValue ? (resource.name ?? resource.id ?? "") : (resource.id ?? resource.name ?? "");
      const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
      if (!optionValue) return null;
      return { value: optionValue, label: optionLabel };
    })
    .filter((option): option is AutoCompleteOption => option !== null);

  const selectedValue =
    useNameAsValue && value ? (resources.find((resource) => resource.id === value)?.name ?? value) : (value ?? "");

  return (
    <div data-testid={toTestId(`app-installation-resource-field-${field.name}`)}>
      <AutoCompleteSelect
        options={options}
        value={selectedValue}
        onChange={(val) => onChange(val || undefined)}
        placeholder={`Select ${resourceType}`}
      />
    </div>
  );
};
