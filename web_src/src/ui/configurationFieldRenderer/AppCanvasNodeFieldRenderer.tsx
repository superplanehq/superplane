import { useEffect, useMemo } from "react";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Select, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";
import { filterAppCanvasNodes, resolveAppCanvasId } from "./appCanvasNodeField";

interface AppCanvasNodeFieldRendererProps {
  field: ConfigurationField;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  allValues?: Record<string, unknown>;
  organizationId?: string;
  readOnly?: boolean;
}

export function AppCanvasNodeFieldRenderer({
  field,
  value,
  onChange,
  allValues,
  organizationId,
  readOnly = false,
}: AppCanvasNodeFieldRendererProps) {
  const typeOptions = field.typeOptions?.appCanvasNode;
  const appCanvasId = useMemo(
    () => resolveAppCanvasId(typeOptions?.parameters, allValues),
    [allValues, typeOptions?.parameters],
  );

  const {
    data: canvas,
    isLoading,
    error,
  } = useCanvas(organizationId ?? "", appCanvasId ?? "", {
    enabled: Boolean(organizationId && appCanvasId),
  });

  const options: AutoCompleteOption[] = useMemo(() => {
    const matchingNodes = filterAppCanvasNodes(
      canvas?.spec?.nodes,
      typeOptions?.nodeTypes,
      typeOptions?.componentTypes,
    );

    return matchingNodes
      .map((node) => ({
        value: node.id!,
        label: node.name?.trim() || node.id!,
      }))
      .sort((left, right) => left.label.localeCompare(right.label));
  }, [canvas?.spec?.nodes, typeOptions?.componentTypes, typeOptions?.nodeTypes]);

  const selectedValue = useMemo(() => {
    if (!value) {
      return "";
    }

    const matchedNode = options.find((option) => option.value === value);
    return matchedNode?.value ?? value;
  }, [options, value]);

  useEffect(() => {
    if (!value || options.length === 0) {
      return;
    }

    const isValid = options.some((option) => option.value === value);
    if (!isValid) {
      onChange(undefined);
    }
  }, [appCanvasId, onChange, options, value]);

  if (!organizationId) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">App canvas node field requires organization context.</div>
    );
  }

  if (!appCanvasId) {
    return (
      <div data-testid={toTestId(`app-canvas-node-field-${field.name}`)} className="space-y-2">
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Select an app first" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">Choose the target app before selecting a node.</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load app nodes: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div data-testid={toTestId(`app-canvas-node-field-${field.name}`)}>
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Loading nodes..." />
          </SelectTrigger>
        </Select>
      </div>
    );
  }

  if (options.length === 0) {
    return (
      <div data-testid={toTestId(`app-canvas-node-field-${field.name}`)} className="space-y-2">
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No matching nodes available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          The selected app has no nodes that match this field&apos;s requirements.
        </p>
      </div>
    );
  }

  return (
    <div data-testid={toTestId(`app-canvas-node-field-${field.name}`)}>
      <AutoCompleteSelect
        options={options}
        value={selectedValue}
        onChange={(nextValue) => onChange(nextValue || undefined)}
        placeholder={field.placeholder ?? "Select node"}
        disabled={readOnly}
      />
    </div>
  );
}
