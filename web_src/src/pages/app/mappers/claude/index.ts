import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { runCodeAgentMapper } from "./run_code_agent";
import { getDailyUsageMapper } from "./get_daily_usage";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  runAgent: baseMapper,
  runCodeAgent: runCodeAgentMapper,
  getDailyUsage: getDailyUsageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  runAgent: buildActionStateRegistry("completed"),
  runCodeAgent: buildActionStateRegistry("completed"),
  getDailyUsage: buildActionStateRegistry("completed"),
};
