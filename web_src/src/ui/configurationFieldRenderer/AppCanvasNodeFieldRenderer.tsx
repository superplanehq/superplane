import { useEffect, useMemo } from "react";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Select, SelectTrigger, SelectValue } from "@/components/ui/select";
import type {
  ComponentsNodeType,
  ConfigurationField,
  ConfigurationParameterRef,
  SuperplaneComponentsNode,
} from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";

interface AppCanvasNodeFieldRendererProps {
  field: ConfigurationField;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  allValues?: Record<string, unknown>;
  organizationId?: string;
  readOnly?: boolean;
}

const NODE_TYPE_BY_CONFIG_VALUE: Record<string, ComponentsNodeType> = {
  trigger: "TYPE_TRIGGER",
  action: "TYPE_ACTION",
  widget: "TYPE_WIDGET",
};

export function resolveConfigurationParameterValue(
  parameter: ConfigurationParameterRef,
  allValues?: Record<string, unknown>,
): string | undefined {
  const name = parameter.name?.trim();
  if (!name) {
    return undefined;
  }

  let rawValue: unknown;
  if (parameter.value !== undefined && parameter.value !== "") {
    rawValue = parameter.value;
  } else if (parameter.valueFrom?.field) {
    rawValue = allValues?.[parameter.valueFrom.field];
  } else {
    return undefined;
  }

  if (rawValue === undefined || rawValue === null) {
    return undefined;
  }

  if (typeof rawValue === "string") {
    return rawValue.length > 0 ? rawValue : undefined;
  }

  if (typeof rawValue === "number" || typeof rawValue === "boolean") {
    return String(rawValue);
  }

  return undefined;
}

export function resolveAppCanvasId(
  parameters: ConfigurationParameterRef[] | undefined,
  allValues?: Record<string, unknown>,
): string | undefined {
  if (!parameters?.length) {
    return undefined;
  }

  for (const parameter of parameters) {
    const value = resolveConfigurationParameterValue(parameter, allValues);
    if (value) {
      return value;
    }
  }

  return undefined;
}

export function filterAppCanvasNodes(
  nodes: SuperplaneComponentsNode[] | undefined,
  nodeTypes: string[] | undefined,
  componentTypes: string[] | undefined,
): SuperplaneComponentsNode[] {
  if (!nodes?.length) {
    return [];
  }

  const allowedNodeTypes = normalizeNodeTypes(nodeTypes);
  const allowedComponentTypes = normalizeComponentTypes(componentTypes);

  return nodes.filter((node) => {
    if (!node.id) {
      return false;
    }

    if (allowedNodeTypes && node.type && !allowedNodeTypes.has(node.type)) {
      return false;
    }

    if (allowedComponentTypes && node.component && !allowedComponentTypes.has(node.component)) {
      return false;
    }

    if (allowedComponentTypes && !node.component) {
      return false;
    }

    return true;
  });
}

function normalizeNodeTypes(nodeTypes: string[] | undefined): Set<ComponentsNodeType> | undefined {
  if (!nodeTypes?.length) {
    return undefined;
  }

  const normalized = new Set<ComponentsNodeType>();
  for (const nodeType of nodeTypes) {
    const mapped = NODE_TYPE_BY_CONFIG_VALUE[nodeType] ?? (nodeType as ComponentsNodeType);
    normalized.add(mapped);
  }

  return normalized;
}

function normalizeComponentTypes(componentTypes: string[] | undefined): Set<string> | undefined {
  if (!componentTypes?.length) {
    return undefined;
  }

  return new Set(componentTypes);
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
