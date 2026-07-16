import type { ActionsAction, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

type ComponentMetadata = Pick<ActionsAction, "configuration" | "label">;

export function resolveNodeComponentMetadata(
  node: ComponentsNode,
  allComponentsByName: Map<string, ComponentMetadata>,
  allTriggersByName: Map<string, ComponentMetadata>,
): ComponentMetadata | undefined {
  if (!node.component) {
    return undefined;
  }

  if (node.type === "TYPE_ACTION") {
    return allComponentsByName.get(node.component);
  }

  if (node.type === "TYPE_TRIGGER") {
    return allTriggersByName.get(node.component);
  }

  return undefined;
}
