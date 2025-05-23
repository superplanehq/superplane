import type { CanvasState, StageWithEventQueue } from "@/canvas/store/types";
import type { EventWithStage } from "@/canvas/types/events";
import { SuperplaneStageEvent } from "@/api-client";

/**
 * Handler for the stage_updated event
 * Manages updating stages based on incoming events
 */
export function handleStageEventApproved(
  payload: EventWithStage,
  state: Pick<CanvasState, 'stages' | 'updateStage'>
): void {
  const event = payload as EventWithStage;
  console.log('Stage updated - event approved event received:', event);
    // first find the stage
    // then find the event, it is in the queues_by_state array
    // probably in waiting state
    // remove it from the array
    // add the new event to the array
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: StageWithEventQueue) => s.id === event.stage_id);
  if (existingStage) {
    const queue = existingStage.queue.filter((q: SuperplaneStageEvent) => q.id !== event.id);
    queue.push(event);
    const updatedStage = {
      ...existingStage,
      queue
    };
    state.updateStage(updatedStage);
  }
}
