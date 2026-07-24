import type { CanvasesCanvas, ActionsAction, SuperplaneComponentsNode } from "@/api-client";

export type LayoutScope = "full-canvas" | "connected-component";

/**
 * Direction in which the layout engine arranges the pipeline.
 * - "RIGHT": nodes flow left-to-right (horizontal, the classic freeform-friendly layout).
 * - "DOWN": nodes flow top-to-bottom (vertical auto-layout view).
 */
export type LayoutDirection = "RIGHT" | "DOWN";

export type LayoutPosition = {
  x: number;
  y: number;
};

export type LayoutEngineApplyOptions = {
  nodeIds?: string[];
  scope?: LayoutScope;
  components?: ActionsAction[];
  direction?: LayoutDirection;
};

export interface LayoutEngine {
  estimateNodeSize(node: SuperplaneComponentsNode): { width: number; height: number };
  apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas>;
  /**
   * Computes engine-controlled node positions without mutating the workflow spec.
   * Useful for a render-only "auto-layout view" overlay that preserves the user's
   * saved freeform positions.
   */
  computeLayoutPositions(
    workflow: CanvasesCanvas,
    options?: LayoutEngineApplyOptions,
  ): Promise<Map<string, LayoutPosition>>;
}
