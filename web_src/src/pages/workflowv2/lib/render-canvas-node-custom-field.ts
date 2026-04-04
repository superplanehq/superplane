import type { OrganizationsIntegration, ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "../mappers/types";
import { buildNodeInfo } from "../utils";

export function renderCanvasNodeCustomField({
  renderer,
  node,
  configuration,
  context,
}: {
  renderer: CustomFieldRenderer;
  node: ComponentsNode;
  configuration?: Record<string, unknown>;
  context?: {
    onRun?: (initialData?: string) => void;
    integration?: OrganizationsIntegration;
  };
}) {
  const nodeWithConfiguration = {
    ...node,
    configuration: configuration ?? node.configuration,
  };

  return renderer.render(
    buildNodeInfo(nodeWithConfiguration),
    context && Object.keys(context).length > 0 ? context : undefined,
  );
}
