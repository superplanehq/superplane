import type { CanvasesCanvas, SuperplaneActionsAction, SuperplaneComponentsNode } from "@/api-client";

export type LayoutScope = "full-canvas" | "connected-component";

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: SuperplaneActionsAction[];
};

export interface LayoutEngine {
  estimateNodeSize(node: SuperplaneComponentsNode): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
}
