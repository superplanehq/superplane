import { createPortal } from "react-dom";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Select, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ConfigurationField } from "../../api-client";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { toTestId } from "@/utils/testID";
import { type RefObject, useEffect, useMemo, useState } from "react";

interface IntegrationResourceFieldRendererProps {
  field: ConfigurationField;
  value: string | string[] | undefined;
  onChange: (value: string | string[] | undefined) => void;
  organizationId?: string;
  integrationId?: string;
  allowExpressions?: boolean;
  autocompleteExampleObj?: Record<string, unknown> | null;
  labelRightRef?: RefObject<HTMLDivElement | null>;
  labelRightReady?: boolean;
  /** Current values of sibling fields (e.g. zone); used to filter resources (e.g. dns_record by zone). */
  allValues?: Record<string, unknown>;
}

type SelectOption = {
  id: string;
  label: string;
  value: string;
};

/**
 * Detect if value looks like a wrapped expression (e.g. {{ $["node-name"].value }}).
 * Requires both {{ and }} so fixed IDs (e.g. channel IDs) are not misclassified.
 * Aligns with AutoCompleteInput wrapped expression detection.
 */
function isExpressionValue(value: string | string[] | undefined): boolean {
  if (value == null) return false;
  const str = Array.isArray(value) ? value[0] : value;
  if (typeof str !== "string") return false;
  const trimmed = str.trim();
  if (!trimmed.length) return false;
  return /\{\{[\s\S]*?\}\}/.test(trimmed);
}

export const IntegrationResourceFieldRenderer = ({
  field,
  value,
  onChange,
  organizationId,
  integrationId,
  allowExpressions = false,
  autocompleteExampleObj = null,
  labelRightRef,
  labelRightReady = false,
  allValues = {},
}: IntegrationResourceFieldRendererProps) => {
  const resourceType = field.typeOptions?.resource?.type;
  const useNameAsValue = field.typeOptions?.resource?.useNameAsValue ?? false;
  // Check for multi - be explicit about truthiness since it's a boolean field
  const isMulti = Boolean(field.typeOptions?.resource?.multi);

  // Fixed vs Expression mode for single-select when expressions are allowed
  const initialIsExpression = allowExpressions && !isMulti && isExpressionValue(value);
  const [useExpressionMode, setUseExpressionMode] = useState(initialIsExpression);

  const {
    data: resources,
    isLoading: isLoadingResources,
    error: resourcesError,
  } = useIntegrationResources(organizationId ?? "", integrationId ?? "", resourceType ?? "");

  // Filter resources by zone when this is a dns_record field and zone is selected (Cloudflare record IDs are "zoneId/recordId")
  const filteredResources = useMemo(() => {
    if (!resources || resources.length === 0) return resources ?? [];
    if (resourceType !== "dns_record") return resources;
    const zoneValue = allValues["zone"];
    const zoneId = typeof zoneValue === "string" ? zoneValue.trim() : "";
    if (!zoneId) return [];
    return resources.filter((r) => r.id != null && r.id.startsWith(zoneId + "/"));
  }, [resources, resourceType, allValues]);

  // All hooks must be called before any early returns
  // Multi-select options (always compute, even if not used)
  const multiSelectOptions: SelectOption[] = useMemo(() => {
    if (!filteredResources || filteredResources.length === 0) return [];
    return filteredResources
      .map((resource) => {
        const optionValue = useNameAsValue
          ? (resource.name ?? resource.id ?? "")
          : (resource.id ?? resource.name ?? "");
        const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
        if (!optionValue) return null;
        return { id: optionValue, label: optionLabel, value: optionValue };
      })
      .filter((option): option is SelectOption => option !== null);
  }, [filteredResources, useNameAsValue]);

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
  if (!organizationId || !integrationId) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Integration resource field requires organizationId and integrationId props
      </div>
    );
  }

  if (isLoadingResources) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading {resourceType} resources...</div>;
  }

  if (resourcesError) {
    return <div className="text-sm text-red-500 dark:text-red-400">Failed to load resources</div>;
  }

  if (!resources || resources.length === 0 || filteredResources.length === 0) {
    return (
      <Select disabled>
        <SelectTrigger className="w-full">
          <SelectValue
            placeholder={
              resourceType === "dns_record" && !(typeof allValues["zone"] === "string" && allValues["zone"].trim())
                ? "Select a zone first"
                : "No resources available"
            }
          />
        </SelectTrigger>
      </Select>
    );
  }

  // Single select mode
  if (!isMulti) {
    const options: AutoCompleteOption[] = filteredResources
      .map((resource) => {
        const optionValue = useNameAsValue
          ? (resource.name ?? resource.id ?? "")
          : (resource.id ?? resource.name ?? "");
        const optionLabel = resource.name ?? resource.id ?? "Unnamed resource";
        if (!optionValue) return null;
        return { value: optionValue, label: optionLabel };
      })
      .filter((option): option is AutoCompleteOption => option !== null);

    const selectedValue =
      useNameAsValue && typeof value === "string" && value
        ? (resources.find((r) => r.id === value)?.name ?? value)
        : typeof value === "string"
          ? value
          : "";

    const expressionValue = typeof value === "string" ? value : "";

    const picker = (
      <AutoCompleteSelect
        options={options}
        value={selectedValue}
        onChange={(val) => onChange(val || undefined)}
        placeholder={field.placeholder ?? `Select ${resourceType}`}
      />
    );

    const expressionInput = (
      <AutoCompleteInput
        exampleObj={autocompleteExampleObj}
        value={expressionValue}
        onChange={(nextValue) => onChange(nextValue || undefined)}
        placeholder={field.placeholder ?? `e.g. {{ $["node-name"].value }}`}
        startWord="{{"
        prefix="{{ "
        suffix=" }}"
        inputSize="md"
        showValuePreview
        quickTip="Tip: type {{ to start an expression."
        className=""
      />
    );

    if (allowExpressions) {
      const tabsList = (
        <TabsList className="h-7 rounded-md p-0.5">
          <TabsTrigger value="fixed" className="text-xs px-2 py-1 data-[state=active]:shadow-sm">
            Fixed
          </TabsTrigger>
          <TabsTrigger value="expression" className="text-xs px-2 py-1 data-[state=active]:shadow-sm">
            Expression
          </TabsTrigger>
        </TabsList>
      );
      const tabsInLabelRow =
        labelRightReady && labelRightRef?.current ? createPortal(tabsList, labelRightRef.current) : null;

      const handleTabChange = (v: string) => {
        const nextExpression = v === "expression";
        if (nextExpression !== useExpressionMode) {
          onChange(undefined);
        }
        setUseExpressionMode(nextExpression);
      };

      return (
        <div data-testid={toTestId(`app-installation-resource-field-${field.name}`)} className="space-y-2">
          <Tabs value={useExpressionMode ? "expression" : "fixed"} onValueChange={handleTabChange}>
            {tabsInLabelRow ?? <div className="flex justify-end">{tabsList}</div>}
            <TabsContent value="fixed">{picker}</TabsContent>
            <TabsContent value="expression">{expressionInput}</TabsContent>
          </Tabs>
        </div>
      );
    }

    return <div data-testid={toTestId(`app-installation-resource-field-${field.name}`)}>{picker}</div>;
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
