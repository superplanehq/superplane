import type { SuperplaneStage } from "@/api-client";
import type { CanvasState } from "../types";

/**
 * Handler for the stage_updated event
 * Manages updating stages based on incoming events
 */
export function handleStageUpdated(
  payload: SuperplaneStage,
  state: Pick<CanvasState, 'stages' | 'updateStage'>
): void {
  console.log('Stage updated event received:', payload);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: SuperplaneStage) => s.id === payload.id);
  if (existingStage) {
    state.updateStage(payload);
  }
}
