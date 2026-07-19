import type { ComponentsNodeType, ConfigurationParameterRef, SuperplaneComponentsNode } from "@/api-client";

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
