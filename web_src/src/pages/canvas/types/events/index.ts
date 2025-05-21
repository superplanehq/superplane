import { SuperplaneStage, SuperplaneEventSource, SuperplaneCanvas, SuperplaneStageEvent } from "@/api-client";

// event_name: payload_type
export type EventMap = {
    stage_added: SuperplaneStage;
    stage_updated: SuperplaneStage;
    event_source_added: SuperplaneEventSource;
    canvas_updated: SuperplaneCanvas;
    new_stage_event: EventWithStage;
    stage_event_approved: EventWithStage;
};
  
export type ServerEvent = {
    [K in keyof EventMap]: {
      event: K;
      payload: EventMap[K];
    };
  }[keyof EventMap]; // Discriminated union
  

export type EventWithStage = SuperplaneStageEvent & { stage_id: string };
