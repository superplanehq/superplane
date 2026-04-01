import type { ComponentsNode } from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";

export function prepareAnnotationNode(node: ComponentsNode): CanvasNode {
  const width = (node.configuration?.width as number) || 320;
  const height = (node.configuration?.height as number) || 200;
  const position = {
    x: node.position?.x ?? 0,
    y: node.position?.y ?? 0,
  };

  return {
    id: node.id!,
    position,
    selectable: true,
    style: { width, height },
    data: {
      type: "annotation",
      label: node.name || "Annotation",
      state: "pending" as const,
      outputChannels: [],
      annotation: {
        title: node.name || "Annotation",
        annotationText: node.configuration?.text || "",
        annotationColor: node.configuration?.color || "yellow",
        width,
        height,
      },
    },
  };
}
