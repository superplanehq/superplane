import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { runCodeAgentMapper } from "./run_code_agent";
import { getDailyUsageMapper } from "./get_daily_usage";
import { createBatchMessageMapper } from "./create_batch_message";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  runAgent: baseMapper,
  runCodeAgent: runCodeAgentMapper,
  getDailyUsage: getDailyUsageMapper,
  createBatchMessage: createBatchMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  runAgent: buildActionStateRegistry("completed"),
  runCodeAgent: buildActionStateRegistry("completed"),
  getDailyUsage: buildActionStateRegistry("completed"),
  createBatchMessage: buildActionStateRegistry("completed"),
};
