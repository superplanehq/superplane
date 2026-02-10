import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { cloudAgentMapper } from "./cloud_agent";
import { getDailyUsageDataMapper } from "./get_daily_usage_data";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  cloudAgent: cloudAgentMapper,
  getDailyUsageData: getDailyUsageDataMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  cloudAgent: buildActionStateRegistry("completed"),
  getDailyUsageData: buildActionStateRegistry("completed"),
};
