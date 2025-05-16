import type { EventSource } from "../../types";
import type { CanvasState } from "../types";

/**
 * Handler for the event_source_added event
 * Manages adding or updating event sources based on incoming events
 */
export function handleEventSourceAdded(
  payload: any,
  state: Pick<CanvasState, 'event_sources' | 'addEventSource' | 'updateEventSource'>
): void {
  const eventSource = payload as EventSource;
  console.log('Event source added event received:', eventSource);
  
  // Check if event source already exists
  const existingSource = state.event_sources.find((es: EventSource) => es.id === eventSource.id);
  if (existingSource) {
    state.updateEventSource(eventSource);
  } else {
    state.addEventSource(eventSource);
  }
}
