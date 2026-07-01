import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { runCloudAgentMapper } from "./run_cloud_agent";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
  runAgent: baseMapper,
  runCloudAgent: runCloudAgentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
  runAgent: buildActionStateRegistry("completed"),
  runCloudAgent: buildActionStateRegistry("completed"),
};
