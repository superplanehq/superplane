import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { launchCloudAgentMapper } from "./launchAgent";
import { getDailyUsageMapper } from "./getDailyUsage";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  launchCloudAgent: launchCloudAgentMapper,
  getDailyUsageData: getDailyUsageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  launchCloudAgent: buildActionStateRegistry("completed"),
  getDailyUsageData: buildActionStateRegistry("completed"),
};
