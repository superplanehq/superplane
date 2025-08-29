import { SuperplaneCanvas } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, StageWithEventQueue } from "@/canvas/store/types";

export interface CanvasData {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  eventSources: EventSourceWithEvents[];
  connectionGroups: ConnectionGroupWithEvents[];
}
