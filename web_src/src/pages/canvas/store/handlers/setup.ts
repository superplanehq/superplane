import { useWebSocketEvent } from "../../hooks/useWebSocket";
import { handleStageAdded } from "./stage_added";
import { handleStageUpdated } from "./stage_updated";
import { handleEventSourceAdded } from "./event_source_added";
import { handleCanvasUpdated } from "./canvas_updated";
import { handleStageEventCreated } from "./stage_event_created";
import { handleStageEventApproved } from "./stage_event_approved";
import { useCanvasStore } from "../canvasStore";

/**
 * Custom React hook that sets up the event handlers for the canvas store
 * Registers listeners for relevant events
 * This must be used within a React component or another custom hook
 */
export function useSetupEventHandlers(canvasId: string): void {
  // Get store from Zustand
  const store = useCanvasStore();
  
  useWebSocketEvent('stage_added', (payload) => {
    handleStageAdded(payload, store);
  }, canvasId);

  useWebSocketEvent('stage_updated', (payload) => {
    handleStageUpdated(payload, store);
  }, canvasId);

  useWebSocketEvent('event_source_added', (payload) => {
    handleEventSourceAdded(payload, store);
  }, canvasId);

  useWebSocketEvent('canvas_updated', (payload) => {
    handleCanvasUpdated(payload, store);
  }, canvasId);

  useWebSocketEvent('new_stage_event', (payload) => {
    handleStageEventCreated(payload, store);
  }, canvasId);

  useWebSocketEvent('stage_event_approved', (payload) => {
    handleStageEventApproved(payload, store);
  }, canvasId);
}
