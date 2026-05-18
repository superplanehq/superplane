import type { OrganizationsIntegration, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "../mappers/types";
import { buildNodeInfo } from "../utils";

export function renderCanvasNodeCustomField({
  renderer,
  node,
  configuration,
  context,
  applyConfigurationPatch,
}: {
  renderer: CustomFieldRenderer;
  node: ComponentsNode;
  configuration?: Record<string, unknown>;
  context?: {
    integration?: OrganizationsIntegration;
  };
  applyConfigurationPatch?: (patch: Record<string, unknown>) => void;
}) {
  const nodeWithConfiguration = {
    ...node,
    configuration: configuration ?? node.configuration,
  };

  const mergedContext =
    applyConfigurationPatch || (context && Object.keys(context).length > 0)
      ? { ...(context ?? {}), ...(applyConfigurationPatch ? { applyConfigurationPatch } : {}) }
      : undefined;

  return renderer.render(buildNodeInfo(nodeWithConfiguration), mergedContext);
}
