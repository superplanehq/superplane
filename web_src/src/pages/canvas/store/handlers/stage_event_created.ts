import type { CanvasState, StageWithEventQueue } from "@/canvas/store/types";
import type { EventWithStage } from "@/canvas/types/events";
import { SuperplaneStageEvent } from "@/api-client";

/**
 * Handler for the stage_event_created event
 * Manages updating stages based on incoming events
 */
export function handleStageEventCreated(
  payload: EventWithStage,
  state: Pick<CanvasState, 'stages' | 'updateStage'>
): void {
  console.log('Stage event created event received:', payload);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: StageWithEventQueue) => s.metadata!.id === payload.stage_id);
  if (existingStage) {
    const queues = existingStage.queue.filter((q: SuperplaneStageEvent) => q.id !== payload.id);
    queues.push(payload);
    const updatedStage = {
      ...existingStage,
      queues
    };
    state.updateStage(updatedStage);
  }
}