import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { runCodeAgentMapper } from "./run_code_agent";
import { getDailyUsageDataMapper } from "./get_daily_usage_data";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  runAgent: baseMapper,
  runCodeAgent: runCodeAgentMapper,
  getDailyUsageData: getDailyUsageDataMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  runAgent: buildActionStateRegistry("completed"),
  runCodeAgent: buildActionStateRegistry("completed"),
  getDailyUsageData: buildActionStateRegistry("completed"),
};
