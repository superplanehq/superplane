import type { BlueprintsBlueprint, CanvasesCanvas, ComponentsComponent, ComponentsNode } from "@/api-client";

export type LayoutScope = "full-canvas" | "connected-component";

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: ComponentsComponent[];
  blueprints?: BlueprintsBlueprint[];
};

export interface LayoutEngine {
  estimateNodeSize(node: ComponentsNode): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
}
