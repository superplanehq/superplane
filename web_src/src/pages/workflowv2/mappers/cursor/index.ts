import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { launchAgentMapper } from "./launch_agent";
import { getDailyUsageDataMapper } from "./get_daily_usage_data";
import { getLastMessageMapper } from "./get_last_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  launchAgent: launchAgentMapper,
  getDailyUsageData: getDailyUsageDataMapper,
  getLastMessage: getLastMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  launchAgent: buildActionStateRegistry("completed"),
  getDailyUsageData: buildActionStateRegistry("completed"),
  getLastMessage: buildActionStateRegistry("completed"),
};
