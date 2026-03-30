import type {
  ComponentsComponent,
  ComponentsNode,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";

export function getNodeIntegrationName(
  node: ComponentsNode,
  availableIntegrations: IntegrationsIntegrationDefinition[],
): string | undefined {
  if (node.type === "TYPE_COMPONENT") {
    const match = availableIntegrations.find((integration) =>
      integration.components?.some((component: ComponentsComponent) => component.name === node.component?.name),
    );
    return match?.name;
  }

  if (node.type === "TYPE_TRIGGER") {
    const match = availableIntegrations.find((integration) =>
      integration.triggers?.some((trigger: TriggersTrigger) => trigger.name === node.trigger?.name),
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
