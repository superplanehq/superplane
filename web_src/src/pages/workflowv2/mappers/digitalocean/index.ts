import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onDropletEventTriggerRenderer } from "./on_droplet_event";
import { createDropletMapper } from "./create_droplet";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDroplet: createDropletMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDropletEvent: onDropletEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDroplet: buildActionStateRegistry("created"),
};
