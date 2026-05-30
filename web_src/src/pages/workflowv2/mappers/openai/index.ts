import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runCodexAgent: baseMapper,
  textPrompt: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runCodexAgent: buildActionStateRegistry("completed"),
  textPrompt: buildActionStateRegistry("completed"),
};
