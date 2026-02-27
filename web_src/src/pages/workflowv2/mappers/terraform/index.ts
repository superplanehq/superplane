import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";

import { onRunEventTriggerRenderer, onNeedsAttentionTriggerRenderer } from "./on_event";
import { terraformComponentMapper } from "./components";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queueRun: buildActionStateRegistry("queued"),
  applyRun: buildActionStateRegistry("applied"),
  discardRun: buildActionStateRegistry("discarded"),
  overridePolicy: buildActionStateRegistry("overridden"),
  readRun: buildActionStateRegistry("retrieved"),
  trackRun: buildActionStateRegistry("tracked"),
  waitForApproval: buildActionStateRegistry("decided"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queueRun: terraformComponentMapper,
  applyRun: terraformComponentMapper,
  discardRun: terraformComponentMapper,
  overridePolicy: terraformComponentMapper,
  readRun: terraformComponentMapper,
  trackRun: terraformComponentMapper,
  waitForApproval: terraformComponentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  runEvent: onRunEventTriggerRenderer,
  needsAttention: onNeedsAttentionTriggerRenderer,
};
