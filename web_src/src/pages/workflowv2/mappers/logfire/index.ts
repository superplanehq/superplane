import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onAlertReceivedTriggerRenderer } from "./on_alert_received";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryLogfire: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertReceived: onAlertReceivedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queryLogfire: buildActionStateRegistry("completed"),
};
