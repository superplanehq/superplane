import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { pushLogsMapper } from "./push_logs";
import { queryLogsMapper } from "./query_logs";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  pushLogs: pushLogsMapper,
  queryLogs: queryLogsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  pushLogs: buildActionStateRegistry("Logs pushed"),
  queryLogs: buildActionStateRegistry("Logs queried"),
};
