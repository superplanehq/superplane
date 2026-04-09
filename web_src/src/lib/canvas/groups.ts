import type { ComponentsNode } from "@/api-client";

export function collectGroupChildIds(node: ComponentsNode): string[] {
  if (node.type !== "TYPE_WIDGET" || node.widget?.name !== "group") {
    return [];
  }

  return ((node.configuration?.childNodeIds as string[]) || []).filter(Boolean);
}

export function buildChildToGroupMap(nodes: ComponentsNode[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const node of nodes) {
    if (node.type !== "TYPE_WIDGET" || node.widget?.name !== "group" || !node.id) continue;
    for (const childId of collectGroupChildIds(node)) {
      map.set(childId, node.id);
    }
  }
  return map;
}
