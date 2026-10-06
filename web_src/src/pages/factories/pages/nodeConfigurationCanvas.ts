import type { CanvasesCanvas, ComponentsConcurrencySpec, ComponentsIntegrationRef } from "@/api-client";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

export type NodeConfigurationUpdate = {
  nodeId: string;
  name: string;
  configuration: Record<string, unknown>;
  integration?: ComponentsIntegrationRef;
  concurrency?: ComponentsConcurrencySpec;
};

export function canvasHasNode(canvas: CanvasesCanvas, nodeId: string): boolean {
  return (canvas.spec?.nodes ?? []).some((node) => node.id === nodeId);
}

export function applyNodeConfiguration(canvas: CanvasesCanvas, update: NodeConfigurationUpdate): CanvasesCanvas {
  const nodes = (canvas.spec?.nodes ?? []).map((node) => {
    if (node.id !== update.nodeId) {
      return node;
    }
    if (node.type === "TYPE_WIDGET") {
      return {
        ...node,
        name: update.name,
        configuration: { ...node.configuration, ...update.configuration },
      };
    }
    return {
      ...node,
      name: update.name,
      configuration: update.configuration,
      integration: update.integration,
      concurrency: update.concurrency,
    };
  });
  return {
    ...canvas,
    spec: { ...canvas.spec, nodes },
  };
}

export function serializeNodeConfiguration(canvas: CanvasesCanvas, update: NodeConfigurationUpdate): string {
  return materializeCanvasSpec(applyNodeConfiguration(canvas, update));
}
