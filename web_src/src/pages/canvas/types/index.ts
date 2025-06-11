import { SuperplaneCanvas } from "@/api-client";
import { EventSourceWithEvents, StageWithEventQueue } from "@/canvas/store/types";

export interface CanvasData {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  event_sources: EventSourceWithEvents[];
}
