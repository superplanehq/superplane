import type { CanvasState } from "../types";
import { SuperplaneEventSource } from "@/api-client";

/**
 * Handler for the event_source_added event
 * Manages adding or updating event sources based on incoming events
 */
export function handleEventSourceAdded(
  payload: SuperplaneEventSource,
  state: Pick<CanvasState, 'event_sources' | 'addEventSource' | 'updateEventSource'>
): void {
  console.log('Event source added event received:', payload);
  
  // Check if event source already exists
  const existingSource = state.event_sources.find((es: SuperplaneEventSource) => es.id === payload.id);
  if (existingSource) {
    state.updateEventSource(payload);
  } else {
    state.addEventSource(payload);
  }
}
