import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { runAgentMapper } from "./run_agent";
import { runCodeAgentMapper } from "./run_code_agent";
import { getDailyUsageMapper } from "./get_daily_usage";
import { getFileMapper } from "./get_file";
import { downloadFileMapper } from "./download_file";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  runAgent: runAgentMapper,
  runCodeAgent: runCodeAgentMapper,
  getDailyUsage: getDailyUsageMapper,
  getFile: getFileMapper,
  downloadFile: downloadFileMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  runAgent: buildActionStateRegistry("completed"),
  runCodeAgent: buildActionStateRegistry("completed"),
  getDailyUsage: buildActionStateRegistry("completed"),
  getFile: buildActionStateRegistry("fetched"),
  downloadFile: buildActionStateRegistry("downloaded"),
};
