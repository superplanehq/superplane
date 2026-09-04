import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onEventTriggerRenderer } from "./on_event";
import { runQueryMapper } from "./run_query";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runQuery: runQueryMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onEvent: onEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runQuery: buildActionStateRegistry("queried"),
};
