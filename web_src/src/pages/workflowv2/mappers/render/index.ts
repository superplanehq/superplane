import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onEventTriggerRenderer } from "./on_event";
import { triggerDeployMapper } from "./trigger_deploy";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerDeploy: triggerDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onEvent: onEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerDeploy: buildActionStateRegistry("triggered"),
};
