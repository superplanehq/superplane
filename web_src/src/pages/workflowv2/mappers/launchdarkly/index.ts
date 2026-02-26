import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onFeatureFlagChangeTriggerRenderer } from "./on_feature_flag_change";
import { getFeatureFlagMapper } from "./get_feature_flag";
import { deleteFeatureFlagMapper } from "./delete_feature_flag";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getFeatureFlag: getFeatureFlagMapper,
  deleteFeatureFlag: deleteFeatureFlagMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onFeatureFlagChange: onFeatureFlagChangeTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getFeatureFlag: buildActionStateRegistry("fetched"),
  deleteFeatureFlag: buildActionStateRegistry("deleted"),
};
