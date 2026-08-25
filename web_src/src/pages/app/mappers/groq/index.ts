import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { baseMapper } from "./base";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  chatCompletion: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  chatCompletion: buildActionStateRegistry("completed"),
};
