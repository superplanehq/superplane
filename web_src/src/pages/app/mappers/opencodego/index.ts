import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { getUsageMapper } from "./get_usage";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  chatCompletion: baseMapper,
  getUsage: getUsageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  chatCompletion: buildActionStateRegistry("completed"),
  getUsage: buildActionStateRegistry("completed"),
};
