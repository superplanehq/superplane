import type { Stage } from "../../types";
import type { CanvasState } from "../types";
import type { Event } from "@/canvas/types/flow";

/**
 * Handler for the stage_updated event
 * Manages updating stages based on incoming events
 */
export function handleStageEventCreated(
  payload: any,
  state: Pick<CanvasState, 'stages' | 'updateStage'>
): void {
  const event = payload as EventWithStage;
  console.log('Stage updated event received:', event);
  
  // Check if stage already exists
  const existingStage = state.stages.find((s: Stage) => s.id === event.stage_id);
  if (existingStage) {
    let queues = existingStage.queues.filter((q: Event) => q.id !== event.id);
    queues.push(event);
    const updatedStage = {
      ...existingStage,
      queues
    };
    state.updateStage(updatedStage);
  }
}

type EventWithStage = Event & { stage_id: string };

  