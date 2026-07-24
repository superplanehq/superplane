import type { CanvasesCanvas, ActionsAction, SuperplaneComponentsNode } from "@/api-client";

export type LayoutScope = "full-canvas" | "connected-component";

/**
 * Orientation of the auto-layout flow.
 *
 * - `horizontal` lays nodes out left-to-right (the classic freeform-friendly
 *   pipeline view).
 * - `vertical` lays nodes out top-to-bottom, matching the vertical auto-layout
 *   canvas view.
 */
export type LayoutDirection = "horizontal" | "vertical";

export const DEFAULT_LAYOUT_DIRECTION: LayoutDirection = "horizontal";

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: ActionsAction[];
  direction?: LayoutDirection;
};

export interface LayoutEngine {
  estimateNodeSize(node: SuperplaneComponentsNode): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
}
