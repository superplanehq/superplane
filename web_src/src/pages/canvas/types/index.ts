import { SuperplaneCanvas, SuperplaneConnectionGroup } from "@/api-client";
import { EventSourceWithEvents, StageWithEventQueue } from "@/canvas/store/types";

export interface CanvasData {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  eventSources: EventSourceWithEvents[];
  connectionGroups: SuperplaneConnectionGroup[];
}
