import type { BlueprintsBlueprint, CanvasesCanvas, ComponentsComponent, SuperplaneComponentsNode } from "@/api-client";

export type LayoutScope = "full-canvas" | "connected-component";

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: ComponentsComponent[];
  blueprints?: BlueprintsBlueprint[];
};

export interface LayoutEngine {
  estimateNodeSize(node: SuperplaneComponentsNode): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
}
