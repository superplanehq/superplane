import { SuperplaneCanvas } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, Stage } from "@/canvas/store/types";

export interface CanvasData {
  canvas: SuperplaneCanvas;
  stages: Stage[];
  eventSources: EventSourceWithEvents[];
  connectionGroups: ConnectionGroupWithEvents[];
}
