import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onImagePushedTriggerRenderer } from "./on_image_pushed";
import { listTagsMapper } from "./list_tags";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  listTags: listTagsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onImagePushed: onImagePushedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listTags: buildActionStateRegistry("retrieved"),
};
