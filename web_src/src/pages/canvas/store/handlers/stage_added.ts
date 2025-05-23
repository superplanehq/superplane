import type { StageWithEventQueue } from "@/canvas/store/types";
import type { CanvasState } from "../types";
import { SuperplaneStage } from "@/api-client";

/**
 * Handler for the stage_added event
 * Manages adding or updating stages based on incoming events
 */
export function handleStageAdded(
  payload: SuperplaneStage,
  state: Pick<CanvasState, 'stages' | 'addStage' | 'updateStage'>
): void {
  console.log('Stage added event received:', payload);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: StageWithEventQueue) => s.id === payload.id);
  if (existingStage) {
    state.updateStage(payload);
  } else {
    state.addStage(payload);
  }
}
