import type { CanvasesCanvas, ActionsAction, SuperplaneComponentsNode } from "@/api-client";
import type { CanvasFlowDirection } from "@/lib/canvasFlowDirection";

export type LayoutScope = "full-canvas" | "connected-component";

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: ActionsAction[];
  /** Defaults to horizontal (LEFT→RIGHT). Factory apps use vertical (TOP→BOTTOM). */
  direction?: CanvasFlowDirection;
};

export interface LayoutEngine {
  estimateNodeSize(node: SuperplaneComponentsNode, direction?: CanvasFlowDirection): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
}
