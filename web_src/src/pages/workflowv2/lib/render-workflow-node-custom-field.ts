import type { OrganizationsIntegration, ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "../mappers/types";
import { buildNodeInfo } from "../utils";

export function renderWorkflowNodeCustomField({
  renderer,
  node,
  configuration,
  nodeId,
  context,
}: {
  renderer: CustomFieldRenderer;
  node: ComponentsNode;
  configuration?: Record<string, unknown>;
  nodeId: string;
  context?: {
    onRun?: (initialData?: string) => void;
    integration?: OrganizationsIntegration;
  };
}) {
  const nodeWithConfiguration = {
    ...node,
    configuration: configuration ?? node.configuration,
  };

  try {
    return renderer.render(
      buildNodeInfo(nodeWithConfiguration),
      context && Object.keys(context).length > 0 ? context : undefined,
    );
  } catch (error) {
    console.error(`[CanvasPage] Failed to render custom field for node "${nodeId}":`, error);
    return null;
  }
}
