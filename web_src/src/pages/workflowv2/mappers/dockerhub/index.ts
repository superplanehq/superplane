import type { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { getImageTagMapper } from "./get_image_tag";
import { onImagePushCustomFieldRenderer, onImagePushTriggerRenderer } from "./on_image_push";
import { onVulnerabilityScanCustomFieldRenderer, onVulnerabilityScanTriggerRenderer } from "./on_vulnerability_scan";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getImageTag: getImageTagMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onImagePush: onImagePushTriggerRenderer,
  onVulnerabilityScan: onVulnerabilityScanTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onImagePush: onImagePushCustomFieldRenderer,
  onVulnerabilityScan: onVulnerabilityScanCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getImageTag: buildActionStateRegistry("retrieved"),
};
