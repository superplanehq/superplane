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

const MISSING_INTEGRATION_ERROR = "This integration does not exist. Choose another integration.";

function integrationById(integrations: OrganizationsIntegration[]): Map<string, OrganizationsIntegration> {
  const map = new Map<string, OrganizationsIntegration>();
  for (const integration of integrations) {
    const id = integration.metadata?.id;
    if (id) {
      map.set(id, integration);
    }
  }
  return map;
}

function integrationOverlayError(integration: OrganizationsIntegration | undefined): string | undefined {
  if (!integration) {
    return MISSING_INTEGRATION_ERROR;
  }

  const state = integration.status?.state;
  if (state === "ready") {
    return undefined;
  }
  if (state === "error") {
    const description = integration.status?.stateDescription;
    return description ? `Integration error: ${description}` : "Integration error";
  }
  return `Integration is ${state ?? "not ready"}`;
}

function withNodeError(canvasNode: CanvasNode, message: string): CanvasNode {
  const data = canvasNode.data as Record<string, unknown>;
  const component = data.component as Record<string, unknown> | undefined;
  const trigger = data.trigger as Record<string, unknown> | undefined;

  if (component && !component.error) {
    return { ...canvasNode, data: { ...data, component: { ...component, error: message } } };
  }
  if (trigger && !trigger.error) {
    return { ...canvasNode, data: { ...data, trigger: { ...trigger, error: message } } };
  }
  return canvasNode;
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
  if (!canvasNodes) {
    return nodes;
  }

  const integrationsById = integrationById(integrations);
  const canvasNodeMap = new Map(canvasNodes.map((node) => [node.id, node]));
  return nodes.map((canvasNode) => {
    const integrationId = canvasNodeMap.get(canvasNode.id)?.integration?.id;
    if (!integrationId) {
      return canvasNode;
    }

    const error = integrationOverlayError(integrationsById.get(integrationId));
    if (!error) {
      return canvasNode;
    }
    return withNodeError(canvasNode, error);
  });
}
