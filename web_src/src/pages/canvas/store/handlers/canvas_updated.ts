import type { CanvasState } from "../types";

/**
 * Handler for the canvas_updated event
 * Updates the canvas data with new information
 */
export function handleCanvasUpdated(
  payload: any,
  state: Pick<CanvasState, 'updateCanvas'>
): void {
  console.log('Canvas updated event received:', payload);
  state.updateCanvas(payload);
}
