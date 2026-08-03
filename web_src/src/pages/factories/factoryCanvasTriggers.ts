import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";

export function listTriggerNodes(canvas: CanvasesCanvas | undefined): SuperplaneComponentsNode[] {
  return (canvas?.spec?.nodes ?? []).filter((node) => node.type === "TYPE_TRIGGER");
}
