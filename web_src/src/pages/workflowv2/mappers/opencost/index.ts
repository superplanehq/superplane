import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { getCostAllocationMapper } from "./get_cost_allocation";
import { onCostExceedsThresholdTriggerRenderer } from "./on_cost_exceeds_threshold";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getCostAllocation: getCostAllocationMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onCostExceedsThreshold: onCostExceedsThresholdTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getCostAllocation: buildActionStateRegistry("retrieved"),
};
