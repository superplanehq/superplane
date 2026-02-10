import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { getImageTagMapper } from "./get_image_tag";
import { onImagePushTriggerRenderer } from "./on_image_push";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getImageTag: getImageTagMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onImagePush: onImagePushTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getImageTag: buildActionStateRegistry("retrieved"),
};
