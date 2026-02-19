import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onRunCompletedTriggerRenderer } from "./on_run_completed";
import { TRIGGER_RUN_STATE_REGISTRY, triggerRunMapper } from "./trigger_run";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerRun: triggerRunMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onRunCompleted: onRunCompletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerRun: TRIGGER_RUN_STATE_REGISTRY,
};
