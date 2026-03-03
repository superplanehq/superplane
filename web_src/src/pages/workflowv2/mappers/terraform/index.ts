import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";

import { onRunEventTriggerRenderer } from "./on_event";
import { terraformComponentMapper, TERRAFORM_STATE_REGISTRY } from "./components";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  plan: TERRAFORM_STATE_REGISTRY,
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  plan: terraformComponentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  runEvent: onRunEventTriggerRenderer,
};
