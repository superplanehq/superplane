import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Select, SelectTrigger, SelectValue } from "../select";
import { ConfigurationField } from "../../api-client";
import { useApplicationResources } from "@/hooks/useApplications";
import { toTestId } from "@/utils/testID";
import { useEffect, useMemo } from "react";

interface AppInstallationResourceFieldRendererProps {
  field: ConfigurationField;
  value: string | string[] | undefined;
  onChange: (value: string | string[] | undefined) => void;
  organizationId?: string;
  appInstallationId?: string;
}

type SelectOption = {
  id: string;
  label: string;
  value: string;
};

export const AppInstallationResourceFieldRenderer = ({
  field,
  value,
  onChange,
  organizationId,
  appInstallationId,
}: AppInstallationResourceFieldRendererProps) => {
  const resourceType = field.typeOptions?.resource?.type;
  const useNameAsValue = field.typeOptions?.resource?.useNameAsValue ?? false;
  // Check for multi - be explicit about truthiness since it's a boolean field
  const isMulti = Boolean(field.typeOptions?.resource?.multi);

  const {
    data: resources,
    isLoading: isLoadingResources,
    error: resourcesError,
  } = useApplicationResources(organizationId ?? "", appInstallationId ?? "", resourceType ?? "");

  // All hooks must be called before any early returns
  // Multi-select options (always compute, even if not used)
  const multiSelectOptions: SelectOption[] = useMemo(() => {
    if (!resources || resources.length === 0) return [];
    return resources
      .map((resource) => {
        const optionValue = useNameAsValue
          ? resource.name ?? resource.id ?? ""
          : resource.id ?? resource.name ?? "";
        const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
        if (!optionValue) return null;
        return { id: optionValue, label: optionLabel, value: optionValue };
      })
      .filter((option): option is SelectOption => option !== null);
  }, [resources, useNameAsValue]);

  // Parse current value - handle both string (JSON) and array formats (always compute)
  const currentValue = useMemo(() => {
    if (value === undefined || value === null) {
      return [];
    }
    if (Array.isArray(value)) {
      return value;
    }
    if (typeof value === "string") {
      try {
        const parsed = JSON.parse(value);
        return Array.isArray(parsed) ? parsed : [];
      } catch {
        return [];
      }
    }
    return [];
  }, [value]);

  // Set initial value on first render if no value is present but there's a default (only for multi-select)
  useEffect(() => {
    if (isMulti && (value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Array.isArray(field.defaultValue)
        ? field.defaultValue
        : field.defaultValue
          ? JSON.parse(field.defaultValue as string)
          : [];
      if (Array.isArray(defaultVal) && defaultVal.length > 0) {
        onChange(defaultVal);
      }
    }
  }, [isMulti, value, field.defaultValue, onChange]);

  // Now we can do early returns
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

  // Single select mode
  if (!isMulti) {
    const options: AutoCompleteOption[] = resources
      .map((resource) => {
        const optionValue = useNameAsValue
          ? resource.name ?? resource.id ?? ""
          : resource.id ?? resource.name ?? "";
        const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
        if (!optionValue) return null;
        return { value: optionValue, label: optionLabel };
      })
      .filter((option): option is AutoCompleteOption => option !== null);

    const selectedValue =
      useNameAsValue && typeof value === "string" && value
        ? resources.find((resource) => resource.id === value)?.name ?? value
        : (typeof value === "string" ? value : "");

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
  }

  // Multi-select mode
  // Convert selected values to SelectOption objects
  const selectedOptions: SelectOption[] = currentValue
    .map((val: string) => {
      const option = multiSelectOptions.find((opt) => opt.value === val);
      return option || { id: val, label: val, value: val };
    })
    .filter((opt) => opt.value !== "");

  const handleChange = (selectedOptions: SelectOption[]) => {
    const selectedValues = selectedOptions.map((opt) => opt.value);
    onChange(selectedValues.length > 0 ? selectedValues : undefined);
  };

  return (
    <div data-testid={toTestId(`app-installation-resource-field-${field.name}`)}>
      <MultiCombobox<SelectOption>
        options={multiSelectOptions}
        displayValue={(option) => option.label}
        placeholder={`Select ${resourceType}...`}
        value={selectedOptions}
        onChange={handleChange}
        showButton={false}
      >
        {(option) => <MultiComboboxLabel>{option.label}</MultiComboboxLabel>}
      </MultiCombobox>
    </div>
  );
};
