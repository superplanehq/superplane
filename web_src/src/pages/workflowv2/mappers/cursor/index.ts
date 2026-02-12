import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { launchAgentMapper } from "./launch_agent";
import { getDailyUsageDataMapper } from "./get_daily_usage_data";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  launchAgent: launchAgentMapper,
  getDailyUsageData: getDailyUsageDataMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  launchAgent: buildActionStateRegistry("completed"),
  getDailyUsageData: buildActionStateRegistry("completed"),
};
