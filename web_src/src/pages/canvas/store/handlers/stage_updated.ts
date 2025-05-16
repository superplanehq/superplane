import type { Stage } from "../../types";
import type { CanvasState } from "../types";

/**
 * Handler for the stage_updated event
 * Manages updating stages based on incoming events
 */
export function handleStageUpdated(
  payload: any,
  state: Pick<CanvasState, 'stages' | 'updateStage'>
): void {
  const stage = payload as Stage;
  console.log('Stage updated event received:', stage);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: Stage) => s.id === stage.id);
  if (existingStage) {
    state.updateStage(stage);
  }
}
