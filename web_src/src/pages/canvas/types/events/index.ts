import { SuperplaneStage, SuperplaneCanvas } from "@/api-client";
import { EventSourceWithEvents } from "../../store/types";

export type ExecutionPayload = { id: string; stage_id: string; canvas_id: string; result: string; timestamp: string }
export type StageEventPayload = { stage_id: string; source_id: string, timestamp: string };

// event_name: payload_type
export type EventMap = {
    stage_added: SuperplaneStage;
    stage_updated: SuperplaneStage;
    event_source_added: EventSourceWithEvents;
    canvas_updated: SuperplaneCanvas;
    new_stage_event: StageEventPayload;
    stage_event_approved: StageEventPayload;
    execution_finished: ExecutionPayload;
    execution_started: ExecutionPayload;
};
  
export type ServerEvent = {
    [K in keyof EventMap]: {
      event: K;
      payload: EventMap[K];
    };
  }[keyof EventMap]; // Discriminated union
