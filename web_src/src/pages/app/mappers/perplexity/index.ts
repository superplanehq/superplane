import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../eventDisplay";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runAgent: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runAgent: buildActionStateRegistry("completed"),
};
