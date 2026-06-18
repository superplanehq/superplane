import type {
  SuperplaneComponentsNode as ComponentsNode,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  IntegrationsCapabilityDefinition,
} from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";

export function getNodeIntegrationName(
  node: ComponentsNode,
  availableIntegrations: IntegrationsIntegrationDefinition[],
): string | undefined {
  if (node.type === "TYPE_ACTION") {
    const match = availableIntegrations.find((integration) =>
      integration.capabilities?.some(
        (capability: IntegrationsCapabilityDefinition) =>
          capability.type === "TYPE_ACTION" && capability.name === node.component,
      ),
    );
    return match?.name;
  }

  if (node.type === "TYPE_TRIGGER") {
    const match = availableIntegrations.find((integration) =>
      integration.capabilities?.some(
        (capability: IntegrationsCapabilityDefinition) =>
          capability.type === "TYPE_TRIGGER" && capability.name === node.component,
      ),
    );
    return match?.name;
  }

  return undefined;
}

function buildNonReadyIntegrationMap(integrations: OrganizationsIntegration[]) {
  const map = new Map<string, { state?: string; description?: string }>();
  for (const integration of integrations) {
    if (integration.metadata?.id && integration.status?.state !== "ready") {
      map.set(integration.metadata.id, {
        state: integration.status?.state,
        description: integration.status?.stateDescription,
      });
    }
  }
  return map;
}

function stripNodeWarnings(
  node: CanvasNode,
  data: Record<string, unknown>,
  field: "component" | "trigger" | "composite",
): CanvasNode {
  const value = data[field];
  if (!value || typeof value !== "object") {
    return node;
  }

  const record = value as Record<string, unknown>;
  const { error: _err, warning: _warn, ...rest } = record;
  return { ...node, data: { ...data, [field]: rest } };
}

export function stripCanvasNodeSetupWarningsForRunsView(nodes: CanvasNode[]): CanvasNode[] {
  return nodes.map((node) => {
    const data = node.data as Record<string, unknown> | undefined;
    if (!data || typeof data !== "object") {
      return node;
    }

    const type = data.type;
    if (type === "component") {
      return stripNodeWarnings(node, data, "component");
    }

    if (type === "trigger") {
      return stripNodeWarnings(node, data, "trigger");
    }

    if (type === "composite") {
      return stripNodeWarnings(node, data, "composite");
    }

    return node;
  });
}

export function overlayIntegrationWarnings(
  nodes: CanvasNode[],
  integrations: OrganizationsIntegration[],
  canvasNodes: ComponentsNode[] | undefined,
): CanvasNode[] {
  if (!integrations.length || !canvasNodes) {
    return nodes;
  }

  const nonReadyIntegrations = buildNonReadyIntegrationMap(integrations);
  if (nonReadyIntegrations.size === 0) {
    return nodes;
  }

  const canvasNodeMap = new Map(canvasNodes.map((node) => [node.id, node]));
  return nodes.map((canvasNode) => {
    const sourceNode = canvasNodeMap.get(canvasNode.id);
    const integrationId = sourceNode?.integration?.id;
    if (!integrationId) {
      return canvasNode;
    }

    const status = nonReadyIntegrations.get(integrationId);
    if (!status) {
      return canvasNode;
    }

    const data = canvasNode.data as Record<string, unknown>;
    const warningMessage =
      status.state === "error"
        ? `Integration error${status.description ? `: ${status.description}` : ""}`
        : `Integration is ${status.state ?? "not ready"}`;

    const component = data.component as Record<string, unknown> | undefined;
    const trigger = data.trigger as Record<string, unknown> | undefined;

    if (component && !component.error) {
      return { ...canvasNode, data: { ...data, component: { ...component, error: warningMessage } } };
    }
    if (trigger && !trigger.error) {
      return { ...canvasNode, data: { ...data, trigger: { ...trigger, error: warningMessage } } };
    }

    return canvasNode;
  });
}
