import type { CanvasInitialData } from "../../types";

/**
 * Sets up the event handlers for the canvas store
 * Registers listeners for relevant events and returns a cleanup function
 */
export function setupEventHandlers(
  initialData: CanvasInitialData,
): () => void {
  if (!initialData.handleEvent) {
    console.warn("handleEvent not provided to Canvas Store");
    return () => {};
  }

  // const handlerRefs: HandlerRef[] = [];

  // // Register the stage_added event handler
  // const stageAddedRef = initialData.handleEvent('stage_added', (payload) => {
  //   handleStageAdded(payload, state);
  // });
  // handlerRefs.push(stageAddedRef);

  // // Register the stage_updated event handler
  // const stageUpdatedRef = initialData.handleEvent('stage_updated', (payload) => {
  //   handleStageUpdated(payload, state);
  // });
  // handlerRefs.push(stageUpdatedRef);

  // // Register the event_source_added event handler
  // const eventSourceAddedRef = initialData.handleEvent('event_source_added', (payload) => {
  //   handleEventSourceAdded(payload, state);
  // });
  // handlerRefs.push(eventSourceAddedRef);

  // // Register the canvas_updated event handler
  // const canvasUpdatedRef = initialData.handleEvent('canvas_updated', (payload) => {
  //   handleCanvasUpdated(payload, state);
  // });
  // handlerRefs.push(canvasUpdatedRef);

  // const stageEventCreatedRef = initialData.handleEvent('new_stage_event', (payload) => {
  //   handleStageEventCreated(payload, state);
  // });
  // handlerRefs.push(stageEventCreatedRef);

  // const stageEventApprovedRef = initialData.handleEvent('stage_event_approved', (payload) => {
  //   handleStageEventApproved(payload, state);
  // });
  // handlerRefs.push(stageEventApprovedRef);

  // // Return cleanup function to remove all handlers
  // return () => {
  //   if (initialData.removeHandleEvent) {
  //     handlerRefs.forEach(ref => initialData.removeHandleEvent(ref));
  //   }
  // };
  return () => {};
}
