import React, { useEffect, useMemo } from "react";
import { FieldRendererProps } from "./types";
import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { useApplicationResources } from "@/hooks/useApplications";

interface SelectOption {
  id: string;
  label: string;
  value: string;
}

export const MultiSelectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  organizationId,
  appInstallationId,
}) => {
  const multiSelectOptions = field.typeOptions?.multiSelect?.options ?? [];
  const resourceType = field.typeOptions?.resource?.type;
  const useNameAsValue = field.typeOptions?.resource?.useNameAsValue ?? false;

  // Fetch resources if resource type is specified
  const {
    data: resources,
    isLoading: isLoadingResources,
    error: resourcesError,
  } = useApplicationResources(
    organizationId ?? "",
    appInstallationId ?? "",
    resourceType ?? "",
  );

  // Combine static options with dynamic resources
  const comboboxOptions: SelectOption[] = useMemo(() => {
    const staticOptions: SelectOption[] = multiSelectOptions.map((opt) => ({
      id: opt.value!,
      label: opt.label!,
      value: opt.value!,
    }));

    if (!resourceType || !resources || resources.length === 0) {
      return staticOptions;
    }

    // Add resources as options
    const resourceOptions: SelectOption[] = resources
      .map((resource) => {
        const optionValue = useNameAsValue
          ? (resource.name ?? resource.id ?? "")
          : (resource.id ?? resource.name ?? "");
        const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
        if (!optionValue) return null;
        return { id: optionValue, label: optionLabel, value: optionValue };
      })
      .filter((option): option is SelectOption => option !== null);

    // Combine static and resource options, avoiding duplicates
    const allOptions = [...staticOptions];
    for (const resourceOption of resourceOptions) {
      if (!allOptions.some((opt) => opt.value === resourceOption.value)) {
        allOptions.push(resourceOption);
      }
    }

    return allOptions;
  }, [multiSelectOptions, resources, resourceType, useNameAsValue]);

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Array.isArray(field.defaultValue)
        ? field.defaultValue
        : field.defaultValue
          ? JSON.parse(field.defaultValue as string)
          : [];
      if (Array.isArray(defaultVal) && defaultVal.length > 0) {
        onChange(defaultVal);
      }
    }
  }, [value, field.defaultValue, onChange]);

  // Get current selected values
  const currentValue =
    (typeof value !== "string" ? value : JSON.parse(value)) ??
    (field.defaultValue
      ? Array.isArray(field.defaultValue)
        ? field.defaultValue
        : JSON.parse(field.defaultValue as string)
      : []);

  // Convert selected values to SelectOption objects
  const selectedOptions: SelectOption[] = currentValue.map((val: string) => {
    const option = comboboxOptions.find((opt) => opt.value === val);
    return option || { id: val, label: val, value: val };
  });

  const handleChange = (selectedOptions: SelectOption[]) => {
    const selectedValues = selectedOptions.map((opt) => opt.value);
    onChange(selectedValues.length > 0 ? selectedValues : undefined);
  };

  if (isLoadingResources) {
    return (
      <div className="text-sm text-gray-500 dark:text-gray-400">
        Loading {resourceType} resources...
      </div>
    );
  }

  if (resourcesError && resourceType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load {resourceType} resources
      </div>
    );
  }

  return (
    <MultiCombobox<SelectOption>
      options={comboboxOptions}
      displayValue={(option) => option.label}
      placeholder={`Select ${field.label || field.name}...`}
      value={selectedOptions}
      onChange={handleChange}
      showButton={false}
    >
      {(option) => <MultiComboboxLabel>{option.label}</MultiComboboxLabel>}
    </MultiCombobox>
  );
};
