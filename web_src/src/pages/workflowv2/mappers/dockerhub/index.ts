import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer, CustomFieldRenderer } from "../types";
import { onImagePushTriggerRenderer, onImagePushCustomFieldRenderer } from "./on_image_push";
import { describeImageTagMapper } from "./describe_image_tag";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  describeImageTag: describeImageTagMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onImagePush: onImagePushTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onImagePush: onImagePushCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  describeImageTag: buildActionStateRegistry("retrieved"),
};
