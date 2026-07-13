import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { getUsageMapper } from "./get_usage";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  getUsage: getUsageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  getUsage: buildActionStateRegistry("completed"),
};
