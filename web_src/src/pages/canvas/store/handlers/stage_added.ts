import type { Stage } from "../../types";
import type { CanvasState } from "../types";

/**
 * Handler for the stage_added event
 * Manages adding or updating stages based on incoming events
 */
export function handleStageAdded(
  payload: any,
  state: Pick<CanvasState, 'stages' | 'addStage' | 'updateStage'>
): void {
  const stage = payload as Stage;
  console.log('Stage added event received:', stage);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: Stage) => s.id === stage.id);
  if (existingStage) {
    state.updateStage(stage);
  } else {
    state.addStage(stage);
  }
}
