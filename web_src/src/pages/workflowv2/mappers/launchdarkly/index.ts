import { CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { onFeatureFlagChangeTriggerRenderer } from "./on_feature_flag_change";
import { onFeatureFlagChangeCustomFieldRenderer } from "./on_feature_flag_change_webhook";
import { buildActionStateRegistry } from "../utils";

export const componentMappers = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onFeatureFlagChange: onFeatureFlagChangeTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onFeatureFlagChange: onFeatureFlagChangeCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getFeatureFlag: buildActionStateRegistry("fetched"),
  deleteFeatureFlag: buildActionStateRegistry("deleted"),
};
