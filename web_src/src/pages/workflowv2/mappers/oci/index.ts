import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createComputeInstanceMapper } from "./create_compute_instance";
import { onComputeInstanceCreatedTriggerRenderer } from "./on_compute_instance_created";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createComputeInstance: createComputeInstanceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComputeInstanceCreated: onComputeInstanceCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createComputeInstance: buildActionStateRegistry("created"),
};
