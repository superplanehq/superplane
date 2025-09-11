import { SuperplaneCanvas } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, Stage } from "../../store/types";

export type ExecutionPayload = { id: string; stage_id: string; canvas_id: string; result: string; timestamp: string }
export type StageEventPayload = { stage_id: string; source_id: string, timestamp: string };
export type EventPayload = { id: string; stage_id: string; source_id: string, source_type: string, timestamp: string };

// event_name: payload_type
export type EventMap = {
    stage_added: Stage;
    connection_group_added: ConnectionGroupWithEvents;
    stage_updated: Stage;
    event_source_added: EventSourceWithEvents;
    canvas_updated: SuperplaneCanvas;
    new_stage_event: StageEventPayload;
    stage_event_approved: StageEventPayload;
    stage_event_discarded: StageEventPayload;
    execution_finished: ExecutionPayload;
    execution_started: ExecutionPayload;
    execution_cancelled: ExecutionPayload;
    event_created: EventPayload;
};
  
export type ServerEvent = {
    [K in keyof EventMap]: {
      event: K;
      payload: EventMap[K];
    };
  }[keyof EventMap]; // Discriminated union
