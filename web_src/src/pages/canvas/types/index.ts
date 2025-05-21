import { SuperplaneCanvas, SuperplaneEventSource } from "@/api-client";
import { StageWithEventQueue } from "@/canvas/store/types";

export interface CanvasData {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  event_sources: SuperplaneEventSource[];
}
